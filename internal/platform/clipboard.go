package platform

import (
	"os"
)

// ClipboardInfo describes the clipboard tooling available on the current platform.
type ClipboardInfo struct {
	CopyCmd    string // Command to copy stdin to clipboard (e.g. "wl-copy")
	PasteCmd   string // Command to paste clipboard to stdout (e.g. "wl-paste")
	Available  bool   // True if clipboard tools are detected
	InstallHint string // Package-manager-specific install suggestion
}

// DetectDisplayServer returns the display server type on Linux:
// "wayland", "x11", or "" if neither is detected.
func DetectDisplayServer() string {
	// Check XDG_SESSION_TYPE first (most reliable)
	if sessionType := os.Getenv("XDG_SESSION_TYPE"); sessionType != "" {
		switch sessionType {
		case "wayland":
			return "wayland"
		case "x11":
			return "x11"
		}
	}

	// Fallback: check for Wayland display
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "wayland"
	}

	// Fallback: check for X11 display
	if os.Getenv("DISPLAY") != "" {
		return "x11"
	}

	return ""
}

// DetectClipboard resolves the clipboard tools available on the given platform.
func DetectClipboard(p *Platform) ClipboardInfo {
	switch p.OS {
	case "darwin":
		return ClipboardInfo{
			CopyCmd:   "pbcopy",
			PasteCmd:  "pbpaste",
			Available: true, // Always available on macOS
		}
	case "linux":
		return detectLinuxClipboard(p)
	default:
		return ClipboardInfo{}
	}
}

func detectLinuxClipboard(p *Platform) ClipboardInfo {
	ds := p.DisplayServer

	switch ds {
	case "wayland":
		if commandExists("wl-copy") && commandExists("wl-paste") {
			return ClipboardInfo{
				CopyCmd:   "wl-copy",
				PasteCmd:  "wl-paste",
				Available: true,
			}
		}
		return ClipboardInfo{
			CopyCmd:     "wl-copy",
			PasteCmd:    "wl-paste",
			Available:   false,
			InstallHint: installHint(p, "wl-clipboard"),
		}
	case "x11":
		if commandExists("xclip") {
			return ClipboardInfo{
				CopyCmd:   "xclip -selection clipboard",
				PasteCmd:  "xclip -selection clipboard -o",
				Available: true,
			}
		}
		if commandExists("xsel") {
			return ClipboardInfo{
				CopyCmd:   "xsel --clipboard --input",
				PasteCmd:  "xsel --clipboard --output",
				Available: true,
			}
		}
		return ClipboardInfo{
			CopyCmd:     "xclip -selection clipboard",
			PasteCmd:    "xclip -selection clipboard -o",
			Available:   false,
			InstallHint: installHint(p, "xclip"),
		}
	default:
		// No display server — headless / TTY
		return ClipboardInfo{}
	}
}

// installHint returns a package-manager-specific install command for the given package.
func installHint(p *Platform, pkg string) string {
	switch p.PackageManager {
	case "apt":
		return "sudo apt install " + pkg
	case "dnf":
		return "sudo dnf install " + pkg
	case "pacman":
		return "sudo pacman -S " + pkg
	case "yum":
		return "sudo yum install " + pkg
	case "zypper":
		return "sudo zypper install " + pkg
	case "apk":
		return "sudo apk add " + pkg
	case "brew":
		return "brew install " + pkg
	default:
		return "install " + pkg
	}
}

// commandExists is declared in packages.go
