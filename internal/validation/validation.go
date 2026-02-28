// Package validation provides reusable input validators for security hardening.
// These validators prevent command injection, flag injection, and path traversal
// attacks by sanitizing user-controlled inputs before they reach shell commands
// or filesystem operations.
package validation

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// maxNameLength is the maximum allowed length for names (binary, package, config).
const maxNameLength = 255

// maxEmailLength is the maximum allowed length for email addresses per RFC 5321.
const maxEmailLength = 254

// allowedVersionCmds is the whitelist of accepted version command arguments.
var allowedVersionCmds = map[string]bool{
	"--version": true,
	"-v":        true,
	"-V":        true,
	"version":   true,
}

// binaryNameRegexp matches only safe characters for binary names:
// alphanumeric, hyphens, underscores, and dots.
var binaryNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// packageNameRegexp matches safe characters for package names:
// alphanumeric, hyphens, underscores, dots, plus, at-sign, and forward slash.
var packageNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9._\-+@/]+$`)

// configNameRegexp matches safe characters for config names:
// alphanumeric, hyphens, underscores, dots, plus, and at-sign.
// Forward slash and backslash are explicitly excluded.
var configNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9._\-+@]+$`)

// emailRegexp matches basic email format: local@domain.
var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)

// gpgKeyIDRegexp matches valid GPG key IDs: 8-40 hex characters.
var gpgKeyIDRegexp = regexp.MustCompile(`^[A-Fa-f0-9]{8,40}$`)

// keyTitleRegexp matches safe key titles: starts with alphanumeric,
// then allows alphanumeric, space, dot, underscore, hyphen, parentheses, at-sign.
var keyTitleRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9 ._\-()@]*$`)

// maxKeyTitleLength is the maximum allowed length for key titles.
const maxKeyTitleLength = 100

// sshKeyFilenameRegexp matches safe SSH key filenames.
var sshKeyFilenameRegexp = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// dangerousSSHFilenames are filenames that should never be written by key generation.
var dangerousSSHFilenames = map[string]bool{
	"authorized_keys":     true,
	"authorized_keys.pub": true,
	"config":              true,
	"config.pub":          true,
	"known_hosts":         true,
	"known_hosts.pub":     true,
	"environment":         true,
	"environment.pub":     true,
}

// gitHTTPSRegexp matches HTTPS git URLs with a fully anchored pattern.
var gitHTTPSRegexp = regexp.MustCompile(`^https://[a-zA-Z0-9][a-zA-Z0-9.\-@:/_~%+]*$`)

// gitSSHRegexp matches SSH git URLs in the git@host:user/repo.git format with a fully anchored pattern.
var gitSSHRegexp = regexp.MustCompile(`^git@[a-zA-Z0-9][a-zA-Z0-9.\-]*:[a-zA-Z0-9][a-zA-Z0-9.\-_/]*$`)

// ValidateBinaryName checks that a binary name contains only safe characters.
// It rejects empty strings, names starting with a hyphen (flag injection),
// names containing path separators or shell metacharacters, and names
// exceeding 255 characters.
func ValidateBinaryName(name string) error {
	if name == "" {
		return fmt.Errorf("binary name must not be empty")
	}

	if len(name) > maxNameLength {
		return fmt.Errorf("binary name exceeds maximum length of %d characters", maxNameLength)
	}

	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("binary name must not start with a hyphen: %q", name)
	}

	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("binary name must not contain whitespace: %q", name)
	}

	if !binaryNameRegexp.MatchString(name) {
		return fmt.Errorf("binary name contains invalid characters: %q (allowed: alphanumeric, hyphen, underscore, dot)", name)
	}

	return nil
}

// ValidateVersionCmd checks that a version command argument is one of the
// allowed values: --version, -v, -V, or version. This whitelist approach
// prevents arbitrary command injection through version check arguments.
func ValidateVersionCmd(cmd string) error {
	if !allowedVersionCmds[cmd] {
		return fmt.Errorf("invalid version command: %q (allowed: --version, -v, -V, version)", cmd)
	}
	return nil
}

// ValidateGitURL checks that a URL is a valid HTTPS or SSH git URL.
// It rejects empty strings, file:// scheme URLs, strings containing control
// characters (newline injection), and strings starting with hyphens (flag
// injection). Only https:// and git@host:user/repo.git formats are accepted.
func ValidateGitURL(url string) error {
	if url == "" {
		return fmt.Errorf("git URL must not be empty")
	}

	if strings.HasPrefix(url, "-") {
		return fmt.Errorf("git URL must not start with a hyphen: %q", url)
	}

	// Reject control characters, whitespace, and shell metacharacters which can be
	// used for argument injection or command injection attacks.
	if strings.ContainsAny(url, "\n\r\t\x00 ;|&$`<>") {
		return fmt.Errorf("git URL must not contain whitespace or shell metacharacters: %q", url)
	}

	if strings.HasPrefix(strings.ToLower(url), "file://") {
		return fmt.Errorf("file:// URLs are not allowed: %q", url)
	}

	if gitHTTPSRegexp.MatchString(url) {
		return nil
	}

	if gitSSHRegexp.MatchString(url) {
		return nil
	}

	return fmt.Errorf("git URL must be https:// or git@host:user/repo.git format: %q", url)
}

