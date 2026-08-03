// Package prompt provides interactive checkbox selection for agents.
package prompt

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"

	"github.com/zainhaider-123/tether-store/internal/agents"
)

// IsTTY reports whether stdin is a terminal.
func IsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// SelectAgents shows a multi-select checkbox. Defaults are pre-checked.
// If not a TTY, returns defaults without prompting.
func SelectAgents(defaults []string) ([]string, error) {
	if len(defaults) == 0 {
		defaults = agents.DefaultIDs
	}
	if !IsTTY() {
		return append([]string{}, defaults...), nil
	}

	defSet := map[string]bool{}
	for _, d := range defaults {
		defSet[d] = true
	}

	var options []huh.Option[string]
	for _, a := range agents.Registry {
		opt := huh.NewOption(fmt.Sprintf("%s (%s)", a.Name, a.ID), a.ID)
		if defSet[a.ID] {
			opt = opt.Selected(true)
		}
		options = append(options, opt)
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select agents to activate").
				Options(options...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return append([]string{}, defaults...), nil
	}
	return selected, nil
}

// SelectSkill prompts to pick one skill name from a list.
func SelectSkill(names []string) (string, error) {
	if len(names) == 0 {
		return "", fmt.Errorf("no skills to select")
	}
	if len(names) == 1 {
		return names[0], nil
	}
	if !IsTTY() {
		return "", fmt.Errorf("multiple skills found; pass --skill: %v", names)
	}

	var options []huh.Option[string]
	for _, n := range names {
		options = append(options, huh.NewOption(n, n))
	}
	var chosen string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select skill").
				Options(options...).
				Value(&chosen),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return chosen, nil
}
