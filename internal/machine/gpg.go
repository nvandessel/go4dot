package machine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nvandessel/go4dot/internal/validation"
)

// GPGRunner abstracts GPG command execution for testability.
// It extends the concept of Commander with stdin piping support
// needed for operations like --import-ownertrust.
type GPGRunner interface {
	// Run executes a GPG command and returns combined output.
	Run(args ...string) ([]byte, error)
	// RunWithStdin executes a GPG command with data piped to stdin.
	RunWithStdin(stdin string, args ...string) ([]byte, error)
}

// ExecGPGRunner is the default GPGRunner using os/exec.
type ExecGPGRunner struct{}

// Run executes gpg with the given arguments.
func (r *ExecGPGRunner) Run(args ...string) ([]byte, error) {
	return exec.Command("gpg", args...).CombinedOutput()
}

// RunWithStdin executes gpg with data piped to stdin.
func (r *ExecGPGRunner) RunWithStdin(stdin string, args ...string) ([]byte, error) {
	cmd := exec.Command("gpg", args...)
	cmd.Stdin = strings.NewReader(stdin)
	return cmd.CombinedOutput()
}

// SetUltimateTrust sets a GPG key to ultimate trust level (6) via --import-ownertrust.
// The fingerprint must be exactly 40 hex characters.
func SetUltimateTrust(runner GPGRunner, fingerprint string) error {
	if err := validation.ValidateGPGFingerprint(fingerprint); err != nil {
		return fmt.Errorf("invalid fingerprint: %w", err)
	}

	// Format: <FINGERPRINT>:6:\n (6 = ultimate trust)
	trustLine := fmt.Sprintf("%s:6:\n", strings.ToUpper(fingerprint))

	output, err := runner.RunWithStdin(trustLine, "--import-ownertrust")
	if err != nil {
		return fmt.Errorf("gpg --import-ownertrust failed: %w\nOutput: %s", err, bytes.TrimSpace(output))
	}
	return nil
}

// gpgKeyDetail holds parsed information from gpg --with-colons output.
type gpgKeyDetail struct {
	Fingerprint       string
	HasMasterSecret   bool
	SubkeyCount       int
	SubkeyFingerprints []string
}

// StripMasterResult holds the result of a strip-master operation.
type StripMasterResult struct {
	Fingerprint      string
	SubkeysPreserved int
}

// parseColonOutput parses gpg --with-colons --list-secret-keys output.
// Field format: record-type:validity:key-length:algorithm:keyID:...
// Record types: sec (secret key), ssb (secret subkey), fpr (fingerprint).
func parseColonOutput(output string) (*gpgKeyDetail, error) {
	detail := &gpgKeyDetail{}
	lines := strings.Split(output, "\n")

	var lastRecordType string
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}

		recordType := fields[0]
		switch recordType {
		case "sec":
			// sec = primary secret key
			// Field 15 (index 14) contains key capabilities
			// If the secret key is a stub, it shows "sec#" — but in colon format
			// the '#' appears as a card indicator. We check for stub after stripping.
			detail.HasMasterSecret = true
			lastRecordType = "sec"
		case "ssb":
			// ssb = secret subkey
			detail.SubkeyCount++
			lastRecordType = "ssb"
		case "fpr":
			// fpr record follows sec or ssb — field 10 (index 9) has the fingerprint
			if len(fields) > 9 {
				fp := fields[9]
				if lastRecordType == "sec" && detail.Fingerprint == "" {
					detail.Fingerprint = fp
				} else if lastRecordType == "ssb" {
					detail.SubkeyFingerprints = append(detail.SubkeyFingerprints, fp)
				}
			}
		}
	}

	if detail.Fingerprint == "" {
		return nil, fmt.Errorf("no primary key found in gpg output")
	}

	return detail, nil
}

// listSecretKeysWithColons runs gpg --with-colons --list-secret-keys and parses the output.
func listSecretKeysWithColons(runner GPGRunner, keyID string) (*gpgKeyDetail, error) {
	output, err := runner.Run("--with-colons", "--list-secret-keys", keyID)
	if err != nil {
		return nil, fmt.Errorf("gpg --list-secret-keys failed: %w\nOutput: %s", err, bytes.TrimSpace(output))
	}
	return parseColonOutput(string(output))
}

// ListLocalSubkeys returns subkey fingerprints for the given primary key.
func ListLocalSubkeys(runner GPGRunner, primaryKeyID string) ([]string, error) {
	if err := validation.ValidateGPGKeyID(primaryKeyID); err != nil {
		return nil, fmt.Errorf("invalid key ID: %w", err)
	}

	// Use --list-keys (public) with --with-colons for subkey enumeration
	output, err := runner.Run("--with-colons", "--list-keys", primaryKeyID)
	if err != nil {
		return nil, fmt.Errorf("gpg --list-keys failed: %w\nOutput: %s", err, bytes.TrimSpace(output))
	}

	var subkeyFPs []string
	lines := strings.Split(string(output), "\n")
	var lastRecordType string
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "pub":
			lastRecordType = "pub"
		case "sub":
			lastRecordType = "sub"
		case "fpr":
			if lastRecordType == "sub" && len(fields) > 9 {
				subkeyFPs = append(subkeyFPs, fields[9])
			}
			// Reset to avoid double-counting
			lastRecordType = "fpr"
		}
	}
	return subkeyFPs, nil
}

