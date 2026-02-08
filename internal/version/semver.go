package version

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer represents a parsed semantic version (major.minor.patch).
type SemVer struct {
	Major int
	Minor int
	Patch int
}

// String returns the version as "major.minor.patch".
func (v SemVer) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// ParseSemVer parses a version string like "1.2.3" or "v1.2.3".
// It handles 1, 2, or 3 component versions (e.g., "1" -> 1.0.0, "1.2" -> 1.2.0).
// Returns an error if the string cannot be parsed.
func ParseSemVer(s string) (SemVer, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")

	if s == "" {
		return SemVer{}, fmt.Errorf("empty version string")
	}

	parts := strings.SplitN(s, ".", 3)
	var v SemVer

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}
	v.Major = major

	if len(parts) >= 2 {
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
		}
		v.Minor = minor
	}

	if len(parts) >= 3 {
		// Strip any pre-release or build metadata (e.g., "1.2.3-beta" -> patch=3)
		patchStr := parts[2]
		if idx := strings.IndexAny(patchStr, "-+"); idx != -1 {
			patchStr = patchStr[:idx]
		}
		patch, err := strconv.Atoi(patchStr)
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid patch version %q: %w", parts[2], err)
		}
		v.Patch = patch
	}

	return v, nil
}

// Compare compares two SemVer values.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func Compare(a, b SemVer) int {
	if a.Major != b.Major {
		if a.Major < b.Major {
			return -1
		}
		return 1
	}
	if a.Minor != b.Minor {
		if a.Minor < b.Minor {
			return -1
		}
		return 1
	}
	if a.Patch != b.Patch {
		if a.Patch < b.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// IsOlderThan returns true if a is strictly older than b.
func (a SemVer) IsOlderThan(b SemVer) bool {
	return Compare(a, b) < 0
}

// IsNewerThan returns true if a is strictly newer than b.
func (a SemVer) IsNewerThan(b SemVer) bool {
	return Compare(a, b) > 0
}

// MajorMismatch returns true when the major versions differ.
// A major version mismatch typically indicates breaking changes.
func MajorMismatch(a, b SemVer) bool {
	return a.Major != b.Major
}
