// Package source parses skill source strings into structured references.
package source

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/zainhaider-123/tether-store/internal/config"
)

// Ref is a resolved skill source.
type Ref struct {
	Owner  string
	Repo   string
	Host   string // default github.com
	Ref    string // git ref; empty → default branch
	Subdir string // path within repo to skill or skills tree
	Skill  string // optional skill name hint from @skill
	Raw    string
}

var (
	shorthandRe = regexp.MustCompile(`^([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)(?:@(.+))?$`)
	treeURLRe   = regexp.MustCompile(`^https?://([^/]+)/([^/]+)/([^/]+)/tree/([^/]+)(?:/(.*))?$`)
)

// Parse parses owner/repo, owner/repo@skill, or GitHub URLs.
func Parse(raw string) (Ref, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Ref{}, fmt.Errorf("empty source")
	}

	if m := treeURLRe.FindStringSubmatch(raw); m != nil {
		subdir := strings.Trim(m[5], "/")
		return Ref{
			Host:   m[1],
			Owner:  m[2],
			Repo:   m[3],
			Ref:    m[4],
			Subdir: subdir,
			Skill:  path.Base(subdir),
			Raw:    raw,
		}, nil
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, err := url.Parse(raw)
		if err != nil {
			return Ref{}, err
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			return Ref{}, fmt.Errorf("invalid github url: %s", raw)
		}
		repo := strings.TrimSuffix(parts[1], ".git")
		return Ref{
			Host:  u.Host,
			Owner: parts[0],
			Repo:  repo,
			Raw:   raw,
		}, nil
	}

	if m := shorthandRe.FindStringSubmatch(raw); m != nil {
		r := Ref{
			Host:  "github.com",
			Owner: m[1],
			Repo:  m[2],
			Skill: m[3],
			Raw:   raw,
		}
		return r, nil
	}

	return Ref{}, fmt.Errorf("unrecognized source: %s", raw)
}

// FromEntry builds a Ref from a config skill entry.
func FromEntry(name string, e config.SkillEntry) (Ref, error) {
	if e.Source != "" {
		r, err := Parse(e.Source)
		if err != nil {
			return Ref{}, err
		}
		if e.Ref != "" {
			r.Ref = e.Ref
		}
		if e.Subdir != "" {
			r.Subdir = e.Subdir
		}
		if e.GitHub != "" {
			if gr, err := Parse(e.GitHub); err == nil {
				r.Host, r.Owner, r.Repo = gr.Host, gr.Owner, gr.Repo
			}
		}
		if r.Skill == "" {
			r.Skill = name
		}
		return r, nil
	}
	if e.GitHub != "" {
		r, err := Parse(e.GitHub)
		if err != nil {
			return Ref{}, err
		}
		r.Ref = e.Ref
		r.Subdir = e.Subdir
		r.Skill = name
		return r, nil
	}
	return Ref{}, fmt.Errorf("skill %q has no source", name)
}

// CloneURL returns the git clone URL.
func (r Ref) CloneURL() string {
	host := r.Host
	if host == "" {
		host = "github.com"
	}
	return fmt.Sprintf("https://%s/%s/%s.git", host, r.Owner, r.Repo)
}

// Shorthand returns owner/repo or owner/repo@skill.
func (r Ref) Shorthand() string {
	base := r.Owner + "/" + r.Repo
	if r.Skill != "" {
		return base + "@" + r.Skill
	}
	return base
}

// DisplaySource returns a stable string for config storage.
func (r Ref) DisplaySource() string {
	if r.Raw != "" && !strings.HasPrefix(r.Raw, "http") {
		return r.Raw
	}
	return r.Shorthand()
}
