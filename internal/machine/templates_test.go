package machine

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
)

// homeTempDir creates a temporary directory under $HOME and returns:
//   - the absolute path to the temp dir
//   - a cleanup function
//
// This is needed because expandPath now requires paths to start with ~/
// and stay within the home directory.
func homeTempDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}
	tmpDir, err := os.MkdirTemp(home, ".go4dot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir under home: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	return tmpDir
}

// tildeRelPath converts an absolute path under $HOME to a ~/ relative path.
func tildeRelPath(t *testing.T, absPath string) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}
	rel, err := filepath.Rel(home, absPath)
	if err != nil {
		t.Fatalf("Failed to compute relative path: %v", err)
	}
	return "~/" + rel
}

func TestRenderMachineConfig(t *testing.T) {
	mc := &config.MachinePrompt{
		ID:          "git",
		Description: "Git configuration",
		Destination: "~/.gitconfig.local",
		Template:    "[user]\n    name = {{ .user_name }}\n    email = {{ .user_email }}",
	}

	values := map[string]string{
		"user_name":  "John Doe",
		"user_email": "john@example.com",
	}

	result, err := RenderMachineConfig(mc, values)
	if err != nil {
		t.Fatalf("RenderMachineConfig failed: %v", err)
	}

	expected := "[user]\n    name = John Doe\n    email = john@example.com"
	if result.Content != expected {
		t.Errorf("Content mismatch.\nGot:\n%s\nWant:\n%s", result.Content, expected)
	}

	if result.ID != "git" {
		t.Errorf("ID mismatch: got %q, want 'git'", result.ID)
	}

	// Destination should be expanded
	home, _ := os.UserHomeDir()
	expectedDest := filepath.Join(home, ".gitconfig.local")
	if result.Destination != expectedDest {
		t.Errorf("Destination mismatch: got %q, want %q", result.Destination, expectedDest)
	}
}

func TestRenderMachineConfigInvalidTemplate(t *testing.T) {
	mc := &config.MachinePrompt{
		ID:          "invalid",
		Destination: "~/test",
		Template:    "{{ .unclosed",
	}

	_, err := RenderMachineConfig(mc, nil)
	if err == nil {
		t.Error("Expected error for invalid template")
	}
}

func TestRenderAndWrite(t *testing.T) {
	tmpDir := homeTempDir(t)
	destPath := filepath.Join(tmpDir, "config.txt")
	tildeDest := tildeRelPath(t, destPath)

	mc := &config.MachinePrompt{
		ID:          "test",
		Description: "Test config",
		Destination: tildeDest,
		Template:    "Hello, {{ .name }}!",
	}

	values := map[string]string{
		"name": "World",
	}

	var progressMessages []string
	opts := RenderOptions{
		ProgressFunc: func(current, total int, msg string) {
			progressMessages = append(progressMessages, msg)
		},
	}

	result, err := RenderAndWrite(mc, values, opts)
	if err != nil {
		t.Fatalf("RenderAndWrite failed: %v", err)
	}

	if result.Content != "Hello, World!" {
		t.Errorf("Content mismatch: got %q", result.Content)
	}

	// Verify file was written
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("File content mismatch: got %q", string(content))
	}

	if len(progressMessages) == 0 {
		t.Error("Expected progress messages")
	}
}

func TestRenderAndWriteDryRun(t *testing.T) {
	tmpDir := homeTempDir(t)
	destPath := filepath.Join(tmpDir, "dryrun.txt")
	tildeDest := tildeRelPath(t, destPath)

	mc := &config.MachinePrompt{
		ID:          "test",
		Destination: tildeDest,
		Template:    "Content",
	}

	opts := RenderOptions{
		DryRun: true,
	}

	result, err := RenderAndWrite(mc, nil, opts)
	if err != nil {
		t.Fatalf("RenderAndWrite failed: %v", err)
	}

	if result.Content != "Content" {
		t.Errorf("Content mismatch: got %q", result.Content)
	}

	// File should NOT exist in dry run
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		t.Error("File should not exist in dry run mode")
	}
}

func TestRenderAndWriteExistingFileNoOverwrite(t *testing.T) {
	tmpDir := homeTempDir(t)
	destPath := filepath.Join(tmpDir, "existing.txt")
	tildeDest := tildeRelPath(t, destPath)

	// Create existing file
	if err := os.WriteFile(destPath, []byte("existing"), 0600); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	mc := &config.MachinePrompt{
		ID:          "test",
		Destination: tildeDest,
		Template:    "new content",
	}

	opts := RenderOptions{
		Overwrite: false,
	}

	_, err := RenderAndWrite(mc, nil, opts)
	if err == nil {
		t.Error("Expected error when file exists and overwrite is false")
	}
}

