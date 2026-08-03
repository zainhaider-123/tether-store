// Package agents maps agent ids to project and global skill directories.
package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Agent describes install destinations for one agent id.
type Agent struct {
	ID          string
	Name        string
	ProjectPath string // relative to project root
	GlobalPath  string // relative to home; empty = project-only
}

// DefaultIDs are pre-checked in the agent prompt.
var DefaultIDs = []string{"cursor", "agents"}

// Registry is the full agent table (skills.sh / vercel-labs/skills subset + agents).
var Registry = []Agent{
	{ID: "agents", Name: "Agents (.agents)", ProjectPath: ".agents/skills/", GlobalPath: ".agents/skills/"},
	{ID: "cursor", Name: "Cursor", ProjectPath: ".cursor/skills/", GlobalPath: ".cursor/skills/"},
	{ID: "claude-code", Name: "Claude Code", ProjectPath: ".claude/skills/", GlobalPath: ".claude/skills/"},
	{ID: "codex", Name: "Codex", ProjectPath: ".agents/skills/", GlobalPath: ".codex/skills/"},
	{ID: "opencode", Name: "OpenCode", ProjectPath: ".agents/skills/", GlobalPath: ".config/opencode/skills/"},
	{ID: "windsurf", Name: "Windsurf", ProjectPath: ".windsurf/skills/", GlobalPath: ".codeium/windsurf/skills/"},
	{ID: "continue", Name: "Continue", ProjectPath: ".continue/skills/", GlobalPath: ".continue/skills/"},
	{ID: "github-copilot", Name: "GitHub Copilot", ProjectPath: ".agents/skills/", GlobalPath: ".copilot/skills/"},
	{ID: "gemini-cli", Name: "Gemini CLI", ProjectPath: ".agents/skills/", GlobalPath: ".gemini/skills/"},
	{ID: "cline", Name: "Cline", ProjectPath: ".agents/skills/", GlobalPath: ".agents/skills/"},
	{ID: "roo", Name: "Roo Code", ProjectPath: ".roo/skills/", GlobalPath: ".roo/skills/"},
	{ID: "amp", Name: "Amp", ProjectPath: ".agents/skills/", GlobalPath: ".config/agents/skills/"},
	{ID: "goose", Name: "Goose", ProjectPath: ".goose/skills/", GlobalPath: ".config/goose/skills/"},
	{ID: "crush", Name: "Crush", ProjectPath: ".crush/skills/", GlobalPath: ".config/crush/skills/"},
	{ID: "aider-desk", Name: "AiderDesk", ProjectPath: ".aider-desk/skills/", GlobalPath: ".aider-desk/skills/"},
	{ID: "augment", Name: "Augment", ProjectPath: ".augment/skills/", GlobalPath: ".augment/skills/"},
	{ID: "droid", Name: "Droid", ProjectPath: ".factory/skills/", GlobalPath: ".factory/skills/"},
	{ID: "kiro-cli", Name: "Kiro CLI", ProjectPath: ".kiro/skills/", GlobalPath: ".kiro/skills/"},
	{ID: "kilo", Name: "Kilo Code", ProjectPath: ".kilocode/skills/", GlobalPath: ".kilocode/skills/"},
	{ID: "trae", Name: "Trae", ProjectPath: ".trae/skills/", GlobalPath: ".trae/skills/"},
	{ID: "zencoder", Name: "Zencoder", ProjectPath: ".zencoder/skills/", GlobalPath: ".zencoder/skills/"},
	{ID: "neovate", Name: "Neovate", ProjectPath: ".neovate/skills/", GlobalPath: ".neovate/skills/"},
	{ID: "pochi", Name: "Pochi", ProjectPath: ".pochi/skills/", GlobalPath: ".pochi/skills/"},
	{ID: "openclaw", Name: "OpenClaw", ProjectPath: "skills/", GlobalPath: ".openclaw/skills/"},
}

var byID map[string]Agent

func init() {
	byID = make(map[string]Agent, len(Registry))
	for _, a := range Registry {
		byID[a.ID] = a
	}
}

// Get returns an agent by id.
func Get(id string) (Agent, bool) {
	a, ok := byID[id]
	return a, ok
}

// AllIDs returns every registered agent id.
func AllIDs() []string {
	ids := make([]string, len(Registry))
	for i, a := range Registry {
		ids[i] = a.ID
	}
	return ids
}

// ParseList splits a comma-separated agent list. "*" expands to all.
func ParseList(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if s == "*" {
		return AllIDs(), nil
	}
	parts := strings.Split(s, ",")
	var out []string
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "*" {
			return AllIDs(), nil
		}
		if _, ok := byID[p]; !ok {
			return nil, fmt.Errorf("unknown agent %q", p)
		}
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out, nil
}

// DestDir returns the absolute skill directory for an agent (parent of the skill symlink).
func DestDir(id, projectRoot string, global bool) (string, error) {
	a, ok := byID[id]
	if !ok {
		return "", fmt.Errorf("unknown agent %q", id)
	}
	if global {
		if a.GlobalPath == "" {
			return "", fmt.Errorf("agent %q does not support global install", id)
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, filepath.FromSlash(strings.TrimSuffix(a.GlobalPath, "/"))), nil
	}
	return filepath.Join(projectRoot, filepath.FromSlash(strings.TrimSuffix(a.ProjectPath, "/"))), nil
}

// UniqueDestDirs returns unique absolute destination directories for the given agents.
func UniqueDestDirs(ids []string, projectRoot string, global bool) ([]string, error) {
	seen := map[string]bool{}
	var dirs []string
	for _, id := range ids {
		dir, err := DestDir(id, projectRoot, global)
		if err != nil {
			return nil, err
		}
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return dirs, nil
}
