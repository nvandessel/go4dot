package deps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/validation"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	repoRoot := "/tmp/repo"

	tests := []struct {
		name     string
		input    string
		repoRoot string
		expected string
	}{
		{
			name:     "Home directory expansion",
			input:    "~/.config/test",
			repoRoot: "",
			expected: filepath.Join(home, ".config/test"),
		},
		{
			name:     "Absolute path unchanged",
			input:    "/usr/local/bin",
			repoRoot: "",
			expected: "/usr/local/bin",
		},
		{
			name:     "Relative path cleaned",
			input:    "./foo/../bar",
			repoRoot: "",
			expected: "bar",
		},
		{
			name:     "Home only",
			input:    "~/",
			repoRoot: "",
			expected: home,
		},
		{
			name:     "RepoRoot expansion",
			input:    "@repoRoot/config",
			repoRoot: repoRoot,
			expected: filepath.Join(repoRoot, "config"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandPath(tt.input, tt.repoRoot)
			if err != nil {
				t.Fatalf("expandPath() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCheckCondition(t *testing.T) {
	// Create test platform
	linuxPlatform := &platform.Platform{
		OS:             "linux",
		Distro:         "fedora",
		DistroVersion:  "43",
		IsWSL:          false,
		PackageManager: "dnf",
		Architecture:   "amd64",
	}

	darwinPlatform := &platform.Platform{
		OS:             "darwin",
		Distro:         "",
		DistroVersion:  "",
		IsWSL:          false,
		PackageManager: "brew",
		Architecture:   "arm64",
	}

	wslPlatform := &platform.Platform{
		OS:             "linux",
		Distro:         "ubuntu",
		DistroVersion:  "22.04",
		IsWSL:          true,
		PackageManager: "apt",
		Architecture:   "amd64",
	}

	tests := []struct {
		name      string
		condition map[string]string
		platform  *platform.Platform
		want      bool
	}{
		{
			name:      "No condition always matches",
			condition: nil,
			platform:  linuxPlatform,
			want:      true,
		},
		{
			name:      "Empty condition always matches",
			condition: map[string]string{},
			platform:  linuxPlatform,
			want:      true,
		},
		{
			name:      "OS match",
			condition: map[string]string{"os": "linux"},
			platform:  linuxPlatform,
			want:      true,
		},
		{
			name:      "OS no match",
			condition: map[string]string{"os": "darwin"},
			platform:  linuxPlatform,
			want:      false,
		},
		{
			name:      "Platform alias for OS",
			condition: map[string]string{"platform": "linux"},
			platform:  linuxPlatform,
			want:      true,
		},
		{
			name:      "Distro match",
			condition: map[string]string{"distro": "fedora"},
			platform:  linuxPlatform,
			want:      true,
		},
		{
			name:      "Distro no match",
			condition: map[string]string{"distro": "ubuntu"},
			platform:  linuxPlatform,
			want:      false,
		},
		{
			name:      "Package manager match",
			condition: map[string]string{"package_manager": "dnf"},
			platform:  linuxPlatform,
			want:      true,
		},
		{
			name:      "WSL true match",
			condition: map[string]string{"wsl": "true"},
			platform:  wslPlatform,
			want:      true,
		},
		{
			name:      "WSL true no match",
			condition: map[string]string{"wsl": "true"},
			platform:  linuxPlatform,
			want:      false,
		},
		{
			name:      "WSL false match",
			condition: map[string]string{"wsl": "false"},
			platform:  linuxPlatform,
			want:      true,
		},
		{
			name:      "Architecture match",
			condition: map[string]string{"arch": "arm64"},
			platform:  darwinPlatform,
			want:      true,
		},
		{
			name:      "Multiple OS comma separated",
			condition: map[string]string{"os": "linux,darwin"},
			platform:  linuxPlatform,
			want:      true,
		},
		{
			name:      "Multiple conditions all match",
			condition: map[string]string{"os": "linux", "distro": "fedora"},
			platform:  linuxPlatform,
			want:      true,
		},
		{
			name:      "Multiple conditions one fails",
			condition: map[string]string{"os": "linux", "distro": "ubuntu"},
			platform:  linuxPlatform,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := platform.CheckCondition(tt.condition, tt.platform)
			if got != tt.want {
				t.Errorf("CheckCondition(%v) = %v, want %v", tt.condition, got, tt.want)
			}
		})
	}
}

func TestMatchesValue(t *testing.T) {
	// This is now tested in platform package, but we can keep a simple test here if needed
	// or just remove it. Since it's internal to platform now, we'll remove it from here.
}

func TestCheckDestination(t *testing.T) {
	// Create temp directories for testing
	tmpDir := t.TempDir()

	// Create a regular directory
	regularDir := filepath.Join(tmpDir, "regular")
	if err := os.MkdirAll(regularDir, 0755); err != nil {
		t.Fatalf("Failed to create regular dir: %v", err)
	}

	// Create a git directory
	gitDir := filepath.Join(tmpDir, "gitrepo")
	if err := os.MkdirAll(filepath.Join(gitDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create git dir: %v", err)
	}

	// Create a file (not directory)
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	tests := []struct {
		name       string
		path       string
		wantExists bool
		wantIsGit  bool
	}{
		{
			name:       "Non-existent path",
			path:       filepath.Join(tmpDir, "nonexistent"),
			wantExists: false,
			wantIsGit:  false,
		},
		{
			name:       "Regular directory",
			path:       regularDir,
			wantExists: true,
			wantIsGit:  false,
		},
		{
			name:       "Git repository",
			path:       gitDir,
			wantExists: true,
			wantIsGit:  true,
		},
		{
			name:       "File (not directory)",
			path:       filePath,
			wantExists: true,
			wantIsGit:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, isGit := checkDestination(tt.path)
			if exists != tt.wantExists {
				t.Errorf("checkDestination(%q) exists = %v, want %v", tt.path, exists, tt.wantExists)
			}
			if isGit != tt.wantIsGit {
				t.Errorf("checkDestination(%q) isGit = %v, want %v", tt.path, isGit, tt.wantIsGit)
			}
		})
	}
}

func TestCheckExternalStatus(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create an installed external dep (git repo)
	installedPath := filepath.Join(tmpDir, "installed")
	if err := os.MkdirAll(filepath.Join(installedPath, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create installed dir: %v", err)
	}

	cfg := &config.Config{
		External: []config.ExternalDep{
			{
				ID:          "installed",
				Name:        "Installed Dep",
				URL:         "https://github.com/example/repo.git",
				Destination: installedPath,
			},
			{
				ID:          "missing",
				Name:        "Missing Dep",
				URL:         "https://github.com/example/missing.git",
				Destination: filepath.Join(tmpDir, "nonexistent"),
			},
			{
				ID:          "skipped",
				Name:        "Skipped Dep",
				URL:         "https://github.com/example/skipped.git",
				Destination: filepath.Join(tmpDir, "skipped"),
				Condition:   map[string]string{"os": "windows"}, // Will not match
			},
		},
	}

	p := &platform.Platform{
		OS:             "linux",
		Distro:         "fedora",
		PackageManager: "dnf",
	}

	statuses := CheckExternalStatus(cfg, p, "")

	if len(statuses) != 3 {
		t.Fatalf("len(statuses) = %d, want 3", len(statuses))
	}

	// Check installed status
	var installedStatus, missingStatus, skippedStatus *ExternalStatus
	for i := range statuses {
		switch statuses[i].Dep.ID {
		case "installed":
			installedStatus = &statuses[i]
		case "missing":
			missingStatus = &statuses[i]
		case "skipped":
			skippedStatus = &statuses[i]
		}
	}

	if installedStatus == nil || installedStatus.Status != "installed" {
		t.Errorf("installed dep status = %v, want 'installed'", installedStatus)
	}

	if missingStatus == nil || missingStatus.Status != "missing" {
		t.Errorf("missing dep status = %v, want 'missing'", missingStatus)
	}

	if skippedStatus == nil || skippedStatus.Status != "skipped" {
		t.Errorf("skipped dep status = %v, want 'skipped'", skippedStatus)
	}
}

func TestCloneExternalDryRun(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		External: []config.ExternalDep{
			{
				ID:          "test1",
				Name:        "Test Repo 1",
				URL:         "https://github.com/example/repo1.git",
				Destination: filepath.Join(tmpDir, "repo1"),
			},
			{
				ID:          "test2",
				Name:        "Test Repo 2",
				URL:         "https://github.com/example/repo2.git",
				Destination: filepath.Join(tmpDir, "repo2"),
				Condition:   map[string]string{"os": "windows"}, // Will be skipped
			},
		},
	}

	p := &platform.Platform{
		OS:             "linux",
		Distro:         "fedora",
		PackageManager: "dnf",
	}

	var progressMessages []string
	opts := ExternalOptions{
		DryRun: true,
		ProgressFunc: func(current, total int, msg string) {
			progressMessages = append(progressMessages, msg)
		},
	}

	result, err := CloneExternal(cfg, p, opts)
	if err != nil {
		t.Fatalf("CloneExternal() error = %v", err)
	}

	// In dry run, nothing should actually be cloned
	if len(result.Cloned) != 1 {
		t.Errorf("len(Cloned) = %d, want 1", len(result.Cloned))
	}

	if len(result.Skipped) != 1 {
		t.Errorf("len(Skipped) = %d, want 1", len(result.Skipped))
	}

	// Check that destination does not exist (dry run)
	if _, err := os.Stat(filepath.Join(tmpDir, "repo1")); !os.IsNotExist(err) {
		t.Error("repo1 should not exist after dry run")
	}

	// Check progress messages
	if len(progressMessages) == 0 {
		t.Error("Expected progress messages")
	}
}

func TestCloneExternalSkipsExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an existing directory
	existingPath := filepath.Join(tmpDir, "existing")
	if err := os.MkdirAll(existingPath, 0755); err != nil {
		t.Fatalf("Failed to create existing dir: %v", err)
	}

	cfg := &config.Config{
		External: []config.ExternalDep{
			{
				ID:          "existing",
				Name:        "Existing Repo",
				URL:         "https://github.com/example/existing.git",
				Destination: existingPath,
			},
		},
	}

	p := &platform.Platform{
		OS:             "linux",
		Distro:         "fedora",
		PackageManager: "dnf",
	}

	result, err := CloneExternal(cfg, p, ExternalOptions{})
	if err != nil {
		t.Fatalf("CloneExternal() error = %v", err)
	}

	if len(result.Skipped) != 1 {
		t.Errorf("len(Skipped) = %d, want 1", len(result.Skipped))
	}

	if len(result.Cloned) != 0 {
		t.Errorf("len(Cloned) = %d, want 0", len(result.Cloned))
	}

	if result.Skipped[0].Reason != "already exists" {
		t.Errorf("Skipped reason = %q, want 'already exists'", result.Skipped[0].Reason)
	}
}

func TestCloneSingleNotFound(t *testing.T) {
	cfg := &config.Config{
		External: []config.ExternalDep{
			{
				ID:          "test",
				Name:        "Test Repo",
				URL:         "https://github.com/example/test.git",
				Destination: "/tmp/test",
			},
		},
	}

	p := &platform.Platform{
		OS: "linux",
	}

	err := CloneSingle(cfg, p, "nonexistent", ExternalOptions{})
	if err == nil {
		t.Error("Expected error for nonexistent ID")
	}

	if err.Error() != "external dependency 'nonexistent' not found" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestRemoveExternalNotFound(t *testing.T) {
	cfg := &config.Config{
		External: []config.ExternalDep{
			{
				ID:          "test",
				Name:        "Test Repo",
				URL:         "https://github.com/example/test.git",
				Destination: "/tmp/test",
			},
		},
	}

	err := RemoveExternal(cfg, "nonexistent", ExternalOptions{})
	if err == nil {
		t.Error("Expected error for nonexistent ID")
	}
}

func TestRemoveExternalDryRun(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory to remove
	toRemove := filepath.Join(tmpDir, "toremove")
	if err := os.MkdirAll(toRemove, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	cfg := &config.Config{
		External: []config.ExternalDep{
			{
				ID:          "toremove",
				Name:        "To Remove",
				URL:         "https://github.com/example/toremove.git",
				Destination: toRemove,
			},
		},
	}

	var progressMessages []string
	opts := ExternalOptions{
		DryRun: true,
		ProgressFunc: func(current, total int, msg string) {
			progressMessages = append(progressMessages, msg)
		},
	}

	err := RemoveExternal(cfg, "toremove", opts)
	if err != nil {
		t.Fatalf("RemoveExternal() error = %v", err)
	}

	// Directory should still exist (dry run)
	if _, err := os.Stat(toRemove); os.IsNotExist(err) {
		t.Error("Directory should still exist after dry run")
	}

	if len(progressMessages) == 0 {
		t.Error("Expected progress messages")
	}
}

func TestRemoveExternal(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory to remove
	toRemove := filepath.Join(tmpDir, "toremove")
	if err := os.MkdirAll(toRemove, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	cfg := &config.Config{
		External: []config.ExternalDep{
			{
				ID:          "toremove",
				Name:        "To Remove",
				URL:         "https://github.com/example/toremove.git",
				Destination: toRemove,
			},
		},
	}

	err := RemoveExternal(cfg, "toremove", ExternalOptions{})
	if err != nil {
		t.Fatalf("RemoveExternal() error = %v", err)
	}

	// Directory should be removed
	if _, err := os.Stat(toRemove); !os.IsNotExist(err) {
		t.Error("Directory should be removed")
	}
}

func TestCopyDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source directory with files
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Copy to destination
	dstDir := filepath.Join(tmpDir, "dst")
	if err := copyDir(srcDir, dstDir, ""); err != nil {
		t.Fatalf("copyDir() error = %v", err)
	}

	// Verify copied files
	content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if err != nil {
		t.Fatalf("Failed to read copied file1: %v", err)
	}
	if string(content1) != "content1" {
		t.Errorf("file1 content = %q, want 'content1'", content1)
	}

	content2, err := os.ReadFile(filepath.Join(dstDir, "subdir", "file2.txt"))
	if err != nil {
		t.Fatalf("Failed to read copied file2: %v", err)
	}
	if string(content2) != "content2" {
		t.Errorf("file2 content = %q, want 'content2'", content2)
	}
}

func TestCopyDirKeepExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source directory
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}
	// New file in source
	if err := os.WriteFile(filepath.Join(srcDir, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}
	// Conflicting file in source
	if err := os.WriteFile(filepath.Join(srcDir, "conflict.txt"), []byte("source_conflict"), 0644); err != nil {
		t.Fatalf("Failed to create source conflict file: %v", err)
	}

	// Create destination directory
	dstDir := filepath.Join(tmpDir, "dst")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatalf("Failed to create dst dir: %v", err)
	}
	// Existing conflicting file in destination
	if err := os.WriteFile(filepath.Join(dstDir, "conflict.txt"), []byte("dest_conflict"), 0644); err != nil {
		t.Fatalf("Failed to create dest conflict file: %v", err)
	}

	// Copy with keep_existing
	if err := copyDir(srcDir, dstDir, "keep_existing"); err != nil {
		t.Fatalf("copyDir() error = %v", err)
	}

	// Verify "new.txt" was copied
	newContent, err := os.ReadFile(filepath.Join(dstDir, "new.txt"))
	if err != nil {
		t.Fatalf("Failed to read new.txt: %v", err)
	}
	if string(newContent) != "new" {
		t.Errorf("new.txt content = %q, want 'new'", newContent)
	}

	// Verify "conflict.txt" was NOT overwritten
	conflictContent, err := os.ReadFile(filepath.Join(dstDir, "conflict.txt"))
	if err != nil {
		t.Fatalf("Failed to read conflict.txt: %v", err)
	}
	if string(conflictContent) != "dest_conflict" {
		t.Errorf("conflict.txt content = %q, want 'dest_conflict' (should have been preserved)", conflictContent)
	}
}

func TestEmptyExternalConfig(t *testing.T) {
	cfg := &config.Config{
		External: []config.ExternalDep{},
	}

	p := &platform.Platform{
		OS: "linux",
	}

	result, err := CloneExternal(cfg, p, ExternalOptions{})
	if err != nil {
		t.Fatalf("CloneExternal() error = %v", err)
	}

	if len(result.Cloned) != 0 || len(result.Failed) != 0 || len(result.Skipped) != 0 {
		t.Error("Expected empty result for empty config")
	}
}

// TestGitClone_URLInjection verifies that malicious URLs are rejected by the
// validation layer before they can reach exec.Command. Tests both through the
// validation function directly and through the unexported gitClone function.
func TestGitClone_URLInjection(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Flag injection attacks — these must be rejected
		{name: "upload-pack flag injection", url: "--upload-pack=malicious", wantErr: true},
		{name: "config flag injection", url: "--config=core.sshCommand=evil", wantErr: true},
		{name: "single hyphen flag", url: "-c", wantErr: true},
		{name: "template flag injection", url: "--template=/tmp/evil", wantErr: true},
		{name: "recurse-submodules injection", url: "--recurse-submodules=evil", wantErr: true},

		// file:// scheme attacks — local file access must be blocked
		{name: "file scheme", url: "file:///etc/passwd", wantErr: true},
		{name: "file scheme uppercase", url: "FILE:///etc/passwd", wantErr: true},

		// Shell metacharacter injection
		{name: "semicolon injection", url: "https://evil.com/repo;rm -rf /", wantErr: true},
		{name: "pipe injection", url: "https://evil.com/repo|cat /etc/passwd", wantErr: true},
		{name: "backtick injection", url: "https://evil.com/`whoami`/repo", wantErr: true},
		{name: "newline injection", url: "https://github.com/user/repo\n--upload-pack=evil", wantErr: true},

		// Valid URLs that must be accepted
		{name: "valid https url", url: "https://github.com/user/repo.git", wantErr: false},
		{name: "valid ssh url", url: "git@github.com:user/repo.git", wantErr: false},
		{name: "valid https without .git", url: "https://github.com/user/repo", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateGitURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGitURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

// TestGitClone_Validation verifies that gitClone rejects invalid URLs
// before attempting to run git. Since gitClone is unexported, we call
// it directly from a test in the same package.
func TestGitClone_Validation(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		errMsg string
	}{
		{
			name:   "flag injection rejected",
			url:    "--upload-pack=malicious",
			errMsg: "invalid git URL",
		},
		{
			name:   "file scheme rejected",
			url:    "file:///etc/passwd",
			errMsg: "invalid git URL",
		},
		{
			name:   "empty url rejected",
			url:    "",
			errMsg: "invalid git URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a dummy destination; validation should fail before git runs
			err := gitClone(tt.url, "/tmp/go4dot-test-should-not-exist")
			if err == nil {
				t.Errorf("gitClone(%q, ...) expected error but got nil", tt.url)
				return
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("gitClone(%q, ...) error = %q, want it to contain %q", tt.url, err.Error(), tt.errMsg)
			}
		})
	}
}

// TestGitPull_PathValidation verifies that gitPull rejects relative paths.
// gitPull is unexported, so we call it directly from within the package.
func TestGitPull_PathValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "relative path with dots", path: "../evil/repo"},
		{name: "relative path no dots", path: "some/path"},
		{name: "dot path", path: "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gitPull(tt.path)
			if err == nil {
				t.Errorf("gitPull(%q) expected error for relative path but got nil", tt.path)
				return
			}
			if !strings.Contains(err.Error(), "must be absolute") {
				t.Errorf("gitPull(%q) error = %q, want it to contain 'must be absolute'", tt.path, err.Error())
			}
		})
	}
}
