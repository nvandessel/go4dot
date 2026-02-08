package version

import (
	"strings"
	"testing"
)

func TestCheckConfigMinVersion(t *testing.T) {
	tests := []struct {
		name        string
		toolVersion string
		minVersion  string
		wantNil     bool   // expect no warning
		wantContain string // substring expected in warning message (if not nil)
	}{
		{
			name:        "tool meets min version exactly",
			toolVersion: "1.2.0",
			minVersion:  "1.2.0",
			wantNil:     true,
		},
		{
			name:        "tool newer than min version",
			toolVersion: "2.0.0",
			minVersion:  "1.5.0",
			wantNil:     true,
		},
		{
			name:        "tool older than min version",
			toolVersion: "1.0.0",
			minVersion:  "1.5.0",
			wantNil:     false,
			wantContain: "requires go4dot 1.5.0 or newer",
		},
		{
			name:        "tool patch older",
			toolVersion: "1.2.3",
			minVersion:  "1.2.4",
			wantNil:     false,
			wantContain: "requires go4dot 1.2.4 or newer",
		},
		{
			name:        "empty tool version",
			toolVersion: "",
			minVersion:  "1.0.0",
			wantNil:     true,
		},
		{
			name:        "empty min version",
			toolVersion: "1.0.0",
			minVersion:  "",
			wantNil:     true,
		},
		{
			name:        "both empty",
			toolVersion: "",
			minVersion:  "",
			wantNil:     true,
		},
		{
			name:        "dev build always compatible",
			toolVersion: "dev",
			minVersion:  "99.0.0",
			wantNil:     true,
		},
		{
			name:        "unknown build always compatible",
			toolVersion: "unknown",
			minVersion:  "99.0.0",
			wantNil:     true,
		},
		{
			name:        "unparseable tool version",
			toolVersion: "not-a-version",
			minVersion:  "1.0.0",
			wantNil:     true,
		},
		{
			name:        "unparseable min version",
			toolVersion: "1.0.0",
			minVersion:  "not-a-version",
			wantNil:     true,
		},
		{
			name:        "v prefix on both",
			toolVersion: "v1.0.0",
			minVersion:  "v1.5.0",
			wantNil:     false,
			wantContain: "requires go4dot 1.5.0 or newer",
		},
		{
			name:        "warning includes upgrade suggestion",
			toolVersion: "1.0.0",
			minVersion:  "2.0.0",
			wantNil:     false,
			wantContain: "upgrade",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckConfigMinVersion(tt.toolVersion, tt.minVersion)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil warning, got: %s", got.String())
				}
				return
			}
			if got == nil {
				t.Fatal("expected warning, got nil")
			}
			full := got.String()
			if !strings.Contains(strings.ToLower(full), strings.ToLower(tt.wantContain)) {
				t.Errorf("warning %q does not contain %q", full, tt.wantContain)
			}
		})
	}
}

func TestCheckStateVersion(t *testing.T) {
	tests := []struct {
		name             string
		toolVersion      string
		stateToolVersion string
		wantNil          bool
		wantContain      string
	}{
		{
			name:             "same version",
			toolVersion:      "1.2.3",
			stateToolVersion: "1.2.3",
			wantNil:          true,
		},
		{
			name:             "tool newer than state",
			toolVersion:      "2.0.0",
			stateToolVersion: "1.5.0",
			wantNil:          true,
		},
		{
			name:             "tool older than state same major",
			toolVersion:      "1.2.0",
			stateToolVersion: "1.5.0",
			wantNil:          false,
			wantContain:      "State file was created by go4dot 1.5.0",
		},
		{
			name:             "tool older than state different major",
			toolVersion:      "1.0.0",
			stateToolVersion: "2.0.0",
			wantNil:          false,
			wantContain:      "Major version mismatch",
		},
		{
			name:             "empty tool version",
			toolVersion:      "",
			stateToolVersion: "1.0.0",
			wantNil:          true,
		},
		{
			name:             "empty state version",
			toolVersion:      "1.0.0",
			stateToolVersion: "",
			wantNil:          true,
		},
		{
			name:             "dev tool version",
			toolVersion:      "dev",
			stateToolVersion: "99.0.0",
			wantNil:          true,
		},
		{
			name:             "unknown tool version",
			toolVersion:      "unknown",
			stateToolVersion: "99.0.0",
			wantNil:          true,
		},
		{
			name:             "dev state version",
			toolVersion:      "1.0.0",
			stateToolVersion: "dev",
			wantNil:          true,
		},
		{
			name:             "unknown state version",
			toolVersion:      "1.0.0",
			stateToolVersion: "unknown",
			wantNil:          true,
		},
		{
			name:             "unparseable tool version",
			toolVersion:      "garbage",
			stateToolVersion: "1.0.0",
			wantNil:          true,
		},
		{
			name:             "unparseable state version",
			toolVersion:      "1.0.0",
			stateToolVersion: "garbage",
			wantNil:          true,
		},
		{
			name:             "minor version mismatch includes consider upgrading",
			toolVersion:      "1.2.0",
			stateToolVersion: "1.3.0",
			wantNil:          false,
			wantContain:      "Consider upgrading",
		},
		{
			name:             "major mismatch includes version numbers",
			toolVersion:      "1.0.0",
			stateToolVersion: "3.0.0",
			wantNil:          false,
			wantContain:      "v1 vs v3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckStateVersion(tt.toolVersion, tt.stateToolVersion)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil warning, got: %s", got.String())
				}
				return
			}
			if got == nil {
				t.Fatal("expected warning, got nil")
			}
			full := got.String()
			if !strings.Contains(full, tt.wantContain) {
				t.Errorf("warning %q does not contain %q", full, tt.wantContain)
			}
		})
	}
}

func TestCompatWarningString(t *testing.T) {
	tests := []struct {
		name       string
		warning    CompatWarning
		wantParts  []string
	}{
		{
			name: "message only",
			warning: CompatWarning{
				Message: "something went wrong",
			},
			wantParts: []string{"something went wrong"},
		},
		{
			name: "message with suggestion",
			warning: CompatWarning{
				Message:    "something went wrong",
				Suggestion: "try this instead",
			},
			wantParts: []string{"something went wrong", "try this instead"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.warning.String()
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("String() = %q, missing %q", got, part)
				}
			}
		})
	}
}