func TestRenderAndWriteExistingFileWithOverwrite(t *testing.T) {
	tmpDir := homeTempDir(t)
	destPath := filepath.Join(tmpDir, "existing.txt")
	tildeDest := tildeRelPath(t, destPath)

	// Create existing file
	if err := os.WriteFile(destPath, []byte("existing"), 0600); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	mc := &config.MachinePrompt{
		ID:          "test",
		Destination: tildeDest,
		Template:    "new content",
	}

	opts := RenderOptions{
		Overwrite: true,
	}

	result, err := RenderAndWrite(mc, nil, opts)
	if err != nil {
		t.Fatalf("RenderAndWrite failed: %v", err)
	}

	if result.Content != "new content" {
		t.Errorf("Content mismatch: got %q", result.Content)
	}

	// Verify file was overwritten
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "new content" {
		t.Errorf("File not overwritten: got %q", string(content))
	}
}

func TestCheckMachineConfigStatus(t *testing.T) {
	tmpDir := homeTempDir(t)

	// Create an existing file
	existingPath := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	cfg := &config.Config{
		MachineConfig: []config.MachinePrompt{
			{
				ID:          "existing",
				Description: "Existing config",
				Destination: tildeRelPath(t, existingPath),
			},
			{
				ID:          "missing",
				Description: "Missing config",
				Destination: tildeRelPath(t, filepath.Join(tmpDir, "missing.txt")),
			},
		},
	}

	statuses := CheckMachineConfigStatus(cfg)

	if len(statuses) != 2 {
		t.Fatalf("Expected 2 statuses, got %d", len(statuses))
	}

	// Find statuses by ID
	var existingStatus, missingStatus *MachineConfigStatus
	for i := range statuses {
		switch statuses[i].ID {
		case "existing":
			existingStatus = &statuses[i]
		case "missing":
			missingStatus = &statuses[i]
		}
	}

	if existingStatus == nil || existingStatus.Status != "configured" {
		t.Errorf("Expected existing status 'configured', got %+v", existingStatus)
	}

	if missingStatus == nil || missingStatus.Status != "missing" {
		t.Errorf("Expected missing status 'missing', got %+v", missingStatus)
	}
}

func TestRemoveMachineConfig(t *testing.T) {
	tmpDir := homeTempDir(t)
	filePath := filepath.Join(tmpDir, "toremove.txt")
	tildePath := tildeRelPath(t, filePath)

	// Create file to remove
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	mc := &config.MachinePrompt{
		ID:          "test",
		Destination: tildePath,
	}

	var progressMessages []string
	opts := RenderOptions{
		ProgressFunc: func(current, total int, msg string) {
			progressMessages = append(progressMessages, msg)
		},
	}

	err := RemoveMachineConfig(mc, opts)
	if err != nil {
		t.Fatalf("RemoveMachineConfig failed: %v", err)
	}

	// File should be removed
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("File should be removed")
	}

	if len(progressMessages) == 0 {
		t.Error("Expected progress messages")
	}
}

func TestRemoveMachineConfigDryRun(t *testing.T) {
	tmpDir := homeTempDir(t)
	filePath := filepath.Join(tmpDir, "dryrun.txt")
	tildePath := tildeRelPath(t, filePath)

	// Create file
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	mc := &config.MachinePrompt{
		ID:          "test",
		Destination: tildePath,
	}

	opts := RenderOptions{DryRun: true}

	err := RemoveMachineConfig(mc, opts)
	if err != nil {
		t.Fatalf("RemoveMachineConfig failed: %v", err)
	}

	// File should still exist in dry run
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File should still exist in dry run mode")
	}
}

func TestRemoveMachineConfigNotExists(t *testing.T) {
	mc := &config.MachinePrompt{
		ID:          "test",
		Destination: "~/nonexistent-go4dot-test/file.txt",
	}

	err := RemoveMachineConfig(mc, RenderOptions{})
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "Valid template",
			template: "Hello, {{ .name }}!",
			wantErr:  false,
		},
		{
			name:     "Valid with conditionals",
			template: "{{ if .enable }}enabled{{ else }}disabled{{ end }}",
			wantErr:  false,
		},
		{
			name:     "Invalid unclosed",
			template: "{{ .unclosed",
			wantErr:  true,
		},
		{
			name:     "Invalid action",
			template: "{{ invalid syntax }}",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.template)
			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestPreviewRender(t *testing.T) {
	mc := &config.MachinePrompt{
		ID:          "test",
		Destination: "~/test",
		Template:    "Hello, {{ .name }}!",
	}

	values := map[string]string{
		"name": "Preview",
	}

	content, err := PreviewRender(mc, values)
	if err != nil {
		t.Fatalf("PreviewRender failed: %v", err)
	}

	if content != "Hello, Preview!" {
		t.Errorf("Content mismatch: got %q", content)
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}

	t.Run("valid tilde path", func(t *testing.T) {
		result, err := expandPath("~/.config")
		if err != nil {
			t.Fatalf("expandPath(\"~/.config\") failed: %v", err)
		}
		expected := filepath.Join(home, ".config")
		if result != expected {
			t.Errorf("expandPath(\"~/.config\") = %q, want %q", result, expected)
		}
	})

	t.Run("valid nested tilde path", func(t *testing.T) {
		result, err := expandPath("~/.config/nvim/init.vim")
		if err != nil {
			t.Fatalf("expandPath failed: %v", err)
		}
		expected := filepath.Join(home, ".config/nvim/init.vim")
		if result != expected {
			t.Errorf("expandPath = %q, want %q", result, expected)
		}
	})
}

