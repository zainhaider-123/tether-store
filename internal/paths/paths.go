// Package paths resolves XDG-style config and data directories for tether.
package paths

import (
	"os"
	"path/filepath"
)

const appName = "tether"

// Home returns the user home directory.
func Home() (string, error) {
	return os.UserHomeDir()
}

// ConfigDir returns ~/.config/tether (or $XDG_CONFIG_HOME/tether).
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appName), nil
	}
	home, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", appName), nil
}

// DataDir returns ~/.local/share/tether (or $XDG_DATA_HOME/tether).
func DataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, appName), nil
	}
	home, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", appName), nil
}

// SkillsCacheDir returns ~/.local/share/tether/skills.
func SkillsCacheDir() (string, error) {
	data, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "skills"), nil
}

// GlobalConfigPath returns ~/.config/tether/tether.json.
func GlobalConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tether.json"), nil
}

// EnsureDir creates dir if it does not exist.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}
