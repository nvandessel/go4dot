package doctor

import (
	"fmt"
	"strings"
)

// Report generates a human-readable health report
func (r *CheckResult) Report() string {
	var sb strings.Builder

	// Header
	sb.WriteString("╔════════════════════════════════════════╗\n")
	sb.WriteString("║           go4dot Health Check          ║\n")
	sb.WriteString("╚════════════════════════════════════════╝\n\n")

	// Platform info
	if r.Platform != nil {
		sb.WriteString(fmt.Sprintf("Platform: %s", r.Platform.OS))
		if r.Platform.Distro != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", r.Platform.Distro))
		}
		sb.WriteString(fmt.Sprintf(" [%s]\n\n", r.Platform.PackageManager))
	}

	// Main checks
	sb.WriteString("── Health Checks ──\n\n")
	for _, check := range r.Checks {
		icon := statusIcon(check.Status)
		sb.WriteString(fmt.Sprintf("%s %s\n", icon, check.Name))
		sb.WriteString(fmt.Sprintf("  %s\n", check.Message))
		if check.Fix != "" && check.Status != StatusOK {
			sb.WriteString(fmt.Sprintf("  → %s\n", check.Fix))
		}
		sb.WriteString("\n")
	}

	// Summary
	ok, warnings, errors, skipped := r.CountByStatus()
	sb.WriteString("── Summary ──\n\n")

	if errors > 0 {
		sb.WriteString(fmt.Sprintf("✗ %d errors found\n", errors))
	}
	if warnings > 0 {
		sb.WriteString(fmt.Sprintf("⚠ %d warnings\n", warnings))
	}
	if ok > 0 {
		sb.WriteString(fmt.Sprintf("✓ %d checks passed\n", ok))
	}
	if skipped > 0 {
		sb.WriteString(fmt.Sprintf("⊘ %d skipped\n", skipped))
	}

	sb.WriteString("\n")

	// Overall status
	if r.IsHealthy() && !r.HasWarnings() {
		sb.WriteString("Overall: ✓ All systems healthy!\n")
	} else if r.IsHealthy() {
		sb.WriteString("Overall: ⚠ Healthy with warnings\n")
	} else {
		sb.WriteString("Overall: ✗ Issues found - see above for fixes\n")
	}

	return sb.String()
}

// DetailedReport generates a detailed report including individual items
func (r *CheckResult) DetailedReport() string {
	var sb strings.Builder

	// Start with standard report
	sb.WriteString(r.Report())

	// Add detailed symlink status if any have issues
	if len(r.SymlinkStatus) > 0 {
		hasIssues := false
		for _, s := range r.SymlinkStatus {
			if s.Status != StatusOK {
				hasIssues = true
				break
			}
		}

		if hasIssues {
			sb.WriteString("\n── Symlink Details ──\n\n")
			for _, s := range r.SymlinkStatus {
				if s.Status != StatusOK {
					icon := statusIcon(s.Status)
					sb.WriteString(fmt.Sprintf("%s [%s] %s\n", icon, s.Config, s.TargetPath))
					sb.WriteString(fmt.Sprintf("  %s\n", s.Message))
				}
			}
		}
	}

	// Add detailed external status if any are missing
	if len(r.ExternalStatus) > 0 {
		hasMissing := false
		for _, s := range r.ExternalStatus {
			if s.Status == "missing" {
				hasMissing = true
				break
			}
		}

		if hasMissing {
			sb.WriteString("\n── Missing External Dependencies ──\n\n")
			for _, s := range r.ExternalStatus {
				if s.Status == "missing" {
					sb.WriteString(fmt.Sprintf("• %s\n", s.Dep.Name))
					sb.WriteString(fmt.Sprintf("  URL: %s\n", s.Dep.URL))
					sb.WriteString(fmt.Sprintf("  Path: %s\n", s.Dep.Destination))
				}
			}
		}
	}

	// Add detailed machine config status if any are missing
	if len(r.MachineStatus) > 0 {
		hasMissing := false
		for _, s := range r.MachineStatus {
			if s.Status == "missing" {
				hasMissing = true
				break
			}
		}

		if hasMissing {
			sb.WriteString("\n── Missing Machine Configurations ──\n\n")
			for _, s := range r.MachineStatus {
				if s.Status == "missing" {
					sb.WriteString(fmt.Sprintf("• %s\n", s.ID))
					sb.WriteString(fmt.Sprintf("  %s\n", s.Description))
					sb.WriteString(fmt.Sprintf("  Destination: %s\n", s.Destination))
				}
			}
		}
	}

	// Add detailed unmanaged symlinks
	if len(r.UnmanagedLinks) > 0 {
		sb.WriteString("\n── Unmanaged Symlinks ──\n\n")
		sb.WriteString("The following symlinks point to your dotfiles but are not in your config:\n\n")
		for _, l := range r.UnmanagedLinks {
			sb.WriteString(fmt.Sprintf("• %s\n", l.TargetPath))
			sb.WriteString(fmt.Sprintf("  Points to: %s\n", l.SourcePath))
		}
	}

	// Add detailed missing deps if any
	if r.DepsResult != nil {
		missing := r.DepsResult.GetMissing()
		if len(missing) > 0 {
			sb.WriteString("\n── Missing Dependencies ──\n\n")
			for _, dep := range missing {
				sb.WriteString(fmt.Sprintf("• %s\n", dep.Item.Name))
			}
		}
	}

	return sb.String()
}

// QuickReport generates a short one-line status
func (r *CheckResult) QuickReport() string {
	ok, warnings, errors, _ := r.CountByStatus()

	if errors > 0 {
		return fmt.Sprintf("✗ %d errors, %d warnings, %d ok", errors, warnings, ok)
	}
	if warnings > 0 {
		return fmt.Sprintf("⚠ %d warnings, %d ok", warnings, ok)
	}
	return fmt.Sprintf("✓ %d checks passed", ok)
}

// statusIcon returns the icon for a check status
func statusIcon(status CheckStatus) string {
	switch status {
	case StatusOK:
		return "✓"
	case StatusWarning:
		return "⚠"
	case StatusError:
		return "✗"
	case StatusSkipped:
		return "⊘"
	default:
		return "?"
	}
}

// GetFixes returns a list of suggested fixes for issues found
func (r *CheckResult) GetFixes() []string {
	var fixes []string
	seen := make(map[string]bool)

	for _, check := range r.Checks {
		if check.Fix != "" && check.Status != StatusOK && !seen[check.Fix] {
			fixes = append(fixes, check.Fix)
			seen[check.Fix] = true
		}
	}

	return fixes
}

// FixReport generates a report of suggested fixes
func (r *CheckResult) FixReport() string {
	fixes := r.GetFixes()
	if len(fixes) == 0 {
		return "No fixes needed - all checks passed!"
	}

	var sb strings.Builder
	sb.WriteString("Suggested fixes:\n\n")
	for i, fix := range fixes {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, fix))
	}
	return sb.String()
}
