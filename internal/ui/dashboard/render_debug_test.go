package dashboard

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
)

// TestRenderDebug is a visual validation test for the dashboard layout.
// Run with: go test ./internal/ui/dashboard/... -run TestRenderDebug -v
// This test renders the dashboard at multiple sizes for visual inspection.
// It does NOT modify the filesystem - it only renders UI strings.
func TestRenderDebug(t *testing.T) {
	// Sample link status with nested file structure
	vimLinkStatus := &stow.ConfigLinkStatus{
		ConfigName: "vim",
		TotalCount: 5,
		LinkedCount: 4,
		Files: []stow.FileStatus{
			{RelPath: ".vimrc", IsLinked: true},
			{RelPath: ".config/nvim/init.vim", IsLinked: true},
			{RelPath: ".config/nvim/lua/plugins.lua", IsLinked: true},
			{RelPath: ".config/nvim/lua/settings.lua", IsLinked: true},
			{RelPath: ".config/nvim/after/ftplugin/go.vim", IsLinked: false, Issue: "file exists"},
		},
	}

	state := State{
		Platform: &platform.Platform{OS: "linux", Distro: "fedora"},
		Configs: []config.ConfigItem{
			{Name: "vim", Description: "Vim configuration"},
			{Name: "zsh", Description: "Zsh shell configuration"},
			{Name: "tmux", Description: "Tmux terminal multiplexer"},
			{Name: "git", Description: "Git configuration"},
			{Name: "alacritty", Description: "Alacritty terminal"},
		},
		LinkStatus: map[string]*stow.ConfigLinkStatus{
			"vim": vimLinkStatus,
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	model := New(state)

	sizes := []struct {
		name   string
		width  int
		height int
	}{
		{"Large (120x35)", 120, 35},
		{"Medium (100x30)", 100, 30},
		{"Small (80x24)", 80, 24},
	}

	for _, size := range sizes {
		sizeMsg := tea.WindowSizeMsg{Width: size.width, Height: size.height}
		updatedModel, _ := model.Update(sizeMsg)
		m := updatedModel.(*Model)

		fmt.Printf("\n=== DASHBOARD RENDER %s ===\n", size.name)
		fmt.Println(m.View())
		fmt.Println("=== END ===")
	}
}
