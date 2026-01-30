//go:build e2e

package helpers

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ContainerRuntime represents docker or podman
type ContainerRuntime string

const (
	RuntimeDocker ContainerRuntime = "docker"
	RuntimePodman ContainerRuntime = "podman"
)

// DockerTestContainer represents a running test container
type DockerTestContainer struct {
	Runtime     ContainerRuntime
	ContainerID string
	ImageName   string
	t           *testing.T
}

// DetectContainerRuntime finds available container runtime (docker or podman)
func DetectContainerRuntime(t *testing.T) ContainerRuntime {
	t.Helper()

	// Try docker first
	if cmd := exec.Command("docker", "info"); cmd.Run() == nil {
		return RuntimeDocker
	}

	// Try podman
	if cmd := exec.Command("podman", "info"); cmd.Run() == nil {
		return RuntimePodman
	}

	t.Fatal("No working container runtime found (tried docker and podman)")
	return ""
}

// GetProjectRoot returns the absolute path to the project root
func GetProjectRoot(t *testing.T) string {
	t.Helper()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Walk up until we find go.mod
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (no go.mod found)")
		}
		dir = parent
	}
}

// BuildTestBinary builds the g4d binary for Linux
func BuildTestBinary(t *testing.T) string {
	t.Helper()

	projectRoot := GetProjectRoot(t)
	binPath := filepath.Join(projectRoot, "bin", "g4d-test-linux-amd64")

	t.Logf("Building test binary: %s", binPath)

	cmd := exec.Command("go", "build",
		"-o", binPath,
		"./cmd/g4d")
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build test binary: %v\nOutput: %s", err, output)
	}

	return binPath
}

// DockerConfig holds configuration for creating a test container
type DockerConfig struct {
	// ImageName is the base image to use (default: ubuntu:latest)
	ImageName string

	// BinaryPath is the path to the g4d binary to copy into container
	BinaryPath string

	// FixturesDir is the directory containing test fixtures to copy
	FixturesDir string

	// WorkDir is the working directory inside container
	WorkDir string

	// Env contains additional environment variables
	Env map[string]string

	// NoCleanup prevents automatic cleanup (useful for debugging)
	NoCleanup bool

	// VHSEnabled installs VHS and its dependencies (ttyd, ffmpeg) in the container
	VHSEnabled bool
}