// ValidatePackageName checks that a package name contains only safe characters.
// It allows alphanumeric characters, hyphens, underscores, dots, plus signs,
// at-signs, and forward slashes (for scoped packages). It rejects empty strings,
// names starting with hyphens (flag injection), shell metacharacters, and names
// exceeding 255 characters.
func ValidatePackageName(name string) error {
	if name == "" {
		return fmt.Errorf("package name must not be empty")
	}

	if len(name) > maxNameLength {
		return fmt.Errorf("package name exceeds maximum length of %d characters", maxNameLength)
	}

	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("package name must not start with a hyphen: %q", name)
	}

	if !packageNameRegexp.MatchString(name) {
		return fmt.Errorf("package name contains invalid characters: %q (allowed: alphanumeric, hyphen, underscore, dot, plus, at-sign, forward slash)", name)
	}

	return nil
}

// ValidateConfigName checks that a config name contains only safe characters.
// It allows alphanumeric characters, hyphens, underscores, dots, plus signs,
// and at-signs. It rejects empty strings, names exceeding 255 characters,
// names starting with hyphens (flag injection via stow), path separators,
// and shell metacharacters.
func ValidateConfigName(name string) error {
	if name == "" {
		return fmt.Errorf("config name must not be empty")
	}

	if len(name) > maxNameLength {
		return fmt.Errorf("config name exceeds maximum length of %d characters", maxNameLength)
	}

	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("config name must not start with a hyphen: %q", name)
	}

	if !configNameRegexp.MatchString(name) {
		return fmt.Errorf("config name contains invalid characters: %q (allowed: alphanumeric, hyphen, underscore, dot, plus, at-sign)", name)
	}

	return nil
}

// ValidateDestinationPath checks that an expanded destination path does not
// escape the base directory through path traversal. Both expanded and baseDir
// must be non-empty. After cleaning the path with filepath.Clean, it verifies
// using filepath.Rel that the result does not start with ".." which would
// indicate a traversal outside baseDir.
func ValidateDestinationPath(expanded string, baseDir string) error {
	if expanded == "" {
		return fmt.Errorf("destination path must not be empty")
	}

	if baseDir == "" {
		return fmt.Errorf("base directory must not be empty")
	}

	if !filepath.IsAbs(expanded) {
		return fmt.Errorf("destination path must be absolute: %q", expanded)
	}

	if !filepath.IsAbs(baseDir) {
		return fmt.Errorf("base directory must be absolute: %q", baseDir)
	}

	cleaned := filepath.Clean(expanded)
	rel, err := filepath.Rel(baseDir, cleaned)
	if err != nil {
		return fmt.Errorf("cannot determine relative path from %q to %q: %w", baseDir, cleaned, err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("destination path %q escapes base directory %q", expanded, baseDir)
	}

	return nil
}

// ValidateEmail rejects: empty, >254 chars, leading hyphen, missing @, control chars, shell metacharacters.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email must not be empty")
	}
	if len(email) > maxEmailLength {
		return fmt.Errorf("email exceeds maximum length of %d characters", maxEmailLength)
	}
	if strings.HasPrefix(email, "-") {
		return fmt.Errorf("email must not start with a hyphen: %q", email)
	}
	for _, r := range email {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("email must not contain control characters: %q", email)
		}
	}
	if strings.ContainsAny(email, ";|&$`<>") {
		return fmt.Errorf("email must not contain shell metacharacters: %q", email)
	}
	if !emailRegexp.MatchString(email) {
		return fmt.Errorf("invalid email format: %q", email)
	}
	return nil
}

// ValidateSSHKeyPath rejects: non-absolute paths, traversal outside sshDir, hyphen filenames, dangerous filenames.
func ValidateSSHKeyPath(path string, sshDir string) error {
	if path == "" {
		return fmt.Errorf("SSH key path must not be empty")
	}
	if sshDir == "" {
		return fmt.Errorf("SSH directory must not be empty")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("SSH key path must be absolute: %q", path)
	}
	if !filepath.IsAbs(sshDir) {
		return fmt.Errorf("SSH directory must be absolute: %q", sshDir)
	}
	// Check path stays within sshDir
	if err := ValidateDestinationPath(path, sshDir); err != nil {
		return fmt.Errorf("SSH key path escapes SSH directory: %w", err)
	}
	// Check filename safety
	filename := filepath.Base(path)
	if strings.HasPrefix(filename, "-") {
		return fmt.Errorf("SSH key filename must not start with a hyphen: %q", filename)
	}
	if !sshKeyFilenameRegexp.MatchString(filename) {
		return fmt.Errorf("SSH key filename contains invalid characters: %q", filename)
	}
	if dangerousSSHFilenames[filename] {
		return fmt.Errorf("SSH key filename is a reserved SSH file: %q", filename)
	}
	return nil
}

// ValidateGPGKeyID rejects: empty, non-hex, <8 or >40 chars.
func ValidateGPGKeyID(keyID string) error {
	if keyID == "" {
		return fmt.Errorf("GPG key ID must not be empty")
	}
	if !gpgKeyIDRegexp.MatchString(keyID) {
		return fmt.Errorf("GPG key ID must be 8-40 hex characters: %q", keyID)
	}
	return nil
}

// ValidateKeyTitle rejects: empty, >100 chars, leading hyphen, control chars, unsafe chars.
func ValidateKeyTitle(title string) error {
	if title == "" {
		return fmt.Errorf("key title must not be empty")
	}
	if len(title) > maxKeyTitleLength {
		return fmt.Errorf("key title exceeds maximum length of %d characters", maxKeyTitleLength)
	}
	if strings.HasPrefix(title, "-") {
		return fmt.Errorf("key title must not start with a hyphen: %q", title)
	}
	if strings.ContainsAny(title, "\n\r\t\x00") {
		return fmt.Errorf("key title must not contain control characters: %q", title)
	}
	if !keyTitleRegexp.MatchString(title) {
		return fmt.Errorf("key title contains invalid characters: %q (allowed: alphanumeric, space, dot, underscore, hyphen, parentheses, at-sign)", title)
	}
	return nil
}