func TestExpandPathRejectsNonTildePaths(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "absolute path", input: "/absolute/path"},
		{name: "relative path", input: "relative/path"},
		{name: "empty string", input: ""},
		{name: "tilde only", input: "~"},
		{name: "tilde without slash", input: "~config"},
		{name: "dot path", input: "./config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := expandPath(tt.input)
			if err == nil {
				t.Errorf("expandPath(%q) should have returned error for non-~/ path", tt.input)
			}
			if err != nil && !strings.Contains(err.Error(), "must start with ~/") {
				t.Errorf("expandPath(%q) error = %q, expected 'must start with ~/' message", tt.input, err.Error())
			}
		})
	}
}

func TestExpandPathRejectsTraversal(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "parent traversal", input: "~/../../etc/shadow"},
		{name: "deep traversal", input: "~/../../../tmp/evil"},
		{name: "single parent", input: "~/.."},
		{name: "dotdot in middle", input: "~/.config/../../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := expandPath(tt.input)
			if err == nil {
				t.Errorf("expandPath(%q) should have returned error for path traversal", tt.input)
			}
			if err != nil && !strings.Contains(err.Error(), "escapes base directory") {
				t.Errorf("expandPath(%q) error = %q, expected 'escapes base directory' message", tt.input, err.Error())
			}
		})
	}
}

func TestRenderAndWriteFilePermissions(t *testing.T) {
	tmpDir := homeTempDir(t)
	destPath := filepath.Join(tmpDir, "subdir", "secret.conf")
	tildeDest := tildeRelPath(t, destPath)

	mc := &config.MachinePrompt{
		ID:          "test",
		Destination: tildeDest,
		Template:    "secret_key = abc123",
	}

	opts := RenderOptions{Overwrite: true}

	_, err := RenderAndWrite(mc, nil, opts)
	if err != nil {
		t.Fatalf("RenderAndWrite failed: %v", err)
	}

	// Verify file permissions are 0600 (owner read/write only)
	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	filePerm := info.Mode().Perm()
	if filePerm != fs.FileMode(0600) {
		t.Errorf("File permissions = %o, want 0600", filePerm)
	}

	// Verify parent directory permissions are 0700 (owner only)
	dirInfo, err := os.Stat(filepath.Dir(destPath))
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}
	dirPerm := dirInfo.Mode().Perm()
	if dirPerm != fs.FileMode(0700) {
		t.Errorf("Directory permissions = %o, want 0700", dirPerm)
	}
}

func TestRenderAll(t *testing.T) {
	tmpDir := homeTempDir(t)

	cfg := &config.Config{
		MachineConfig: []config.MachinePrompt{
			{
				ID:          "config1",
				Destination: tildeRelPath(t, filepath.Join(tmpDir, "config1.txt")),
				Template:    "Config 1: {{ .value }}",
			},
			{
				ID:          "config2",
				Destination: tildeRelPath(t, filepath.Join(tmpDir, "config2.txt")),
				Template:    "Config 2: {{ .value }}",
			},
		},
	}

	results := []PromptResult{
		{
			ID:     "config1",
			Values: map[string]string{"value": "Value1"},
		},
		{
			ID:     "config2",
			Values: map[string]string{"value": "Value2"},
		},
	}

	opts := RenderOptions{}

	rendered, err := RenderAll(cfg, results, opts)
	if err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	if len(rendered) != 2 {
		t.Fatalf("Expected 2 rendered, got %d", len(rendered))
	}

	// Verify files were written
	content1, err := os.ReadFile(filepath.Join(tmpDir, "config1.txt"))
	if err != nil {
		t.Fatalf("Failed to read config1: %v", err)
	}
	if string(content1) != "Config 1: Value1" {
		t.Errorf("Config1 content mismatch: got %q", string(content1))
	}

	content2, err := os.ReadFile(filepath.Join(tmpDir, "config2.txt"))
	if err != nil {
		t.Fatalf("Failed to read config2: %v", err)
	}
	if string(content2) != "Config 2: Value2" {
		t.Errorf("Config2 content mismatch: got %q", string(content2))
	}
}
