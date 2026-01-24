package dashboard

import (
	"testing"

	"github.com/nvandessel/go4dot/internal/platform"
)

func TestNew(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
	}
	m := New(s)
	if m.state.Platform.OS != "linux" {
		t.Errorf("expected OS to be linux, got %s", m.state.Platform.OS)
	}
}
