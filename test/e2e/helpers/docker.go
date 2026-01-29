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
}

// NewDockerTestContainer creates and starts a test container
func NewDockerTestContainer(t *testing.T, cfg DockerConfig) *DockerTestContainer {
	t.Helper()

	runtime := DetectContainerRuntime(t)

	// Set defaults
	if cfg.ImageName == "" {
		cfg.ImageName = "ubuntu:latest"
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

	// Build image
	imageName := fmt.Sprintf("g4d-test-%d", time.Now().Unix())
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
	t.Logf("Started container: %s", containerID[:12])

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
		return fmt.Errorf("failed to copy to container: %v\nOutput: %s", err, output)
	}

	return nil
}

// CopyFromContainer copies a file or directory from the container
func (c *DockerTestContainer) CopyFromContainer(src, dest string) error {
	c.t.Helper()

	cmd := exec.Command(string(c.Runtime), "cp", c.ContainerID+":"+src, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy from container: %v\nOutput: %s", err, output)
	}

	return nil
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
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
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
