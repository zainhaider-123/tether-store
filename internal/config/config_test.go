package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zainhaider-123/tether-store/internal/config"
)

func TestDiscoverMergeNearerWins(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	proj := filepath.Join(root, "proj", "app")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "tether"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	global := config.File{
		Version: 1,
		Settings: config.Settings{DefaultAgents: []string{"cursor"}},
		Skills: map[string]config.SkillEntry{
			"a": {Source: "owner/repo@a", Agents: []string{"cursor"}},
			"b": {Source: "owner/repo@b"},
		},
	}
	if err := config.Write(filepath.Join(home, ".config", "tether", "tether.json"), global); err != nil {
		t.Fatal(err)
	}

	project := config.File{
		Version: 1,
		Settings: config.Settings{Agents: []string{"cursor", "agents"}},
		Skills: map[string]config.SkillEntry{
			"a": {Source: "other/repo@a", Agents: []string{"claude-code"}},
		},
	}
	if err := config.Write(filepath.Join(proj, "tether.json"), project); err != nil {
		t.Fatal(err)
	}

	local := config.File{
		Version: 1,
		Skills: map[string]config.SkillEntry{
			"c": {Source: "x/y@c"},
		},
	}
	if err := config.Write(filepath.Join(proj, "tether.local.json"), local); err != nil {
		t.Fatal(err)
	}

	m, err := config.DiscoverAndMerge(proj)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Skills) != 3 {
		t.Fatalf("expected 3 skills, got %d: %#v", len(m.Skills), m.Skills)
	}
	if m.Skills["a"].Source != "other/repo@a" {
		t.Fatalf("project should override global for a: %s", m.Skills["a"].Source)
	}
	if got := m.AgentsFor("a"); len(got) != 1 || got[0] != "claude-code" {
		t.Fatalf("agents for a: %v", got)
	}
	if _, ok := m.Skills["c"]; !ok {
		t.Fatal("missing local skill c")
	}
	defs := m.DefaultAgents()
	if len(defs) != 2 {
		t.Fatalf("default agents: %v", defs)
	}
}
