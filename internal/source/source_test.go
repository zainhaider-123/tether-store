package source_test

import (
	"testing"

	"github.com/zainhaider-123/tether-store/internal/source"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in    string
		owner string
		repo  string
		skill string
		ref   string
		sub   string
	}{
		{"vercel-labs/agent-skills", "vercel-labs", "agent-skills", "", "", ""},
		{"vercel-labs/agent-skills@frontend-design", "vercel-labs", "agent-skills", "frontend-design", "", ""},
		{"https://github.com/vercel-labs/agent-skills", "vercel-labs", "agent-skills", "", "", ""},
		{
			"https://github.com/vercel-labs/agent-skills/tree/main/skills/frontend-design",
			"vercel-labs", "agent-skills", "frontend-design", "main", "skills/frontend-design",
		},
	}
	for _, tc := range cases {
		r, err := source.Parse(tc.in)
		if err != nil {
			t.Fatalf("%s: %v", tc.in, err)
		}
		if r.Owner != tc.owner || r.Repo != tc.repo {
			t.Fatalf("%s: owner/repo = %s/%s", tc.in, r.Owner, r.Repo)
		}
		if r.Skill != tc.skill {
			t.Fatalf("%s: skill = %q want %q", tc.in, r.Skill, tc.skill)
		}
		if r.Ref != tc.ref {
			t.Fatalf("%s: ref = %q want %q", tc.in, r.Ref, tc.ref)
		}
		if r.Subdir != tc.sub {
			t.Fatalf("%s: subdir = %q want %q", tc.in, r.Subdir, tc.sub)
		}
	}
}
