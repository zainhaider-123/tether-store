// Package cli implements the tether cobra command tree.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/zainhaider-123/tether-store/internal/agents"
	"github.com/zainhaider-123/tether-store/internal/config"
	"github.com/zainhaider-123/tether-store/internal/link"
	"github.com/zainhaider-123/tether-store/internal/prompt"
	"github.com/zainhaider-123/tether-store/internal/search"
	"github.com/zainhaider-123/tether-store/internal/source"
	"github.com/zainhaider-123/tether-store/internal/store"
)

// Execute runs the root command.
func Execute() error {
	return rootCmd().Execute()
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "tether",
		Short:         "Agent skill management CLI",
		Long:          "tether installs skills into a shared cache and activates them via hierarchical JSON config + agent symlinks.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(
		useCmd(),
		installCmd(),
		unuseCmd(),
		lsCmd(),
		lsCacheCmd(),
		searchCmd(),
		upgradeCmd(),
	)
	return cmd
}

func cwd() (string, error) {
	return os.Getwd()
}

func useCmd() *cobra.Command {
	var (
		global     bool
		skillName  string
		agentFlag  string
		yes        bool
		installAll bool
	)
	cmd := &cobra.Command{
		Use:     "use <source>",
		Aliases: []string{"u"},
		Short:   "Install a skill into the cache, write config, and activate agent symlinks",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := cwd()
			if err != nil {
				return err
			}
			ref, err := source.Parse(args[0])
			if err != nil {
				return err
			}

			wantAll := installAll || skillName == "*"
			if skillName == "*" {
				skillName = ""
			}
			installed, err := store.Install(ref, skillName, wantAll)
			if err != nil {
				if errors.Is(err, store.ErrMultipleSkills) && !wantAll {
					if yes {
						return err
					}
					chosen, selErr := prompt.SelectSkill(installed)
					if selErr != nil {
						return selErr
					}
					installed, err = store.Install(ref, chosen, false)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}
			if len(installed) == 0 {
				return fmt.Errorf("no skills installed")
			}

			agentIDs, err := resolveAgents(wd, agentFlag, yes)
			if err != nil {
				return err
			}

			cfgPath, err := config.WriteTarget(wd, global)
			if err != nil {
				return err
			}

			for _, name := range installed {
				entry := skillEntryFor(ref, name, agentIDs)
				if err := config.UpsertSkill(cfgPath, name, entry); err != nil {
					return err
				}
				if err := link.Activate(name, agentIDs, wd, global); err != nil {
					return err
				}
				fmt.Printf("✓ activated %s → %s\n", name, cfgPath)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&global, "global", "g", false, "write global config (~/.config/tether/tether.json)")
	cmd.Flags().StringVarP(&skillName, "skill", "s", "", "skill name when repo has multiple")
	cmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "comma-separated agents (or '*'); skips prompt")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "non-interactive; use default agents")
	cmd.Flags().BoolVar(&installAll, "all", false, "install all skills from the source")
	return cmd
}

func skillEntryFor(ref source.Ref, name string, agentIDs []string) config.SkillEntry {
	entry := config.SkillEntry{Agents: agentIDs}
	if strings.HasPrefix(ref.Raw, "http") && (ref.Ref != "" || ref.Subdir != "") {
		entry.GitHub = strings.TrimSuffix(ref.CloneURL(), ".git")
		entry.Ref = ref.Ref
		entry.Subdir = ref.Subdir
		return entry
	}
	entry.Source = ref.Owner + "/" + ref.Repo + "@" + name
	return entry
}

func resolveAgents(wd, agentFlag string, yes bool) ([]string, error) {
	if agentFlag != "" {
		return agents.ParseList(agentFlag)
	}
	merged, err := config.DiscoverAndMerge(wd)
	if err != nil {
		return nil, err
	}
	defaults := merged.DefaultAgents()
	if yes || !prompt.IsTTY() {
		return defaults, nil
	}
	return prompt.SelectAgents(defaults)
}

func installCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install [name…]",
		Aliases: []string{"i"},
		Short:   "Ensure active skills are cached and (re)linked; does not add to config",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := cwd()
			if err != nil {
				return err
			}
			merged, err := config.DiscoverAndMerge(wd)
			if err != nil {
				return err
			}
			names := args
			if len(names) == 0 {
				for n := range merged.Skills {
					names = append(names, n)
				}
			}
			if len(names) == 0 {
				fmt.Println("no active skills in current context")
				return nil
			}

			for _, name := range names {
				entry, ok := merged.Skills[name]
				if !ok {
					return fmt.Errorf("skill %q is not active in current config", name)
				}
				ref, err := source.FromEntry(name, entry)
				if err != nil {
					return err
				}
				okCached, err := store.Exists(name)
				if err != nil {
					return err
				}
				if !okCached {
					if _, err := store.Install(ref, name, false); err != nil {
						return err
					}
					fmt.Printf("✓ cached %s\n", name)
				} else {
					fmt.Printf("· cached %s (already present)\n", name)
				}

				agentIDs := merged.AgentsFor(name)
				// Activate at project level for project-declared skills; global for global-only.
				global := isGlobalOnly(merged, name)
				if err := link.Activate(name, agentIDs, wd, global); err != nil {
					return err
				}
				fmt.Printf("✓ linked %s → %s\n", name, strings.Join(agentIDs, ","))
			}
			return nil
		},
	}
	return cmd
}

