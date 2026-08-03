// Package store manages the canonical skill install cache.
package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zainhaider-123/tether-store/internal/paths"
	"github.com/zainhaider-123/tether-store/internal/source"
)

// ErrMultipleSkills is returned when a repo has several skills and none was selected.
var ErrMultipleSkills = errors.New("multiple skills found")

// SkillInfo describes a discovered SKILL.md location in a repo checkout.
type SkillInfo struct {
	Name string
	Dir  string // absolute path to the skill directory (contains SKILL.md)
}

// CachePath returns ~/.local/share/tether/skills/<name>.
func CachePath(name string) (string, error) {
	root, err := paths.SkillsCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

// Exists reports whether the skill is present in the cache.
func Exists(name string) (bool, error) {
	p, err := CachePath(name)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(filepath.Join(p, "SKILL.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

// ListCache returns skill names present under the share cache.
func ListCache() ([]string, error) {
	root, err := paths.SkillsCacheDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, e.Name(), "SKILL.md")); err == nil {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Install ensures the skill named skillName is in the cache, cloning from ref as needed.
// If the repo contains multiple skills and skillName is empty, returns the discovered list
// without installing (caller selects). When installAll is true, installs every discovered skill.
func Install(ref source.Ref, skillName string, installAll bool) ([]string, error) {
	cacheRoot, err := paths.SkillsCacheDir()
	if err != nil {
		return nil, err
	}
	if err := paths.EnsureDir(cacheRoot); err != nil {
		return nil, err
	}

	if skillName == "" {
		skillName = ref.Skill
	}

	// Fast path: already cached.
	if skillName != "" {
		ok, err := Exists(skillName)
		if err != nil {
			return nil, err
		}
		if ok {
			return []string{skillName}, nil
		}
	}

	tmp, err := os.MkdirTemp("", "tether-clone-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	repoDir := filepath.Join(tmp, "repo")
	if err := gitClone(ref, repoDir); err != nil {
		return nil, err
	}

	searchRoot := repoDir
	if ref.Subdir != "" {
		searchRoot = filepath.Join(repoDir, filepath.FromSlash(ref.Subdir))
	}

	found, err := DiscoverSkills(searchRoot)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		// If subdir itself is a skill dir we already searched; try repo root as fallback.
		if ref.Subdir != "" {
			found, err = DiscoverSkills(repoDir)
			if err != nil {
				return nil, err
			}
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("no SKILL.md found in %s", ref.Shorthand())
	}

	var selected []SkillInfo
	switch {
	case skillName != "":
		for _, s := range found {
			if s.Name == skillName {
				selected = append(selected, s)
				break
			}
		}
		if len(selected) == 0 {
			names := skillNames(found)
			return nil, fmt.Errorf("skill %q not found; available: %s", skillName, strings.Join(names, ", "))
		}
	case installAll:
		selected = found
	case len(found) == 1:
		selected = found
	default:
		names := skillNames(found)
		return names, fmt.Errorf("%w; pass --skill <name>: %s", ErrMultipleSkills, strings.Join(names, ", "))
	}

	var installed []string
	for _, s := range selected {
		dest := filepath.Join(cacheRoot, s.Name)
		if err := os.RemoveAll(dest); err != nil {
			return installed, err
		}
		if err := copyDir(s.Dir, dest); err != nil {
			return installed, fmt.Errorf("install %s: %w", s.Name, err)
		}
		// Record provenance for upgrades.
		meta := fmt.Sprintf("source=%s\nclone=%s\nref=%s\nsubdir=%s\n",
			ref.DisplaySource(), ref.CloneURL(), ref.Ref, relOrEmpty(repoDir, s.Dir))
		_ = os.WriteFile(filepath.Join(dest, ".tether-source"), []byte(meta), 0o644)
		installed = append(installed, s.Name)
	}
	return installed, nil
}

func relOrEmpty(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

// Upgrade re-fetches a cached skill using its .tether-source metadata, or ref if provided.
func Upgrade(name string, ref *source.Ref) error {
	dest, err := CachePath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dest); err != nil {
		return fmt.Errorf("skill %q not in cache", name)
	}

	var r source.Ref
	if ref != nil {
		r = *ref
	} else {
		r, err = readMeta(dest)
		if err != nil {
			return err
		}
	}
	r.Skill = name
	_, err = Install(r, name, false)
	return err
}

func readMeta(dest string) (source.Ref, error) {
	data, err := os.ReadFile(filepath.Join(dest, ".tether-source"))
	if err != nil {
		return source.Ref{}, fmt.Errorf("no source metadata for %s; re-install with tether use", filepath.Base(dest))
	}
	var cloneURL, src, gitRef, subdir string
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "source":
			src = v
		case "clone":
			cloneURL = v
		case "ref":
			gitRef = v
		case "subdir":
			subdir = v
		}
	}
	var r source.Ref
	if src != "" {
		r, err = source.Parse(src)
		if err != nil && cloneURL != "" {
			r, err = source.Parse(strings.TrimSuffix(cloneURL, ".git"))
		}
	} else if cloneURL != "" {
		r, err = source.Parse(strings.TrimSuffix(cloneURL, ".git"))
	}
	if err != nil {
		return source.Ref{}, err
	}
	if gitRef != "" {
		r.Ref = gitRef
	}
	if subdir != "" && subdir != "." {
		// subdir in meta is the skill dir relative to repo; parent may be the catalog path
		r.Subdir = filepath.Dir(subdir)
		if r.Subdir == "." {
			r.Subdir = ""
		}
	}
	r.Skill = filepath.Base(dest)
	return r, nil
}

func gitClone(ref source.Ref, dest string) error {
	args := []string{"clone", "--depth", "1"}
	if ref.Ref != "" {
		args = append(args, "--branch", ref.Ref)
	}
	args = append(args, ref.CloneURL(), dest)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", ref.CloneURL(), err)
	}
	return nil
}

// DiscoverSkills finds SKILL.md at root or under skills/ (+ one nested catalog level).
func DiscoverSkills(root string) ([]SkillInfo, error) {
	var found []SkillInfo
	seen := map[string]bool{}

	add := func(dir string) {
		name := filepath.Base(dir)
		if seen[name] {
			return
		}
		seen[name] = true
		found = append(found, SkillInfo{Name: name, Dir: dir})
	}

	if hasSkillMD(root) {
		add(root)
		return found, nil
	}

	containers := []string{
		"skills",
		"skills/.curated",
		"skills/.experimental",
		"skills/.system",
		".agents/skills",
		".cursor/skills",
		".claude/skills",
	}
	for _, c := range containers {
		base := filepath.Join(root, filepath.FromSlash(c))
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			dir := filepath.Join(base, e.Name())
			if hasSkillMD(dir) {
				add(dir)
				continue
			}
			// one nested catalog level: skills/<catalog>/<skill>/SKILL.md
			nested, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, n := range nested {
				if !n.IsDir() {
					continue
				}
				nd := filepath.Join(dir, n.Name())
				if hasSkillMD(nd) {
					add(nd)
				}
			}
		}
	}
	return found, nil
}

func skillNames(skills []SkillInfo) []string {
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	return names
}

func hasSkillMD(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !info.IsDir()
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		// Skip .git
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
