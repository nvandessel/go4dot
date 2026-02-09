package deps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/validation"
)

// ExternalResult represents the result of cloning external dependencies
type ExternalResult struct {
	Cloned  []config.ExternalDep
	Updated []config.ExternalDep
	Failed  []ExternalError
	Skipped []ExternalSkipped
}

// ExternalError represents a failed clone operation
type ExternalError struct {
	Dep   config.ExternalDep
	Error error
}

// ExternalSkipped represents a skipped external dependency with reason
type ExternalSkipped struct {
	Dep    config.ExternalDep
	Reason string
}

// ExternalOptions configures the clone behavior
type ExternalOptions struct {
	DryRun       bool                                 // Don't actually clone, just report
	Update       bool                                 // Pull updates for existing repos
	RepoRoot     string                               // Path to dotfiles root for @repoRoot expansion
	ProgressFunc func(current, total int, msg string) // Called for progress updates with item counts
}

// CloneExternal clones all external dependencies from the config
func CloneExternal(cfg *config.Config, p *platform.Platform, opts ExternalOptions) (*ExternalResult, error) {
	result := &ExternalResult{}

	if len(cfg.External) == 0 {
		return result, nil
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git is required but not found in PATH")
	}

	total := len(cfg.External)
	for i, ext := range cfg.External {
		current := i + 1

		// Check condition
		if !platform.CheckCondition(ext.Condition, p) {
			result.Skipped = append(result.Skipped, ExternalSkipped{
				Dep:    ext,
				Reason: "condition not met",
			})
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(current, total, fmt.Sprintf("⊘ Skipping %s (condition not met)", ext.Name))
			}
			continue
		}

		// Expand destination path
		destPath, err := expandPath(ext.Destination, opts.RepoRoot)
		if err != nil {
			result.Failed = append(result.Failed, ExternalError{
				Dep:   ext,
				Error: fmt.Errorf("failed to expand path: %w", err),
			})
			continue
		}

		// Check if already exists
		exists, isGit := checkDestination(destPath)

		if exists {
			if ext.Method == "copy" {
				goto Execute
			}

			if opts.Update && isGit {
				// Update existing repo
				if opts.ProgressFunc != nil {
					opts.ProgressFunc(current, total, fmt.Sprintf("↻ Updating %s...", ext.Name))
				}

				if !opts.DryRun {
					if err := gitPull(destPath); err != nil {
						result.Failed = append(result.Failed, ExternalError{
							Dep:   ext,
							Error: fmt.Errorf("failed to update: %w", err),
						})
						continue
					}
				}

				result.Updated = append(result.Updated, ext)
				if opts.ProgressFunc != nil {
					opts.ProgressFunc(current, total, fmt.Sprintf("✓ Updated %s", ext.Name))
				}
			} else {
				// Skip existing
				result.Skipped = append(result.Skipped, ExternalSkipped{
					Dep:    ext,
					Reason: "already exists",
				})
				if opts.ProgressFunc != nil {
					opts.ProgressFunc(current, total, fmt.Sprintf("⊘ Skipping %s (already exists)", ext.Name))
				}
			}
			continue
		}

	Execute:
		// Clone the repository
		if opts.ProgressFunc != nil {
			opts.ProgressFunc(current, total, fmt.Sprintf("⬇ Cloning %s...", ext.Name))
		}

		if opts.DryRun {
			result.Cloned = append(result.Cloned, ext)
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(current, total, fmt.Sprintf("✓ Would clone %s to %s", ext.Name, destPath))
			}
			continue
		}

		// Determine method (clone vs copy)
		method := ext.Method
		if method == "" {
			method = "clone" // Default to clone
		}

		var cloneErr error
		switch method {
		case "clone":
			cloneErr = gitClone(ext.URL, destPath)
		case "copy":
			cloneErr = gitCloneThenCopy(ext.URL, destPath, ext.MergeStrategy)
		default:
			cloneErr = fmt.Errorf("unknown method: %s", method)
		}

		if cloneErr != nil {
			result.Failed = append(result.Failed, ExternalError{
				Dep:   ext,
				Error: cloneErr,
			})
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(current, total, fmt.Sprintf("✗ Failed to clone %s: %v", ext.Name, cloneErr))
			}
		} else {
			result.Cloned = append(result.Cloned, ext)
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(current, total, fmt.Sprintf("✓ Cloned %s", ext.Name))
			}
		}
	}

	return result, nil
}

