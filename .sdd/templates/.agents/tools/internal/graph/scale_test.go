package graph

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tools/internal/meta"
)

func TestScale5000PerspectiveConsumers(t *testing.T) {
	if testing.Short() {
		t.Skip("5,000-file scale tests")
	}
	for _, source := range []string{"specs/business/spec.md", P} {
		t.Run(source, func(t *testing.T) {
			root := gitInit(t)
			const technicalA = "src/specs/a/spec.md"
			const technicalB = "src/specs/b/spec.md"
			const body = "# Policy\n\nPurpose.\n\n## Behavior\n\n### Rule A\n\nA v1.\n\n### Rule B\n\nB v1.\n"
			write(t, root, source, body)
			refresh(t, root, source)
			for i, technical := range []string{technicalA, technicalB} {
				use := []string{"Rule A", "Rule B"}[i]
				write(t, root, technical, "# Implementation\n\nPurpose.\n\n## Behavior\n\n### Execute\n\nExecute the policy.\n")
				write(t, root, metaOf(technical), fmt.Sprintf(`{"version":2,"sources":[{"path":%q,"uses":[%q]}]}`, source, use))
				sync(t, root, technical)
			}
			g := load(t, root)
			before := map[string]string{}
			for i := 0; i < 5000; i++ {
				technical, use := technicalB, "Rule B"
				if i < 50 {
					technical, use = technicalA, "Rule A"
				}
				rel := fmt.Sprintf("src/code/f%04d.ts", i)
				content := []byte("export const value = 1;\n")
				write(t, root, rel, string(content))
				audited := 1
				m := &meta.Metadata{Version: meta.Version, Algorithm: meta.Algorithm, Hash: singleFileNodeHash(filepath.Base(rel), content), Modified: meta.Stamp(time.Unix(0, 0)), Audited: &audited,
					Sources: []meta.Source{
						{Path: technical, Hash: g.Node(technical).Hash, Rules: maps.Clone(g.Node(technical).Rules)},
						{Path: source, Uses: []string{use}, Rules: map[string]string{use: g.Node(source).Rules[use]}},
					},
				}
				b, err := meta.Encode(m)
				if err != nil {
					t.Fatal(err)
				}
				write(t, root, metaOf(rel), string(b))
				if i >= 50 {
					before[metaOf(rel)] = string(b)
				}
			}
			b, err := os.ReadFile(filepath.Join(root, metaOf(technicalB)))
			if err != nil {
				t.Fatal(err)
			}
			before[metaOf(technicalB)] = string(b)
			git := gitIn(t, root)
			git("add", "-A")
			git("commit", "-q", "-m", "perspective scale baseline")
			write(t, root, source, strings.Replace(body, "A v1.", "A v2.", 1))
			refresh(t, root, source)
			g = load(t, root)
			if !g.Node(technicalA).Sources[0].Stale() || g.Node(technicalB).Sources[0].Stale() {
				t.Fatal("wrong technical impact")
			}
			pending := 0
			for _, u := range g.Units() {
				pending += len(u.Nodes)
			}
			if pending != 50 {
				t.Fatalf("got %d pending code nodes, want 50", pending)
			}
			if st := g.ReviewState(g.Node(source)); st.Owner != "/sdd-spec" {
				t.Fatalf("wrong owner %s", st.Owner)
			}
			if closed, err := g.ClassifyReview(g.Node(source), "preserved", nil, time.Now()); err != nil || !closed {
				t.Fatalf("classify: %v %v", closed, err)
			}
			git("add", source, metaOf(source))
			nodes, unc, err := g.Staged()
			if err != nil || len(unc) > 0 {
				t.Fatalf("staged: %v %v", err, unc)
			}
			for _, n := range nodes {
				m, err := Refresh(n, time.Now())
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Save(n, m); err != nil {
					t.Fatal(err)
				}
			}
			for rel, want := range before {
				got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
				if err != nil || string(got) != want {
					t.Fatalf("unaffected metadata changed: %s (%v)", rel, err)
				}
			}
			t.Log("50 pending code nodes, one pending technical source; all 4,950 unaffected code metadata files and the other technical source preserved")
		})
	}
}
