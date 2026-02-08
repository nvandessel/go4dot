package version

import (
	"testing"
)

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SemVer
		wantErr bool
	}{
		{
			name:  "full version",
			input: "1.2.3",
			want:  SemVer{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "with v prefix",
			input: "v1.2.3",
			want:  SemVer{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "major only",
			input: "2",
			want:  SemVer{Major: 2, Minor: 0, Patch: 0},
		},
		{
			name:  "major.minor only",
			input: "3.4",
			want:  SemVer{Major: 3, Minor: 4, Patch: 0},
		},
		{
			name:  "zero version",
			input: "0.0.0",
			want:  SemVer{Major: 0, Minor: 0, Patch: 0},
		},
		{
			name:  "large numbers",
			input: "100.200.300",
			want:  SemVer{Major: 100, Minor: 200, Patch: 300},
		},
		{
			name:  "with pre-release suffix",
			input: "1.2.3-beta",
			want:  SemVer{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "with build metadata",
			input: "1.2.3+build.456",
			want:  SemVer{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "with whitespace",
			input: " v1.0.0 ",
			want:  SemVer{Major: 1, Minor: 0, Patch: 0},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "just v",
			input:   "v",
			wantErr: true,
		},
		{
			name:    "non-numeric major",
			input:   "abc.1.2",
			wantErr: true,
		},
		{
			name:    "non-numeric minor",
			input:   "1.abc.2",
			wantErr: true,
		},
		{
			name:    "non-numeric patch",
			input:   "1.2.abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSemVer(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseSemVer(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSemVer(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseSemVer(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSemVerString(t *testing.T) {
	tests := []struct {
		name string
		v    SemVer
		want string
	}{
		{"zero", SemVer{0, 0, 0}, "0.0.0"},
		{"simple", SemVer{1, 2, 3}, "1.2.3"},
		{"large", SemVer{10, 20, 30}, "10.20.30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.String(); got != tt.want {
				t.Errorf("SemVer.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name string
		a    SemVer
		b    SemVer
		want int
	}{
		{
			name: "equal",
			a:    SemVer{1, 2, 3},
			b:    SemVer{1, 2, 3},
			want: 0,
		},
		{
			name: "a major less than b",
			a:    SemVer{1, 0, 0},
			b:    SemVer{2, 0, 0},
			want: -1,
		},
		{
			name: "a major greater than b",
			a:    SemVer{3, 0, 0},
			b:    SemVer{2, 0, 0},
			want: 1,
		},
		{
			name: "a minor less than b",
			a:    SemVer{1, 1, 0},
			b:    SemVer{1, 2, 0},
			want: -1,
		},
		{
			name: "a minor greater than b",
			a:    SemVer{1, 3, 0},
			b:    SemVer{1, 2, 0},
			want: 1,
		},
		{
			name: "a patch less than b",
			a:    SemVer{1, 2, 3},
			b:    SemVer{1, 2, 4},
			want: -1,
		},
		{
			name: "a patch greater than b",
			a:    SemVer{1, 2, 5},
			b:    SemVer{1, 2, 4},
			want: 1,
		},
		{
			name: "major dominates minor",
			a:    SemVer{2, 0, 0},
			b:    SemVer{1, 99, 99},
			want: 1,
		},
		{
			name: "minor dominates patch",
			a:    SemVer{1, 2, 0},
			b:    SemVer{1, 1, 99},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Compare(tt.a, tt.b); got != tt.want {
				t.Errorf("Compare(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsOlderThan(t *testing.T) {
	tests := []struct {
		name string
		a    SemVer
		b    SemVer
		want bool
	}{
		{"older", SemVer{1, 0, 0}, SemVer{2, 0, 0}, true},
		{"equal", SemVer{1, 0, 0}, SemVer{1, 0, 0}, false},
		{"newer", SemVer{2, 0, 0}, SemVer{1, 0, 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.IsOlderThan(tt.b); got != tt.want {
				t.Errorf("IsOlderThan(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsNewerThan(t *testing.T) {
	tests := []struct {
		name string
		a    SemVer
		b    SemVer
		want bool
	}{
		{"newer", SemVer{2, 0, 0}, SemVer{1, 0, 0}, true},
		{"equal", SemVer{1, 0, 0}, SemVer{1, 0, 0}, false},
		{"older", SemVer{1, 0, 0}, SemVer{2, 0, 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.IsNewerThan(tt.b); got != tt.want {
				t.Errorf("IsNewerThan(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMajorMismatch(t *testing.T) {
	tests := []struct {
		name string
		a    SemVer
		b    SemVer
		want bool
	}{
		{"same major", SemVer{1, 0, 0}, SemVer{1, 5, 3}, false},
		{"different major", SemVer{1, 0, 0}, SemVer{2, 0, 0}, true},
		{"both zero", SemVer{0, 1, 0}, SemVer{0, 2, 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MajorMismatch(tt.a, tt.b); got != tt.want {
				t.Errorf("MajorMismatch(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