// CloneSingle clones a single external dependency by ID
func CloneSingle(cfg *config.Config, p *platform.Platform, id string, opts ExternalOptions) error {
	var found *config.ExternalDep
	for i := range cfg.External {
		if cfg.External[i].ID == id {
			found = &cfg.External[i]
			break
		}
	}

	if found == nil {
		return fmt.Errorf("external dependency '%s' not found", id)
	}

	// Check condition
	if !platform.CheckCondition(found.Condition, p) {
		return fmt.Errorf("condition not met for '%s'", id)
	}

	destPath, err := expandPath(found.Destination, opts.RepoRoot)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	exists, isGit := checkDestination(destPath)

	if exists {
		// Special handling for copy method with merge strategy
		if found.Method == "copy" {
			goto Execute
		}

		if opts.Update && isGit {
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(1, 1, fmt.Sprintf("↻ Updating %s...", found.Name))
			}
			if !opts.DryRun {
				if err := gitPull(destPath); err != nil {
					return fmt.Errorf("failed to update: %w", err)
				}
			}
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(1, 1, fmt.Sprintf("✓ Updated %s", found.Name))
			}
			return nil
		}
		return fmt.Errorf("destination already exists: %s", destPath)
	}

Execute:
	if opts.ProgressFunc != nil {
		opts.ProgressFunc(1, 1, fmt.Sprintf("⬇ Cloning %s...", found.Name))
	}

	if opts.DryRun {
		if opts.ProgressFunc != nil {
			opts.ProgressFunc(1, 1, fmt.Sprintf("✓ Would clone %s to %s", found.Name, destPath))
		}
		return nil
	}

	method := found.Method
	if method == "" {
		method = "clone"
	}

	switch method {
	case "clone":
		return gitClone(found.URL, destPath)
	case "copy":
		return gitCloneThenCopy(found.URL, destPath, found.MergeStrategy)
	default:
		return fmt.Errorf("unknown method: %s", method)
	}
}

// CheckExternalStatus returns the status of all external dependencies
func CheckExternalStatus(cfg *config.Config, p *platform.Platform, repoRoot string) []ExternalStatus {
	var statuses []ExternalStatus

	for _, ext := range cfg.External {
		status := ExternalStatus{
			Dep: ext,
		}

		// Check condition
		if !platform.CheckCondition(ext.Condition, p) {
			status.Status = "skipped"
			status.Reason = "condition not met"
			statuses = append(statuses, status)
			continue
		}

		destPath, err := expandPath(ext.Destination, repoRoot)
		if err != nil {
			status.Status = "error"
			status.Reason = fmt.Sprintf("invalid path: %v", err)
			statuses = append(statuses, status)
			continue
		}

		exists, isGit := checkDestination(destPath)
		if exists {
			if isGit {
				status.Status = "installed"
			} else {
				status.Status = "installed"
				if ext.Method == "copy" {
					status.Reason = "copied"
				} else {
					status.Reason = "not a git repo"
				}
			}
		} else {
			status.Status = "missing"
		}

		status.Path = destPath
		statuses = append(statuses, status)
	}

	return statuses
}

// ExternalStatus represents the status of an external dependency
type ExternalStatus struct {
	Dep    config.ExternalDep
	Status string // "installed", "missing", "skipped", "error"
	Reason string
	Path   string
}

// expandPath expands ~ to home directory and resolves @repoRoot.
// It validates that expanded paths stay within their base directory
// and rejects bare absolute paths that don't use ~/ or @repoRoot/ prefixes.
func expandPath(path, repoRoot string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		expanded := filepath.Clean(filepath.Join(home, path[2:]))
		if err := validation.ValidateDestinationPath(expanded, home); err != nil {
			return "", fmt.Errorf("path traversal detected: %w", err)
		}
		return expanded, nil
	} else if strings.HasPrefix(path, "@repoRoot/") {
		if repoRoot == "" {
			return "", fmt.Errorf("repoRoot is not set, cannot expand @repoRoot")
		}
		expanded := filepath.Clean(filepath.Join(repoRoot, path[10:])) // 10 is length of "@repoRoot/"
		if err := validation.ValidateDestinationPath(expanded, repoRoot); err != nil {
			return "", fmt.Errorf("path traversal detected: %w", err)
		}
		return expanded, nil
	}

	// Reject bare absolute paths and any other paths not using ~/ or @repoRoot/
	return "", fmt.Errorf("destination path must start with ~/ or @repoRoot/, got: %q", path)
}

