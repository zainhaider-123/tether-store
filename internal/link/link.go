// Package link activates and deactivates skill symlinks into agent directories.
package link

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zainhaider-123/tether-store/internal/agents"
	"github.com/zainhaider-123/tether-store/internal/store"
)

// Activate creates symlinks for skillName into each unique agent destination.
func Activate(skillName string, agentIDs []string, projectRoot string, global bool) error {
	cache, err := store.CachePath(skillName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(cache); err != nil {
		return fmt.Errorf("skill %q not in cache: %w", skillName, err)
	}

	dirs, err := agents.UniqueDestDirs(agentIDs, projectRoot, global)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		linkPath := filepath.Join(dir, skillName)
		if err := replaceSymlink(linkPath, cache); err != nil {
			return fmt.Errorf("link %s: %w", linkPath, err)
		}
	}
	return nil
}

// Deactivate removes symlinks for skillName from the given agent destinations.
func Deactivate(skillName string, agentIDs []string, projectRoot string, global bool) error {
	dirs, err := agents.UniqueDestDirs(agentIDs, projectRoot, global)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		linkPath := filepath.Join(dir, skillName)
		if err := removeIfSymlink(linkPath); err != nil {
			return err
		}
	}
	return nil
}

// IsLinked reports whether skillName is symlinked into all unique destinations for agentIDs.
func IsLinked(skillName string, agentIDs []string, projectRoot string, global bool) (bool, error) {
	cache, err := store.CachePath(skillName)
	if err != nil {
		return false, err
	}
	cache, err = filepath.EvalSymlinks(cache)
	if err != nil {
		return false, nil
	}
	dirs, err := agents.UniqueDestDirs(agentIDs, projectRoot, global)
	if err != nil {
		return false, err
	}
	if len(dirs) == 0 {
		return false, nil
	}
	for _, dir := range dirs {
		linkPath := filepath.Join(dir, skillName)
		target, err := os.Readlink(linkPath)
		if err != nil {
			return false, nil
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(dir, target)
		}
		resolved, err := filepath.EvalSymlinks(target)
		if err != nil || resolved != cache {
			return false, nil
		}
	}
	return true, nil
}

// LinkedAgents returns agent ids whose destination contains a symlink to the cache skill.
// Agents that share a destination directory are all reported if that directory is linked.
func LinkedAgents(skillName string, projectRoot string, global bool) ([]string, error) {
	cache, err := store.CachePath(skillName)
	if err != nil {
		return nil, err
	}
	cache, err = filepath.EvalSymlinks(cache)
	if err != nil {
		// skill not in cache
		return nil, nil
	}

	var linked []string
	seenDir := map[string]bool{}
	for _, a := range agents.Registry {
		dir, err := agents.DestDir(a.ID, projectRoot, global)
		if err != nil {
			continue
		}
		key := dir
		if seenDir[key] {
			// still report agent if symlink exists
		}
		seenDir[key] = true
		linkPath := filepath.Join(dir, skillName)
		target, err := os.Readlink(linkPath)
		if err != nil {
			continue
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(dir, target)
		}
		resolved, err := filepath.EvalSymlinks(target)
		if err != nil {
			continue
		}
		if resolved == cache {
			linked = append(linked, a.ID)
		}
	}
	return linked, nil
}

func replaceSymlink(linkPath, target string) error {
	if fi, err := os.Lstat(linkPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(linkPath); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%s exists and is not a symlink", linkPath)
		}
	}
	return os.Symlink(target, linkPath)
}

func removeIfSymlink(linkPath string) error {
	fi, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is not a symlink; refusing to remove", linkPath)
	}
	return os.Remove(linkPath)
}
