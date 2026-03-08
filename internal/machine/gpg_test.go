package machine

import (
	"fmt"
	"strings"
	"testing"
)

// mockGPGRunner records calls and returns configured responses.
// For multi-step operations, use runResponses to queue sequential responses.
type mockGPGRunner struct {
	runCalls      [][]string
	runStdinCalls []struct {
		stdin string
		args  []string
	}

	// Single-response mode (used by SetUltimateTrust tests)
	runOutput      []byte
	runErr         error
	runStdinOutput []byte
	runStdinErr    error

	// Multi-response mode (used by StripMasterKey tests)
	// Each Run() call pops the next response. Falls back to runOutput/runErr when exhausted.
	runResponses []mockResponse
	runCallIdx   int
}

type mockResponse struct {
	output []byte
	err    error
}

func (m *mockGPGRunner) Run(args ...string) ([]byte, error) {
	m.runCalls = append(m.runCalls, args)
	if m.runCallIdx < len(m.runResponses) {
		resp := m.runResponses[m.runCallIdx]
		m.runCallIdx++
		return resp.output, resp.err
	}
	return m.runOutput, m.runErr
}

func (m *mockGPGRunner) RunWithStdin(stdin string, args ...string) ([]byte, error) {
	m.runStdinCalls = append(m.runStdinCalls, struct {
		stdin string
		args  []string
	}{stdin, args})
	return m.runStdinOutput, m.runStdinErr
}

func TestSetUltimateTrust(t *testing.T) {
	validFP := "ABCDEF0123456789ABCDEF0123456789ABCDEF01"

	tests := []struct {
		name        string
		fingerprint string
		runErr      error
		runOutput   []byte
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid fingerprint succeeds",
			fingerprint: validFP,
			wantErr:     false,
		},
		{
			name:        "lowercase fingerprint succeeds",
			fingerprint: strings.ToLower(validFP),
			wantErr:     false,
		},
		{
			name:        "empty fingerprint rejected",
			fingerprint: "",
			wantErr:     true,
			errContains: "invalid fingerprint",
		},
		{
			name:        "short key ID rejected",
			fingerprint: "ABCD1234",
			wantErr:     true,
			errContains: "invalid fingerprint",
		},
		{
			name:        "41 chars rejected",
			fingerprint: validFP + "A",
			wantErr:     true,
			errContains: "invalid fingerprint",
		},
		{
			name:        "non-hex rejected",
			fingerprint: "GHIJKLMN0123456789ABCDEF0123456789ABCDEF",
			wantErr:     true,
			errContains: "invalid fingerprint",
		},
		{
			name:        "gpg command failure",
			fingerprint: validFP,
			runErr:      fmt.Errorf("gpg process exited with code 2"),
			runOutput:   []byte("gpg: error"),
			wantErr:     true,
			errContains: "gpg --import-ownertrust failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockGPGRunner{
				runStdinOutput: tt.runOutput,
				runStdinErr:    tt.runErr,
			}

			err := SetUltimateTrust(mock, tt.fingerprint)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetUltimateTrust() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if !tt.wantErr {
				if len(mock.runStdinCalls) != 1 {
					t.Fatalf("expected 1 RunWithStdin call, got %d", len(mock.runStdinCalls))
				}
				call := mock.runStdinCalls[0]
				expectedStdin := strings.ToUpper(tt.fingerprint) + ":6:\n"
				if call.stdin != expectedStdin {
					t.Errorf("stdin = %q, want %q", call.stdin, expectedStdin)
				}
				if len(call.args) != 1 || call.args[0] != "--import-ownertrust" {
					t.Errorf("args = %v, want [--import-ownertrust]", call.args)
				}
			}
		})
	}
}

func TestParseColonOutput(t *testing.T) {
	tests := []struct {
		name              string
		output            string
		wantFingerprint   string
		wantHasMaster     bool
		wantSubkeyCount   int
		wantSubkeyFPs     []string
		wantErr           bool
	}{
		{
			name: "key with master and two subkeys",
			output: `sec::4096:1:AABBCCDD11223344:1609459200:::u:::scESC:::+::ed25519:::0:
fpr:::::::::ABCDEF0123456789ABCDEF0123456789ABCDEF01:
uid:::::::::::User Name <user@example.com>::::::::::0:
ssb::4096:1:1111222233334444:1609459200::::::::e:::+::cv25519::
fpr:::::::::1111222233334444AABBCCDD1111222233334444:
ssb::4096:1:5555666677778888:1609459200::::::::s:::+::ed25519::
fpr:::::::::5555666677778888AABBCCDD5555666677778888:
`,
			wantFingerprint: "ABCDEF0123456789ABCDEF0123456789ABCDEF01",
			wantHasMaster:   true,
			wantSubkeyCount: 2,
			wantSubkeyFPs: []string{
				"1111222233334444AABBCCDD1111222233334444",
				"5555666677778888AABBCCDD5555666677778888",
			},
		},
		{
			name: "key with master only, no subkeys",
			output: `sec::4096:1:AABBCCDD11223344:1609459200:::u:::scESC:::+::ed25519:::0:
fpr:::::::::ABCDEF0123456789ABCDEF0123456789ABCDEF01:
uid:::::::::::User Name <user@example.com>::::::::::0:
`,
			wantFingerprint: "ABCDEF0123456789ABCDEF0123456789ABCDEF01",
			wantHasMaster:   true,
			wantSubkeyCount: 0,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:    "garbage output",
			output:  "not:valid:gpg:output\nfoo:bar\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail, err := parseColonOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseColonOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if detail.Fingerprint != tt.wantFingerprint {
				t.Errorf("Fingerprint = %q, want %q", detail.Fingerprint, tt.wantFingerprint)
			}
			if detail.HasMasterSecret != tt.wantHasMaster {
				t.Errorf("HasMasterSecret = %v, want %v", detail.HasMasterSecret, tt.wantHasMaster)
			}
			if detail.SubkeyCount != tt.wantSubkeyCount {
				t.Errorf("SubkeyCount = %d, want %d", detail.SubkeyCount, tt.wantSubkeyCount)
			}
			if tt.wantSubkeyFPs != nil {
				if len(detail.SubkeyFingerprints) != len(tt.wantSubkeyFPs) {
					t.Errorf("SubkeyFingerprints count = %d, want %d", len(detail.SubkeyFingerprints), len(tt.wantSubkeyFPs))
				} else {
					for i, fp := range detail.SubkeyFingerprints {
						if fp != tt.wantSubkeyFPs[i] {
							t.Errorf("SubkeyFingerprints[%d] = %q, want %q", i, fp, tt.wantSubkeyFPs[i])
						}
					}
				}
			}
		})
	}
}