func isGlobalOnly(m config.Merged, name string) bool {
	origin, ok := m.SkillOrigins[name]
	if !ok {
		return false
	}
	globalPath, err := config.WriteTarget(".", true)
	if err != nil {
		return false
	}
	return filepath.Clean(origin) == filepath.Clean(globalPath)
}

func unuseCmd() *cobra.Command {
	var global bool
	cmd := &cobra.Command{
		Use:   "unuse <name>",
		Short: "Remove a skill from project or global config and unlink agent symlinks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := cwd()
			if err != nil {
				return err
			}
			name := args[0]
			cfgPath, err := config.WriteTarget(wd, global)
			if err != nil {
				return err
			}

			f, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			entry, ok := f.Skills[name]
			if !ok {
				return fmt.Errorf("skill %q not found in %s", name, cfgPath)
			}
			agentIDs := entry.Agents
			if len(agentIDs) == 0 {
				merged, _ := config.DiscoverAndMerge(wd)
				agentIDs = merged.DefaultAgents()
			}

			if err := config.RemoveSkill(cfgPath, name); err != nil {
				return err
			}
			if err := link.Deactivate(name, agentIDs, wd, global); err != nil {
				return err
			}
			fmt.Printf("✓ removed %s from %s\n", name, cfgPath)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&global, "global", "g", false, "remove from global config")
	return cmd
}

func lsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List active skills in the current context",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := cwd()
			if err != nil {
				return err
			}
			merged, err := config.DiscoverAndMerge(wd)
			if err != nil {
				return err
			}
			if len(merged.Skills) == 0 {
				fmt.Println("no active skills")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tCACHE\tLINKED\tAGENTS\tSOURCE\tCONFIG")
			for name, entry := range merged.Skills {
				cached := "missing"
				if ok, _ := store.Exists(name); ok {
					cached = "ok"
				}
				agentIDs := merged.AgentsFor(name)
				linked := "no"
				if ok, _ := link.IsLinked(name, agentIDs, wd, isGlobalOnly(merged, name)); ok {
					linked = "yes"
				}
				src := entry.Source
				if src == "" {
					src = entry.GitHub
				}
				origin := merged.SkillOrigins[name]
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", name, cached, linked, strings.Join(agentIDs, ","), src, origin)
			}
			return w.Flush()
		},
	}
	return cmd
}

func lsCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls-cache",
		Short: "List all skills in the shared install cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := store.ListCache()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Println("cache is empty")
				return nil
			}
			for _, n := range names {
				p, _ := store.CachePath(n)
				fmt.Printf("%s\t%s\n", n, p)
			}
			return nil
		},
	}
	return cmd
}

func searchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search skills via skills.sh",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := ""
			if len(args) > 0 {
				q = args[0]
			}
			results, err := search.Query(q)
			if err != nil {
				return err
			}
			if len(results) == 0 {
				fmt.Println("no results")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSOURCE\tINSTALLS\tDESCRIPTION")
			for _, r := range results {
				name := r.Name
				if name == "" {
					name = r.ID
				}
				desc := r.Description
				if len(desc) > 60 {
					desc = desc[:57] + "…"
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", name, r.Source, r.Installs, desc)
			}
			return w.Flush()
		},
	}
	return cmd
}

func upgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade [names…]",
		Short: "Re-fetch cached skills for active (or named) skills and refresh symlinks",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := cwd()
			if err != nil {
				return err
			}
			merged, err := config.DiscoverAndMerge(wd)
			if err != nil {
				return err
			}
			names := args
			if len(names) == 0 {
				for n := range merged.Skills {
					names = append(names, n)
				}
			}
			if len(names) == 0 {
				// upgrade everything in cache
				names, err = store.ListCache()
				if err != nil {
					return err
				}
			}
			for _, name := range names {
				var ref *source.Ref
				if entry, ok := merged.Skills[name]; ok {
					r, err := source.FromEntry(name, entry)
					if err == nil {
						ref = &r
					}
				}
				if err := store.Upgrade(name, ref); err != nil {
					return err
				}
				fmt.Printf("✓ upgraded %s\n", name)
				if entry, ok := merged.Skills[name]; ok {
					agentIDs := entry.Agents
					if len(agentIDs) == 0 {
						agentIDs = merged.DefaultAgents()
					}
					global := isGlobalOnly(merged, name)
					if err := link.Activate(name, agentIDs, wd, global); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	return cmd
}