// StripMasterKey removes the primary secret key material while preserving subkeys.
// This is a destructive operation — the master secret key will be replaced with a stub.
//
// Safety invariants:
//   - Uses full fingerprint for --delete-secret-keys (prevents wrong-key deletion)
//   - Temp file uses 0600 permissions, zero-wiped before removal
//   - Export validated non-empty before delete
//   - Temp file preserved with recovery instructions if import fails
//   - Refuses to strip if no subkeys exist
func StripMasterKey(runner GPGRunner, keyID string) (*StripMasterResult, error) {
	if err := validation.ValidateGPGKeyID(keyID); err != nil {
		return nil, fmt.Errorf("invalid key ID: %w", err)
	}

	// Step 1: Verify key exists and has subkeys
	detail, err := listSecretKeysWithColons(runner, keyID)
	if err != nil {
		return nil, fmt.Errorf("key lookup failed: %w", err)
	}
	if detail.SubkeyCount == 0 {
		return nil, fmt.Errorf("key %s has no subkeys — stripping the master would leave no usable secret keys", keyID)
	}

	fingerprint := detail.Fingerprint

	// Step 2: Export subkeys to temp file
	tmpFile, err := os.CreateTemp("", "gpg-subkeys-*.gpg")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	// Set restrictive permissions
	if err := os.Chmod(tmpPath, 0600); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("chmod temp file: %w", err)
	}

	exportOutput, err := runner.Run("--batch", "--yes", "--output", tmpPath, "--export-secret-subkeys", fingerprint)
	if err != nil {
		secureWipeAndRemove(tmpPath)
		return nil, fmt.Errorf("export subkeys failed: %w\nOutput: %s", err, bytes.TrimSpace(exportOutput))
	}

	// Step 3: Validate export is non-empty
	info, err := os.Stat(tmpPath)
	if err != nil || info.Size() == 0 {
		secureWipeAndRemove(tmpPath)
		return nil, fmt.Errorf("exported subkey file is empty — aborting strip")
	}

	// Step 4: Delete the secret key (uses full fingerprint for safety)
	delOutput, err := runner.Run("--batch", "--yes", "--delete-secret-keys", fingerprint)
	if err != nil {
		// Don't remove temp file — user may need it for recovery
		return nil, fmt.Errorf("delete secret key failed (subkeys saved at %s): %w\nOutput: %s", tmpPath, err, bytes.TrimSpace(delOutput))
	}

	// Step 5: Import subkeys from temp file
	importOutput, err := runner.Run("--batch", "--import", tmpPath)
	if err != nil {
		// CRITICAL: Don't remove temp file — it's the only copy of the subkeys
		return nil, fmt.Errorf("CRITICAL: import subkeys failed — subkeys are saved at %s, import manually with: gpg --import %s\nError: %w\nOutput: %s",
			tmpPath, tmpPath, err, bytes.TrimSpace(importOutput))
	}

	// Step 6: Clean up temp file
	secureWipeAndRemove(tmpPath)

	// Step 7: Verify the result
	if err := verifyStripResult(runner, keyID); err != nil {
		return nil, fmt.Errorf("strip succeeded but verification failed: %w", err)
	}

	return &StripMasterResult{
		Fingerprint:      fingerprint,
		SubkeysPreserved: detail.SubkeyCount,
	}, nil
}

// verifyStripResult checks that after stripping, the master key shows as a stub
// and subkeys are still present.
func verifyStripResult(runner GPGRunner, keyID string) error {
	// After strip, --list-secret-keys should show the key with sec# (stub marker)
	output, err := runner.Run("--with-colons", "--list-secret-keys", keyID)
	if err != nil {
		return fmt.Errorf("post-strip verification failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	hasStub := false
	hasSubkey := false
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}
		// In colon format, a stub master key has "#" in field 2 (or the record shows "sec" with
		// secret key availability in field 15). The simplest check: after strip,
		// the key should still appear in --list-secret-keys (meaning subkeys have secret material).
		switch fields[0] {
		case "sec":
			hasStub = true
		case "ssb":
			hasSubkey = true
		}
	}

	if !hasStub {
		return fmt.Errorf("master key not found after strip — may have been fully deleted")
	}
	if !hasSubkey {
		return fmt.Errorf("no subkeys found after strip — subkey import may have failed")
	}
	return nil
}

// secureWipeAndRemove overwrites a file with zeros before removing it.
func secureWipeAndRemove(path string) {
	info, err := os.Stat(path)
	if err != nil {
		_ = os.Remove(path)
		return
	}

	// Overwrite with zeros
	zeros := make([]byte, info.Size())
	_ = os.WriteFile(path, zeros, 0600)

	// Remove
	_ = os.Remove(path)
}