func TestStripMasterKey_InvalidKeyID(t *testing.T) {
	mock := &mockGPGRunner{}
	_, err := StripMasterKey(mock, "")
	if err == nil {
		t.Error("expected error for empty key ID")
	}
	if !strings.Contains(err.Error(), "invalid key ID") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStripMasterKey_NoSubkeys(t *testing.T) {
	// gpg --list-secret-keys returns a key with no subkeys
	colonOutput := `sec::4096:1:AABBCCDD11223344:1609459200:::u:::scESC:::+::ed25519:::0:
fpr:::::::::ABCDEF0123456789ABCDEF0123456789ABCDEF01:
uid:::::::::::User Name <user@example.com>::::::::::0:
`
	mock := &mockGPGRunner{
		runResponses: []mockResponse{
			{output: []byte(colonOutput)}, // listSecretKeysWithColons
		},
	}

	_, err := StripMasterKey(mock, "ABCDEF0123456789")
	if err == nil {
		t.Error("expected error when no subkeys exist")
	}
	if !strings.Contains(err.Error(), "no subkeys") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStripMasterKey_KeyLookupFails(t *testing.T) {
	mock := &mockGPGRunner{
		runResponses: []mockResponse{
			{output: []byte("gpg: error"), err: fmt.Errorf("exit 2")},
		},
	}

	_, err := StripMasterKey(mock, "ABCDEF0123456789")
	if err == nil {
		t.Error("expected error when key lookup fails")
	}
	if !strings.Contains(err.Error(), "key lookup failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListLocalSubkeys(t *testing.T) {
	tests := []struct {
		name      string
		keyID     string
		output    string
		runErr    error
		wantFPs   []string
		wantErr   bool
	}{
		{
			name:  "two subkeys",
			keyID: "ABCDEF0123456789",
			output: `pub:-:4096:1:AABBCCDD11223344:1609459200:::-:::scESC:::+::ed25519:::0:
fpr:::::::::ABCDEF0123456789ABCDEF0123456789ABCDEF01:
uid:::::::::::User Name <user@example.com>::::::::::0:
sub:-:4096:1:1111222233334444:1609459200::::::::e:::+::cv25519::
fpr:::::::::1111222233334444AABBCCDD1111222233334444:
sub:-:4096:1:5555666677778888:1609459200::::::::s:::+::ed25519::
fpr:::::::::5555666677778888AABBCCDD5555666677778888:
`,
			wantFPs: []string{
				"1111222233334444AABBCCDD1111222233334444",
				"5555666677778888AABBCCDD5555666677778888",
			},
		},
		{
			name:    "no subkeys",
			keyID:   "ABCDEF0123456789",
			output:  "pub:-:4096:1:AABBCCDD11223344:1609459200:::-:::scESC:::+::ed25519:::0:\nfpr:::::::::ABCDEF0123456789ABCDEF0123456789ABCDEF01:\n",
			wantFPs: nil,
		},
		{
			name:    "invalid key ID",
			keyID:   "",
			wantErr: true,
		},
		{
			name:    "gpg failure",
			keyID:   "ABCDEF0123456789",
			runErr:  fmt.Errorf("exit 2"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockGPGRunner{
				runOutput: []byte(tt.output),
				runErr:    tt.runErr,
			}

			fps, err := ListLocalSubkeys(mock, tt.keyID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListLocalSubkeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(fps) != len(tt.wantFPs) {
				t.Errorf("got %d subkeys, want %d", len(fps), len(tt.wantFPs))
				return
			}
			for i, fp := range fps {
				if fp != tt.wantFPs[i] {
					t.Errorf("subkey[%d] = %q, want %q", i, fp, tt.wantFPs[i])
				}
			}
		})
	}
}