// checkDestination returns whether the path exists and if it's a git repo
func checkDestination(path string) (exists bool, isGit bool) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, false
	}
	if err != nil {
		return false, false
	}
	if !info.IsDir() {
		return true, false
	}

	// Check if it's a git repo
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return true, true
	}
	return true, false
}

// gitClone clones a repository to the destination.
// It validates the URL to prevent flag injection and uses "--" to separate
// git options from the URL operand as defense-in-depth.
func gitClone(url, dest string) error {
	// Validate URL to reject flag injection, file:// scheme, and shell metacharacters
	if err := validation.ValidateGitURL(url); err != nil {
		return fmt.Errorf("invalid git URL: %w", err)
	}

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(dest)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Use "--" to separate options from operands, preventing URL from being
	// interpreted as a git flag (e.g., --upload-pack=malicious).
	cmd := exec.Command("git", "clone", "--depth", "1", "--", url, dest)
	cmd.Stdout = nil // Suppress output
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

// gitPull pulls updates for an existing repository.
// It validates that path is absolute to prevent path traversal attacks.
func gitPull(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("git pull path must be absolute: %q", path)
	}

	cmd := exec.Command("git", "-C", path, "pull", "--ff-only")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	return nil
}

// gitCloneThenCopy clones to a temp directory and copies content (removes .git)
// This is useful for dependencies where you want to own the files
func gitCloneThenCopy(url, dest, mergeStrategy string) error {
	// Create a temp directory for cloning
	tmpDir, err := os.MkdirTemp("", "go4dot-clone-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Clone to temp
	tmpDest := filepath.Join(tmpDir, "repo")
	if err := gitClone(url, tmpDest); err != nil {
		return err
	}

	// Remove .git directory
	gitDir := filepath.Join(tmpDest, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("failed to remove .git: %w", err)
	}

	// Create parent directory of destination
	parentDir := filepath.Dir(dest)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Try to copy (copyDir handles merge strategy)
	return copyDir(tmpDest, dest, mergeStrategy)
}

// copyDir recursively copies a directory
func copyDir(src, dst, mergeStrategy string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath, mergeStrategy); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath, mergeStrategy); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst, mergeStrategy string) error {
	// Check merge strategy
	if mergeStrategy == "keep_existing" {
		if _, err := os.Stat(dst); err == nil {
			// File exists, skip
			return nil
		}
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	buf := make([]byte, 32*1024)
	for {
		n, err := srcFile.Read(buf)
		if n > 0 {
			if _, err := dstFile.Write(buf[:n]); err != nil {
				return err
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
	}

	return nil
}

// RemoveExternal removes an external dependency by ID
func RemoveExternal(cfg *config.Config, id string, opts ExternalOptions) error {
	var found *config.ExternalDep
	for i := range cfg.External {
		if cfg.External[i].ID == id {
			found = &cfg.External[i]
			break
		}
	}

	if found == nil {
		return fmt.Errorf("external dependency '%s' not found", id)
	}

	destPath, err := expandPath(found.Destination, opts.RepoRoot)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return fmt.Errorf("'%s' is not installed (path does not exist: %s)", id, destPath)
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(1, 1, fmt.Sprintf("Removing %s...", found.Name))
	}

	if opts.DryRun {
		if opts.ProgressFunc != nil {
			opts.ProgressFunc(1, 1, fmt.Sprintf("✓ Would remove %s from %s", found.Name, destPath))
		}
		return nil
	}

	if err := os.RemoveAll(destPath); err != nil {
		return fmt.Errorf("failed to remove %s: %w", destPath, err)
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(1, 1, fmt.Sprintf("✓ Removed %s", found.Name))
	}

	return nil
}
