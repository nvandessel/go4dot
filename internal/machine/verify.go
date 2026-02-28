package machine

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/nvandessel/go4dot/internal/validation"
)

// VerifyStatus constants for verification results.
const (
	VerifyPass = "pass"
	VerifyFail = "fail"
	VerifySkip = "skip"
)

// VerifyResult represents the outcome of a single verification check.
type VerifyResult struct {
	Name    string // "ssh-github", "gpg-sign", "git-user-name", "git-user-email", "git-signing-key"
	Status  string // VerifyPass, VerifyFail, VerifySkip
	Message string
}

// VerifySSHGitHub tests SSH connectivity to GitHub.
// Uses a hardcoded hostname (never user-supplied) with a timeout.
func VerifySSHGitHub(ctx context.Context) VerifyResult {
	result := VerifyResult{Name: "ssh-github"}

	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "ssh", "-T", "-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes", "git@github.com")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	// GitHub returns exit code 1 with "successfully authenticated" in stderr
	stderrStr := stderr.String()
	if strings.Contains(stderrStr, "successfully authenticated") {
		result.Status = VerifyPass
		result.Message = "SSH authentication to GitHub successful"
		return result
	}

	if cmdCtx.Err() != nil {
		result.Status = VerifySkip
		result.Message = "SSH verification timed out"
		return result
	}

	if err != nil {
		result.Status = VerifyFail
		result.Message = fmt.Sprintf("SSH authentication failed: %s", strings.TrimSpace(stderrStr))
		return result
	}

	result.Status = VerifyPass
	result.Message = "SSH connection to GitHub successful"
	return result
}

// VerifyGPGSign tests that a GPG key can sign data.
func VerifyGPGSign(keyID string) VerifyResult {
	result := VerifyResult{Name: "gpg-sign"}

	if err := validation.ValidateGPGKeyID(keyID); err != nil {
		result.Status = VerifyFail
		result.Message = fmt.Sprintf("Invalid GPG key ID: %v", err)
		return result
	}

	cmd := exec.Command("gpg", "--batch", "--no-tty", "--yes", "--sign", "--default-key", keyID, "--output", "/dev/null", "/dev/null")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result.Status = VerifyFail
		result.Message = fmt.Sprintf("GPG signing failed: %s", strings.TrimSpace(stderr.String()))
		return result
	}

	result.Status = VerifyPass
	result.Message = fmt.Sprintf("GPG key %s can sign successfully", keyID)
	return result
}

// VerifyGitConfig checks git user configuration.
func VerifyGitConfig() []VerifyResult {
	var results []VerifyResult

	// Check user.name
	name, _ := GetGitUserName()
	if name != "" {
		results = append(results, VerifyResult{
			Name:    "git-user-name",
			Status:  VerifyPass,
			Message: fmt.Sprintf("user.name = %s", name),
		})
	} else {
		results = append(results, VerifyResult{
			Name:    "git-user-name",
			Status:  VerifyFail,
			Message: "git user.name not configured",
		})
	}

	// Check user.email
	email, _ := GetGitUserEmail()
	if email != "" {
		results = append(results, VerifyResult{
			Name:    "git-user-email",
			Status:  VerifyPass,
			Message: fmt.Sprintf("user.email = %s", email),
		})
	} else {
		results = append(results, VerifyResult{
			Name:    "git-user-email",
			Status:  VerifyFail,
			Message: "git user.email not configured",
		})
	}

	// Check user.signingkey
	signingKey, _ := GetGitSigningKey()
	if signingKey != "" {
		results = append(results, VerifyResult{
			Name:    "git-signing-key",
			Status:  VerifyPass,
			Message: fmt.Sprintf("user.signingkey = %s", signingKey),
		})
	} else {
		results = append(results, VerifyResult{
			Name:    "git-signing-key",
			Status:  VerifySkip,
			Message: "git user.signingkey not configured (optional)",
		})
	}

	return results
}

// RunAllVerifications runs all verification checks and returns aggregated results.
func RunAllVerifications(ctx context.Context, gpgKeyID string) []VerifyResult {
	var results []VerifyResult

	// Git config checks
	results = append(results, VerifyGitConfig()...)

	// SSH GitHub check
	results = append(results, VerifySSHGitHub(ctx))

	// GPG sign check (only if key ID provided)
	if gpgKeyID != "" {
		results = append(results, VerifyGPGSign(gpgKeyID))
	}

	return results
}