// NewDockerTestContainer creates and starts a test container
func NewDockerTestContainer(t *testing.T, cfg DockerConfig) *DockerTestContainer {
	t.Helper()

	runtime := DetectContainerRuntime(t)

	// Set defaults
	if cfg.ImageName == "" {
		cfg.ImageName = "ubuntu:22.04" // Pinned version for reproducible tests
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = "/home/testuser"
	}

	// Create a minimal Dockerfile for the test
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")

	dockerfile := fmt.Sprintf(`FROM %s

# Install dependencies
RUN apt-get update && apt-get install -y \
    git \
    stow \
    curl \
    sudo \
    zsh \
    vim \
    locales \
    && rm -rf /var/lib/apt/lists/*

# Set up locales for UTF-8 support
RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8
ENV TERM=xterm-256color

# Create test user
RUN useradd -m -s /bin/zsh testuser && \
    echo "testuser ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

USER testuser
WORKDIR %s

# Default command
CMD ["/bin/zsh", "-i"]
`, cfg.ImageName, cfg.WorkDir)

	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}

	// Copy binary to build context if provided
	if cfg.BinaryPath != "" {
		binDest := filepath.Join(tmpDir, "g4d")
		if err := copyFile(cfg.BinaryPath, binDest); err != nil {
			t.Fatalf("failed to copy binary: %v", err)
		}

		// Add binary installation to Dockerfile
		dockerfile += "\nCOPY g4d /usr/local/bin/g4d\nRUN sudo chmod +x /usr/local/bin/g4d\n"
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
			t.Fatalf("failed to update Dockerfile: %v", err)
		}
	}

	// Copy fixtures if provided
	if cfg.FixturesDir != "" {
		fixturesDest := filepath.Join(tmpDir, "fixtures")
		if err := copyDir(cfg.FixturesDir, fixturesDest); err != nil {
			t.Fatalf("failed to copy fixtures: %v", err)
		}

		dockerfile += "\nCOPY fixtures /home/testuser/fixtures\n"
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
			t.Fatalf("failed to update Dockerfile: %v", err)
		}
	}

	// Install VHS and dependencies if enabled
	if cfg.VHSEnabled {
		vhsInstall := `
# Install VHS dependencies (as root)
USER root
RUN apt-get update && apt-get install -y \
    ffmpeg \
    chromium-browser \
    fonts-noto-color-emoji \
    fonts-dejavu \
    && rm -rf /var/lib/apt/lists/*

# Install ttyd for terminal recording
RUN curl -sL https://github.com/tsl0922/ttyd/releases/download/1.7.7/ttyd.x86_64 -o /usr/local/bin/ttyd && \
    chmod +x /usr/local/bin/ttyd

# Install VHS using Go
RUN curl -sL https://go.dev/dl/go1.23.5.linux-amd64.tar.gz | tar -C /usr/local -xzf -
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"
RUN /usr/local/go/bin/go install github.com/charmbracelet/vhs@latest && \
    cp /root/go/bin/vhs /usr/local/bin/vhs

# Set Chrome path for VHS
ENV VHS_CHROME_PATH=/usr/bin/chromium-browser

# Switch back to testuser
USER testuser
ENV PATH="/usr/local/go/bin:${PATH}"
`
		dockerfile += vhsInstall
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
			t.Fatalf("failed to update Dockerfile with VHS: %v", err)
		}
	}

	// Build image with unique name for parallel test safety
	imageName := fmt.Sprintf("g4d-test-%d", time.Now().UnixNano())
	t.Logf("Building test image: %s", imageName)

	buildCmd := exec.Command(string(runtime), "build", "-t", imageName, tmpDir)
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build image: %v\nOutput: %s", err, buildOutput)
	}

	// Run container
	runArgs := []string{"run", "-d"}

	// Add environment variables
	for k, v := range cfg.Env {
		runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	runArgs = append(runArgs, imageName, "tail", "-f", "/dev/null")

	runCmd := exec.Command(string(runtime), runArgs...)
	runOutput, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run container: %v\nOutput: %s", err, runOutput)
	}

	containerID := strings.TrimSpace(string(runOutput))
	// Safe substring for logging (avoid panic on empty/short output)
	shortID := containerID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	t.Logf("Started container: %s", shortID)

	container := &DockerTestContainer{
		Runtime:     runtime,
		ContainerID: containerID,
		ImageName:   imageName,
		t:           t,
	}

	// Register cleanup
	if !cfg.NoCleanup {
		t.Cleanup(func() {
			container.Cleanup()
		})
	}

	return container
}

// Exec runs a command inside the container and returns output
func (c *DockerTestContainer) Exec(command ...string) (string, error) {
	c.t.Helper()

	args := []string{"exec", c.ContainerID}
	args = append(args, command...)

	cmd := exec.Command(string(c.Runtime), args...)
	output, err := cmd.CombinedOutput()

	return string(output), err
}

