package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tools/internal/meta"
)

func openReview(t *testing.T, g *Graph, source string, candidates ...string) {
	t.Helper()
	if _, err := g.ClassifyReview(g.Node(source), "expanded", candidates, time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, c := range candidates {
		if err := g.RecordVerdict(g.Node(source), g.Node(c), "keep", nil, ""); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReviewSupportInvalidatesOnlyDependentCandidates(t *testing.T) {
	root := fixture(t)
	g := settle(t, root)
	const other = "src/other.ts"
	const support = "src/specs/other/spec.md"
	write(t, root, support, "# Other\n\n## Rules\n\n### Other\n\nOther behavior.\n")
	write(t, root, metaOf(support), `{"version":2,"sources":[{"path":"`+P+`"}]}`)
	sync(t, root, support)
	write(t, root, other, "export const other = 1;\n")
	write(t, root, metaOf(other), `{"version":2,"sources":[{"path":"`+support+`"},{"path":"`+P+`"}]}`)
	g = sync(t, root, other)
	openReview(t, g, P, C, other)
	write(t, root, I, strings.Replace(g.Node(I).text, "create one", "create exactly one", 1))
	refresh(t, root, I)
	g = load(t, root)
	st := g.ReviewState(g.Node(P))
	if st.Resolved != 1 || st.Total != 2 {
		t.Fatalf("support change: %+v", st)
	}
	for _, c := range st.Candidates {
		if c.VerdictValid != (c.Candidate.Path == other) {
			t.Fatalf("wrong invalidation: %+v", c)
		}
	}
	if err := g.CloseReview(g.Node(P), time.Now()); err == nil {
		t.Fatal("closed with changed support")
	}
	if _, err := g.ClassifyReview(g.Node(P), "expanded", []string{C, other}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if st := g.ReviewState(g.Node(P)); st.Resolved != 1 {
		t.Fatalf("reclassification reused invalid judgment: %+v", st)
	}
}

func TestReviewRevalidatesAllLinksAndDirectoryAssignments(t *testing.T) {
	for _, dir := range []bool{false, true} {
		t.Run(map[bool]string{false: "file", true: "directory"}[dir], func(t *testing.T) {
			root := fixture(t)
			g := settle(t, root)
			candidate := C
			if dir {
				candidate = "src/assets"
				write(t, root, candidate+"/a.json", "{}")
				write(t, root, candidate+"/b.json", "{}")
				write(t, root, candidate+"/.metadata.json", `{"version":2,"sources":[{"path":"`+I+`"},{"path":"`+T+`"},{"path":"`+P+`"}],"files":{"a.json":["`+I+`"],"b.json":["`+T+`"]}}`)
				g = sync(t, root, candidate)
			}
			openReview(t, g, P, candidate)
			m := g.Node(candidate).Meta
			if dir {
				m.Files["a.json"], m.Files["b.json"] = m.Files["b.json"], m.Files["a.json"]
			} else {
				m.Sources[0].Uses = []string{"Invite endpoint"}
			}
			if _, err := Save(g.Node(candidate), m); err != nil {
				t.Fatal(err)
			}
			g = load(t, root)
			if st := g.ReviewState(g.Node(P)); st.Resolved != 0 {
				t.Fatalf("changed links resolved: %+v", st)
			}
			if err := g.CloseReview(g.Node(P), time.Now()); err == nil {
				t.Fatal("closed with changed links")
			}
		})
	}
}

func TestReviewCloseRejectsCandidateDrift(t *testing.T) {
	root := fixture(t)
	g := settle(t, root)
	openReview(t, g, P, C)
	m := g.Node(C).Meta
	m.Sources = append(m.Sources, meta.Source{Path: T, Uses: []string{"unknown"}})
	if _, err := Save(g.Node(C), m); err != nil {
		t.Fatal(err)
	}
	g = load(t, root)
	if st := g.ReviewState(g.Node(P)); st.Resolved != 0 {
		t.Fatalf("drift counted as resolved: %+v", st)
	}
	if err := g.CloseReview(g.Node(P), time.Now()); err == nil || !strings.Contains(err.Error(), "drift") {
		t.Fatalf("close with candidate drift: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, meta.ReviewRel(P))); err != nil {
		t.Fatal(err)
	}
}

func TestCorruptReviewRequiresRebuildingEvidence(t *testing.T) {
	root := fixture(t)
	g := settle(t, root)
	m := g.Node(P).Meta
	m.Reviewed = g.Node(P).Hash
	if _, err := Save(g.Node(P), m); err != nil {
		t.Fatal(err)
	}
	if _, err := g.ClassifyReview(g.Node(P), "expanded", []string{C}, time.Now()); err != nil {
		t.Fatal(err)
	}
	write(t, root, meta.ReviewRel(P), "{broken")
	g = load(t, root)
	before := g.Node(P).Meta.Reviewed
	if !g.Node(P).OwesReview() {
		t.Fatal("corrupt checkpoint lost review obligation")
	}
	if closed, err := g.ClassifyReview(g.Node(P), "preserved", nil, time.Now()); err == nil || closed {
		t.Fatalf("corrupt checkpoint closed as preserved: %v %v", closed, err)
	}
	if err := g.CloseReview(g.Node(P), time.Now()); err == nil {
		t.Fatal("corrupt checkpoint closed")
	}
	write(t, root, P, pSpec+"\n### New rule\n\nNew obligation.\n")
	refresh(t, root, P)
	g = load(t, root)
	if g.Node(P).Meta.Reviewed != before {
		t.Fatal("write advanced reviewed through corrupt checkpoint")
	}
	if closed, err := g.ClassifyReview(g.Node(P), "unknown", nil, time.Now()); err != nil || closed {
		t.Fatalf("rebuild: %v %v", closed, err)
	}
	g = load(t, root)
	if st := g.ReviewState(g.Node(P)); st.Resolved != 0 || st.Total != 3 {
		t.Fatalf("lost candidates not rebuilt: %+v", st)
	}
	if err := g.CloseReview(g.Node(P), time.Now()); err == nil {
		t.Fatal("rebuilt review closed without judgment")
	}
}

func TestIntegralToSelectiveInitializesReviewed(t *testing.T) {
	for _, approving := range []bool{false, true} {
		t.Run(map[bool]string{false: "sync", true: "approve"}[approving], func(t *testing.T) {
			root := fixture(t)
			g := settle(t, root)
			m := g.Node(C).Meta
			for i := range m.Sources {
				if m.Sources[i].Path == T {
					m.Sources[i].Uses = []string{"Vitest as test runner"}
				}
			}
			if _, err := Save(g.Node(C), m); err != nil {
				t.Fatal(err)
			}
			if approving {
				g = approve(t, root, C)
			} else {
				g = sync(t, root, C)
			}
			if n := g.Node(T); n.Meta.Reviewed != n.Hash || n.OwesReview() {
				t.Fatalf("conversion opened review: %+v", n.Meta)
			}
		})
	}
}

func TestUnknownTechnicalReviewExcludesItself(t *testing.T) {
	root := fixture(t)
	g := settle(t, root)
	if _, err := g.ClassifyReview(g.Node(T), "unknown", nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := g.RecordVerdict(g.Node(T), g.Node(T), "exclude", nil, "self"); err == nil {
		t.Fatal("accepted self judgment")
	}
	for _, candidate := range []string{I, C} {
		if err := g.RecordVerdict(g.Node(T), g.Node(candidate), "keep", nil, ""); err != nil {
			t.Fatal(err)
		}
	}
	g = load(t, root)
	if st := g.ReviewState(g.Node(T)); st.Total != 2 || st.Resolved != 2 {
		t.Fatalf("unknown scope: %+v", st)
	}
	if err := g.CloseReview(g.Node(T), time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestInterruptedApplyAcceptsOnlyPlannedLinks(t *testing.T) {
	for _, extraEdit := range []bool{false, true} {
		t.Run(map[bool]string{false: "expected", true: "unrelated edit"}[extraEdit], func(t *testing.T) {
			root := fixture(t)
			g := settle(t, root)
			if _, err := g.ClassifyReview(g.Node(T), "expanded", []string{C}, time.Now()); err != nil {
				t.Fatal(err)
			}
			if err := g.RecordVerdict(g.Node(T), g.Node(C), "reduce", []string{"Vitest as test runner"}, ""); err != nil {
				t.Fatal(err)
			}
			m := g.Node(C).Meta
			for i := range m.Sources {
				if m.Sources[i].Path == T {
					m.Sources[i].Uses = []string{"Vitest as test runner"}
					m.Sources[i].Hash = ""
				}
				if extraEdit && m.Sources[i].Path == I {
					m.Sources[i].Uses = []string{"Invite endpoint"}
				}
			}
			if _, err := Save(g.Node(C), m); err != nil {
				t.Fatal(err)
			}
			g = load(t, root)
			err := g.ApplyReviewCandidate(g.Node(T), g.Node(C), time.Now())
			if (err != nil) != extraEdit {
				t.Fatalf("resume extraEdit=%v: %v", extraEdit, err)
			}
		})
	}
}
