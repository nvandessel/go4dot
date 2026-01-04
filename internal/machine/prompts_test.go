package machine

import (
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
)

func TestCollectMachineConfig(t *testing.T) {
	cfg := &config.Config{
		MachineConfig: []config.MachinePrompt{
			{
				ID:          "git",
				Description: "Git configuration",
				Destination: "~/.gitconfig.local",
				Prompts: []config.PromptField{
					{
						ID:      "user_name",
						Prompt:  "Full name for git commits",
						Type:    "text",
						Default: "Test User",
					},
					{
						ID:      "user_email",
						Prompt:  "Email for git commits",
						Type:    "text",
						Default: "test@example.com",
					},
				},
				Template: "[user]\n    name = {{ .user_name }}\n    email = {{ .user_email }}",
			},
		},
	}

	// Use skip prompts to use defaults
	opts := PromptOptions{
		SkipPrompts: true,
	}

	results, err := CollectMachineConfig(cfg, opts)
	if err != nil {
		t.Fatalf("CollectMachineConfig failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].ID != "git" {
		t.Errorf("Expected ID 'git', got %q", results[0].ID)
	}

	if results[0].Values["user_name"] != "Test User" {
		t.Errorf("Expected user_name 'Test User', got %q", results[0].Values["user_name"])
	}

	if results[0].Values["user_email"] != "test@example.com" {
		t.Errorf("Expected user_email 'test@example.com', got %q", results[0].Values["user_email"])
	}
}

func TestCollectSingleConfig(t *testing.T) {
	cfg := &config.Config{
		MachineConfig: []config.MachinePrompt{
			{
				ID:          "git",
				Description: "Git configuration",
				Prompts: []config.PromptField{
					{
						ID:      "name",
						Prompt:  "Name",
						Default: "Test",
					},
				},
			},
			{
				ID:          "other",
				Description: "Other config",
				Prompts:     []config.PromptField{},
			},
		},
	}

	opts := PromptOptions{SkipPrompts: true}

	// Test finding existing config
	result, err := CollectSingleConfig(cfg, "git", opts)
	if err != nil {
		t.Fatalf("CollectSingleConfig failed: %v", err)
	}
	if result.ID != "git" {
		t.Errorf("Expected ID 'git', got %q", result.ID)
	}

	// Test not found
	_, err = CollectSingleConfig(cfg, "nonexistent", opts)
	if err == nil {
		t.Error("Expected error for nonexistent config")
	}
}

func TestGetMachineConfigByID(t *testing.T) {
	cfg := &config.Config{
		MachineConfig: []config.MachinePrompt{
			{ID: "git", Description: "Git config"},
			{ID: "ssh", Description: "SSH config"},
		},
	}

	// Test found
	mc := GetMachineConfigByID(cfg, "git")
	if mc == nil {
		t.Fatal("Expected to find 'git' config")
	}
	if mc.ID != "git" {
		t.Errorf("Expected ID 'git', got %q", mc.ID)
	}

	// Test not found
	mc = GetMachineConfigByID(cfg, "nonexistent")
	if mc != nil {
		t.Error("Expected nil for nonexistent config")
	}
}

func TestListMachineConfigs(t *testing.T) {
	cfg := &config.Config{
		MachineConfig: []config.MachinePrompt{
			{ID: "git", Description: "Git config"},
			{ID: "ssh", Description: "SSH config"},
		},
	}

	list := ListMachineConfigs(cfg)

	if len(list) != 2 {
		t.Fatalf("Expected 2 configs, got %d", len(list))
	}

	if list[0].ID != "git" || list[0].Description != "Git config" {
		t.Errorf("Unexpected first item: %+v", list[0])
	}

	if list[1].ID != "ssh" || list[1].Description != "SSH config" {
		t.Errorf("Unexpected second item: %+v", list[1])
	}
}
