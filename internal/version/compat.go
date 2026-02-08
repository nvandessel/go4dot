package version

import "fmt"

// CompatWarning represents a version compatibility warning.
// Warnings are non-fatal: the tool continues running but informs the user.
type CompatWarning struct {
	Message    string
	Suggestion string
}

// String returns the warning formatted for display.
func (w CompatWarning) String() string {
	if w.Suggestion != "" {
		return fmt.Sprintf("%s\n  %s", w.Message, w.Suggestion)
	}
	return w.Message
}

// CheckConfigMinVersion checks whether the current tool version satisfies the
// config's min_version requirement. Returns a warning if the tool is too old,
// or nil if everything is fine.
//
// If either version string is empty or unparseable (e.g., "dev"), no warning
// is returned -- we assume development builds are always compatible.
func CheckConfigMinVersion(toolVersion, minVersion string) *CompatWarning {
	if toolVersion == "" || minVersion == "" {
		return nil
	}

	// Dev/unknown builds are always considered compatible
	if toolVersion == "dev" || toolVersion == "unknown" {
		return nil
	}

	tool, err := ParseSemVer(toolVersion)
	if err != nil {
		return nil
	}

	min, err := ParseSemVer(minVersion)
	if err != nil {
		return nil
	}

	if tool.IsOlderThan(min) {
		return &CompatWarning{
			Message: fmt.Sprintf(
				"WARNING: This config requires go4dot %s or newer, but you are running %s.",
				min.String(), tool.String(),
			),
			Suggestion: "Please upgrade go4dot: https://github.com/nvandessel/go4dot/releases",
		}
	}

	return nil
}

// CheckStateVersion checks whether a state file was created by a newer version
// of go4dot than is currently running. Returns a warning if there is a potential
// incompatibility, or nil if everything is fine.
//
// The stateVersion parameter is the tool version that last wrote the state file
// (stored in state.json's "tool_version" field), not the state format version.
func CheckStateVersion(toolVersion, stateToolVersion string) *CompatWarning {
	if toolVersion == "" || stateToolVersion == "" {
		return nil
	}

	// Dev/unknown builds skip compatibility checks
	if toolVersion == "dev" || toolVersion == "unknown" {
		return nil
	}
	if stateToolVersion == "dev" || stateToolVersion == "unknown" {
		return nil
	}

	tool, err := ParseSemVer(toolVersion)
	if err != nil {
		return nil
	}

	stateTool, err := ParseSemVer(stateToolVersion)
	if err != nil {
		return nil
	}

	// State was created by a newer version
	if tool.IsOlderThan(stateTool) {
		w := &CompatWarning{
			Message: fmt.Sprintf(
				"WARNING: State file was created by go4dot %s, but you are running %s.",
				stateTool.String(), tool.String(),
			),
		}

		if MajorMismatch(tool, stateTool) {
			w.Suggestion = fmt.Sprintf(
				"Major version mismatch detected (v%d vs v%d). Some features may not work correctly. Please upgrade: https://github.com/nvandessel/go4dot/releases",
				tool.Major, stateTool.Major,
			)
		} else {
			w.Suggestion = "The state file may contain fields not recognized by this version. Consider upgrading: https://github.com/nvandessel/go4dot/releases"
		}

		return w
	}

	return nil
}
