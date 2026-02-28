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
		fmt.Fprintf(&sb, "Platform: %s", r.Platform.OS)
		if r.Platform.Distro != "" {
			fmt.Fprintf(&sb, " (%s)", r.Platform.Distro)
		}
		fmt.Fprintf(&sb, " [%s]\n\n", r.Platform.PackageManager)
	}

	// Main checks
	sb.WriteString("── Health Checks ──\n\n")
	for _, check := range r.Checks {
		icon := statusIcon(check.Status)
		fmt.Fprintf(&sb, "%s %s\n", icon, check.Name)
		fmt.Fprintf(&sb, "  %s\n", check.Message)
		if check.Fix != "" && check.Status != StatusOK {
			fmt.Fprintf(&sb, "  → %s\n", check.Fix)
		}
		sb.WriteString("\n")
	}

	// Summary
	ok, warnings, errors, skipped := r.CountByStatus()
	sb.WriteString("── Summary ──\n\n")

	if errors > 0 {
		fmt.Fprintf(&sb, "✗ %d errors found\n", errors)
	}
	if warnings > 0 {
		fmt.Fprintf(&sb, "⚠ %d warnings\n", warnings)
	}
	if ok > 0 {
		fmt.Fprintf(&sb, "✓ %d checks passed\n", ok)
	}
	if skipped > 0 {
		fmt.Fprintf(&sb, "⊘ %d skipped\n", skipped)
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
					fmt.Fprintf(&sb, "%s [%s] %s\n", icon, s.Config, s.TargetPath)
					fmt.Fprintf(&sb, "  %s\n", s.Message)
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
					fmt.Fprintf(&sb, "• %s\n", s.Dep.Name)
					fmt.Fprintf(&sb, "  URL: %s\n", s.Dep.URL)
					fmt.Fprintf(&sb, "  Path: %s\n", s.Dep.Destination)
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
					fmt.Fprintf(&sb, "• %s\n", s.ID)
					fmt.Fprintf(&sb, "  %s\n", s.Description)
					fmt.Fprintf(&sb, "  Destination: %s\n", s.Destination)
				}
			}
		}
	}

	// Add detailed unmanaged symlinks
	if len(r.UnmanagedLinks) > 0 {
		sb.WriteString("\n── Unmanaged Symlinks ──\n\n")
		sb.WriteString("The following symlinks point to your dotfiles but are not in your config:\n\n")
		for _, l := range r.UnmanagedLinks {
			fmt.Fprintf(&sb, "• %s\n", l.TargetPath)
			fmt.Fprintf(&sb, "  Points to: %s\n", l.SourcePath)
		}
	}

	// Add detailed missing deps if any
	if r.DepsResult != nil {
		missing := r.DepsResult.GetMissing()
		if len(missing) > 0 {
			sb.WriteString("\n── Missing Dependencies ──\n\n")
			for _, dep := range missing {
				fmt.Fprintf(&sb, "• %s\n", dep.Item.Name)
			}
		}

		manualMissing := r.DepsResult.GetManualMissing()
		if len(manualMissing) > 0 {
			sb.WriteString("\n── Manual Dependencies (not installed) ──\n\n")
			sb.WriteString("These must be installed manually (e.g., proprietary software, AUR, built from source):\n\n")
			for _, dep := range manualMissing {
				fmt.Fprintf(&sb, "• %s (manual)\n", dep.Item.Name)
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
		fmt.Fprintf(&sb, "%d. %s\n", i+1, fix)
	}
	return sb.String()
}
