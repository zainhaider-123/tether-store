// Package config discovers, merges, and writes hierarchical tether.json files.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zainhaider-123/tether-store/internal/paths"
)

const Version = 1

// Settings holds shared and project-level defaults.
type Settings struct {
	DefaultAgents []string `json:"defaultAgents,omitempty"`
	Agents        []string `json:"agents,omitempty"`
}

// SkillEntry declares an active skill in a config layer.
type SkillEntry struct {
	Source  string   `json:"source,omitempty"`
	GitHub  string   `json:"github,omitempty"`
	Ref     string   `json:"ref,omitempty"`
	Subdir  string   `json:"subdir,omitempty"`
	Agents  []string `json:"agents,omitempty"`
}

// File is the on-disk tether.json schema.
type File struct {
	Version  int                   `json:"version"`
	Settings Settings              `json:"settings,omitempty"`
	Skills   map[string]SkillEntry `json:"skills"`
}

// Layer is one discovered config file with its absolute path.
type Layer struct {
	Path   string
	File   File
	Global bool
}

// Merged is the resolved view after walking the directory hierarchy.
type Merged struct {
	Layers   []Layer // farthest → nearest (global first)
	Settings Settings
	Skills   map[string]SkillEntry
	// SkillOrigins maps skill name → config path that last defined it.
	SkillOrigins map[string]string
}

// Empty returns an empty config file with version set.
func Empty() File {
	return File{
		Version: Version,
		Skills:  map[string]SkillEntry{},
	}
}

// Load reads a config file. Missing files return Empty() with no error.
func Load(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Empty(), nil
		}
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.Skills == nil {
		f.Skills = map[string]SkillEntry{}
	}
	if f.Version == 0 {
		f.Version = Version
	}
	return f, nil
}

// Write writes f to path, creating parent directories as needed.
func Write(path string, f File) error {
	if f.Version == 0 {
		f.Version = Version
	}
	if f.Skills == nil {
		f.Skills = map[string]SkillEntry{}
	}
	if err := paths.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// Discover walks from cwd up to / (or home ceiling) collecting configs,
// then appends the global config as the farthest baseline.
// Returned layers are ordered farthest → nearest (global first, cwd last).
func Discover(cwd string) ([]Layer, error) {
	cwd, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}

	var localLayers []Layer // nearer first (cwd → root)
	dir := cwd
	for {
		for _, name := range []string{"tether.local.json", "tether.json"} {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err != nil {
				continue
			}
			f, err := Load(p)
			if err != nil {
				return nil, err
			}
			localLayers = append(localLayers, Layer{Path: p, File: f})
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Reverse so farthest project layers come first, then nearest.
	for i, j := 0, len(localLayers)-1; i < j; i, j = i+1, j-1 {
		localLayers[i], localLayers[j] = localLayers[j], localLayers[i]
	}

	var layers []Layer
	globalPath, err := paths.GlobalConfigPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(globalPath); err == nil {
		gf, err := Load(globalPath)
		if err != nil {
			return nil, err
		}
		layers = append(layers, Layer{Path: globalPath, File: gf, Global: true})
	}

	layers = append(layers, localLayers...)
	return layers, nil
}

// Merge combines layers (farthest → nearest). Nearer entries override.
func Merge(layers []Layer) Merged {
	m := Merged{
		Layers:       layers,
		Skills:       map[string]SkillEntry{},
		SkillOrigins: map[string]string{},
	}
	for _, layer := range layers {
		if len(layer.File.Settings.DefaultAgents) > 0 {
			m.Settings.DefaultAgents = append([]string{}, layer.File.Settings.DefaultAgents...)
		}
		if len(layer.File.Settings.Agents) > 0 {
			m.Settings.Agents = append([]string{}, layer.File.Settings.Agents...)
		}
		for name, entry := range layer.File.Skills {
			m.Skills[name] = entry
			m.SkillOrigins[name] = layer.Path
		}
	}
	return m
}

// DiscoverAndMerge is Discover + Merge for cwd.
func DiscoverAndMerge(cwd string) (Merged, error) {
	layers, err := Discover(cwd)
	if err != nil {
		return Merged{}, err
	}
	return Merge(layers), nil
}

// DefaultAgents returns the effective default agent list from merged settings.
func (m Merged) DefaultAgents() []string {
	if len(m.Settings.Agents) > 0 {
		return m.Settings.Agents
	}
	if len(m.Settings.DefaultAgents) > 0 {
		return m.Settings.DefaultAgents
	}
	return []string{"cursor", "agents"}
}

// AgentsFor returns per-skill agents, falling back to defaults.
func (m Merged) AgentsFor(name string) []string {
	if e, ok := m.Skills[name]; ok && len(e.Agents) > 0 {
		return e.Agents
	}
	return m.DefaultAgents()
}

// ProjectConfigPath returns ./tether.json under cwd.
func ProjectConfigPath(cwd string) (string, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	return filepath.Join(abs, "tether.json"), nil
}

// WriteTarget returns the config path to write for project or global scope.
func WriteTarget(cwd string, global bool) (string, error) {
	if global {
		return paths.GlobalConfigPath()
	}
	return ProjectConfigPath(cwd)
}

// UpsertSkill loads path (or empty), sets skill, and writes.
func UpsertSkill(path, name string, entry SkillEntry) error {
	f, err := Load(path)
	if err != nil {
		return err
	}
	if f.Skills == nil {
		f.Skills = map[string]SkillEntry{}
	}
	f.Skills[name] = entry
	return Write(path, f)
}

// RemoveSkill removes a skill from the config at path.
func RemoveSkill(path, name string) error {
	f, err := Load(path)
	if err != nil {
		return err
	}
	if _, ok := f.Skills[name]; !ok {
		return fmt.Errorf("skill %q not found in %s", name, path)
	}
	delete(f.Skills, name)
	return Write(path, f)
}