// ExecWithStdin runs a command with stdin input
func (c *DockerTestContainer) ExecWithStdin(stdin string, command ...string) (string, error) {
	c.t.Helper()

	args := []string{"exec", "-i", c.ContainerID}
	args = append(args, command...)

	cmd := exec.Command(string(c.Runtime), args...)
	cmd.Stdin = strings.NewReader(stdin)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ExecContext runs a command with context support
func (c *DockerTestContainer) ExecContext(ctx context.Context, command ...string) (string, error) {
	c.t.Helper()

	args := []string{"exec", c.ContainerID}
	args = append(args, command...)

	cmd := exec.CommandContext(ctx, string(c.Runtime), args...)
	output, err := cmd.CombinedOutput()

	return string(output), err
}

// CopyToContainer copies a file or directory into the container
func (c *DockerTestContainer) CopyToContainer(src, dest string) error {
	c.t.Helper()

	cmd := exec.Command(string(c.Runtime), "cp", src, c.ContainerID+":"+dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy to container: %w\nOutput: %s", err, output)
	}

	return nil
}

// CopyFromContainer copies a file or directory from the container
func (c *DockerTestContainer) CopyFromContainer(src, dest string) error {
	c.t.Helper()

	cmd := exec.Command(string(c.Runtime), "cp", c.ContainerID+":"+src, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy from container: %w\nOutput: %s", err, output)
	}

	return nil
}

// VHSTapeConfig configures VHS tape execution in a container
type VHSTapeConfig struct {
	// TapePath is the local path to the VHS tape file
	TapePath string

	// OutputPath is the local path where output file will be saved
	OutputPath string

	// ContainerWorkDir is the working directory in the container for VHS execution
	// Defaults to /home/testuser
	ContainerWorkDir string
}

// RunVHSTape executes a VHS tape inside the container and extracts the output
func (c *DockerTestContainer) RunVHSTape(cfg VHSTapeConfig) error {
	c.t.Helper()

	if cfg.ContainerWorkDir == "" {
		cfg.ContainerWorkDir = "/home/testuser"
	}

	// Get tape filename
	tapeFilename := filepath.Base(cfg.TapePath)
	containerTapePath := filepath.Join(cfg.ContainerWorkDir, tapeFilename)

	// Read and modify tape to use container paths
	tapeContent, err := os.ReadFile(cfg.TapePath)
	if err != nil {
		return fmt.Errorf("failed to read tape file: %w", err)
	}

	// Extract output filename from tape (look for "Output" directive)
	outputFilename := extractVHSOutputFilename(string(tapeContent))
	if outputFilename == "" {
		return fmt.Errorf("tape does not contain Output directive")
	}

	// Modify tape to use container output path
	containerOutputPath := filepath.Join(cfg.ContainerWorkDir, filepath.Base(outputFilename))
	modifiedTape := modifyVHSTapeOutput(string(tapeContent), containerOutputPath)

	// Modify tape to use g4d from PATH instead of ./bin/g4d
	modifiedTape = strings.ReplaceAll(modifiedTape, "./bin/g4d", "g4d")

	// Write modified tape to temp file
	tmpTape := filepath.Join(c.t.TempDir(), tapeFilename)
	if err := os.WriteFile(tmpTape, []byte(modifiedTape), 0644); err != nil {
		return fmt.Errorf("failed to write modified tape: %w", err)
	}

	// Copy tape to container
	if err := c.CopyToContainer(tmpTape, containerTapePath); err != nil {
		return fmt.Errorf("failed to copy tape to container: %w", err)
	}

	// Run VHS inside container
	c.t.Logf("Running VHS tape: %s", tapeFilename)
	output, err := c.Exec("vhs", containerTapePath)
	if err != nil {
		return fmt.Errorf("VHS execution failed: %w\nOutput: %s", err, output)
	}
	c.t.Logf("VHS output: %s", output)

	// Create output directory if needed
	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Copy output file from container
	if err := c.CopyFromContainer(containerOutputPath, cfg.OutputPath); err != nil {
		return fmt.Errorf("failed to copy output from container: %w", err)
	}

	return nil
}

// extractVHSOutputFilename extracts the output filename from a VHS tape
func extractVHSOutputFilename(tape string) string {
	lines := strings.Split(tape, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Output ") {
			return strings.TrimPrefix(line, "Output ")
		}
	}
	return ""
}

// modifyVHSTapeOutput modifies the Output directive in a VHS tape
func modifyVHSTapeOutput(tape, newOutputPath string) string {
	lines := strings.Split(tape, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Output ") {
			lines[i] = "Output " + newOutputPath
			break
		}
	}
	return strings.Join(lines, "\n")
}

// Cleanup stops and removes the container and image
func (c *DockerTestContainer) Cleanup() {
	c.t.Helper()

	// Stop container
	stopCmd := exec.Command(string(c.Runtime), "stop", c.ContainerID)
	if output, err := stopCmd.CombinedOutput(); err != nil {
		c.t.Logf("Warning: failed to stop container: %v\nOutput: %s", err, output)
	}

	// Remove container
	rmCmd := exec.Command(string(c.Runtime), "rm", c.ContainerID)
	if output, err := rmCmd.CombinedOutput(); err != nil {
		c.t.Logf("Warning: failed to remove container: %v\nOutput: %s", err, output)
	}

	// Remove image
	rmiCmd := exec.Command(string(c.Runtime), "rmi", c.ImageName)
	if output, err := rmiCmd.CombinedOutput(); err != nil {
		c.t.Logf("Warning: failed to remove image: %v\nOutput: %s", err, output)
	}
}

// Helper functions

func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer destFile.Close()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copy content: %w", err)
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Construct destination path
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

