package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tools/internal/graph"
)

func write(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitInit(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	write(t, root, "AGENTS.md", "# Root\n")
	write(t, root, ".sdd/.sdd-config.json", `{"version": 0, "paths": {"specs": "specs", "src": "src", "technical": "src/specs"}}`+"\n")
	write(t, root, "specs/AGENTS.md", "# P\n")
	return root
}

func load(t *testing.T, root string) *graph.Graph {
	t.Helper()
	g, err := graph.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func sync(t *testing.T, root string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		g := load(t, root)
		m, err := g.Sync(g.Node(rel), time.Unix(0, 0))
		if err != nil {
			t.Fatalf("sync %s: %v", rel, err)
		}
		if _, err := graph.Save(g.Node(rel), m); err != nil {
			t.Fatal(err)
		}
	}
}

func refresh(t *testing.T, root, rel string) {
	t.Helper()
	g := load(t, root)
	m, err := graph.Refresh(g.Node(rel), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	graph.Save(g.Node(rel), m)
}

func approve(t *testing.T, root, rel string) {
	t.Helper()
	g := load(t, root)
	m, err := g.Approve(g.Node(rel), time.Unix(0, 0))
	if err != nil {
		t.Fatalf("approve %s: %v", rel, err)
	}
	graph.Save(g.Node(rel), m)
}

func TestBatchModeNamesDesignWorkWhenNothingCanBeImplemented(t *testing.T) {
	root := gitInit(t)
	write(t, root, "specs/business/invite/spec.md", "# Invite\n\n## S\n\n### R\n\nx\n")
	g := load(t, root)
	out := render(g, true)
	for _, want := range []string{"# Next batch: none", "Nothing to implement; /sdd-design owes:", "- `specs/business/invite` has no technical spec", "# Drift"} {
		if !strings.Contains(out, want) {
			t.Fatalf("batch output lacks %q:\n%s", want, out)
		}
	}
	if full := render(g, false); !strings.Contains(full, "# Design") || strings.Contains(full, "# Next batch") {
		t.Fatalf("full output:\n%s", full)
	}
}

const pSpec = "# Invite\n\nPurpose.\n\n## S\n\n### Invite expiry\n\nx\n"

// auditFixture: one approved code node (src, the root manifests) and one that
// owes an audit (src/backend/invite), with a global spec it does not list.
func auditFixture(t *testing.T) string {
	t.Helper()
	root := gitInit(t)
	write(t, root, "specs/business/invite/spec.md", pSpec)
	write(t, root, "src/specs/toolchain/spec.md", "# Toolchain\n\nThe toolchain of every module.\n\n## Place\n\n### Backend placement\n\nModules live under `src/backend/`.\n")
	write(t, root, "src/specs/toolchain/.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "src/specs/invite/spec.md", "# Invite\n\nHow invites are built.\n\n## HTTP\n\n### Invite endpoint\n\nx\n")
	write(t, root, "src/specs/invite/.metadata.json", `{"version": 1, "sources": [{"path": "specs/business/invite"}]}`+"\n")
	write(t, root, "src/backend/invite/invite.ts", "x\n")
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "specs/business/invite"}]}`+"\n")
	write(t, root, "src/go.mod", "module x\n")
	write(t, root, "src/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/toolchain"}]}`+"\n")
	sync(t, root, "specs/business/invite", "src/specs/toolchain", "src/specs/invite", "src/backend/invite", "src")
	approve(t, root, "src")
	return root
}

func TestAuditSectionAndDossier(t *testing.T) {
	root := auditFixture(t)
	g := load(t, root)
	full := render(g, false)
	if !strings.Contains(full, "No pending work.") || !strings.Contains(full, "# Audit\n\n- `src/backend/invite` owes an audit (/sdd-audit)") || strings.Contains(full, "`src` ") {
		t.Fatalf("full output:\n%s", full)
	}
	if batch := render(g, true); strings.Contains(batch, "# Audit") {
		t.Fatalf("batch output carries the audit section:\n%s", batch)
	}
	targets, err := codeNodes(g, nil)
	if err != nil || len(targets) != 2 || targets[0].Rel != "src" {
		t.Fatalf("targets = %v %v", targets, err)
	}
	out := renderAudit(g, false, targets)
	for _, want := range []string{
		"# Audit\n\n- `src` audited 1; skipped (--all re-audits)\n\n## `src/backend/invite` — audited 0",
		"Files:\n- invite.ts",
		"Listed technical specs:\n- `src/specs/invite`: \"Invite endpoint\"",
		"Listed perspective specs:\n- `specs/business/invite`",
		"Candidates:\n- `src/specs/toolchain` — names this directory; global; The toolchain of every module.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("dossier lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Perspectives to drop") || strings.Contains(out, "# Drift") {
		t.Fatalf("dossier:\n%s", out)
	}
	if _, err := codeNodes(g, []string{"src/specs"}); err == nil {
		t.Fatal("a directory with no code node resolved")
	}
	if abs, err := codeNodes(g, []string{filepath.Join(root, "src", "backend") + "/"}); err != nil || len(abs) != 1 || abs[0].Rel != "src/backend/invite" {
		t.Fatalf("absolute dir: %v %v", abs, err)
	}
	approve(t, root, "src/backend/invite")
	g = load(t, root)
	if full := render(g, false); strings.Contains(full, "# Audit") {
		t.Fatalf("approved node still owed:\n%s", full)
	}
	targets, _ = codeNodes(g, []string{"src/backend"})
	if out := renderAudit(g, false, targets); !strings.Contains(out, "- `src/backend/invite` audited 1; skipped (--all re-audits)") || strings.Contains(out, "## `src/backend/invite`") {
		t.Fatalf("skipped output:\n%s", out)
	}
	if out := renderAudit(g, true, targets); !strings.Contains(out, "## `src/backend/invite` — audited 1") {
		t.Fatalf("--all output:\n%s", out)
	}
	// Parked: content changed and written, a gap names the node.
	write(t, root, "specs/TODO.md", "# TODO\n\n## Audit gap\n\n- **Ghost** (`src/backend/invite`) nothing fits.\n")
	write(t, root, "src/backend/invite/invite.ts", "y\n")
	refresh(t, root, "src/backend/invite")
	g = load(t, root)
	if full := render(g, false); !strings.Contains(full, "- `src/backend/invite` parked: nothing fits (/sdd-design)") {
		t.Fatalf("parked node:\n%s", full)
	}
	targets, _ = codeNodes(g, []string{"src/backend"})
	if out := renderAudit(g, false, targets); !strings.Contains(out, "- `src/backend/invite` parked: nothing fits (/sdd-design)") || strings.Contains(out, "## `src/backend/invite`") {
		t.Fatalf("parked, not judged:\n%s", out)
	}
	if out := renderAudit(g, true, targets); !strings.Contains(out, "## `src/backend/invite` — audited 0\n\nParked: nothing fits (/sdd-design)") {
		t.Fatalf("parked judged under --all:\n%s", out)
	}
	// Blocked wins over parked: content changed again, not written.
	write(t, root, "src/backend/invite/invite.ts", "z\n")
	g = load(t, root)
	if full := render(g, false); !strings.Contains(full, "- `src/backend/invite` owes an audit — blocked: `src/backend/invite/.metadata.json` does not match the content; run metadata --write src/backend/invite") {
		t.Fatalf("blocked node:\n%s", full)
	}
	targets, _ = codeNodes(g, []string{"src/backend"})
	if out := renderAudit(g, true, targets); !strings.Contains(out, "- `src/backend/invite` not judged — blocked:") || strings.Contains(out, "## `src/backend/invite`") {
		t.Fatalf("blocked --all output:\n%s", out)
	}
	// Approved, then its spec changed: after /sdd-implement, never "skipped".
	write(t, root, "specs/TODO.md", "# TODO\n")
	refresh(t, root, "src/backend/invite")
	approve(t, root, "src/backend/invite")
	write(t, root, "specs/business/invite/spec.md", strings.Replace(pSpec, "\nx\n", "\ny\n", 1))
	sync(t, root, "specs/business/invite")
	g = load(t, root)
	targets, _ = codeNodes(g, []string{"src/backend"})
	for _, all := range []bool{false, true} {
		out := renderAudit(g, all, targets)
		if !strings.Contains(out, "- `src/backend/invite` not judged — after /sdd-implement: stale against `specs/business/invite`") || strings.Contains(out, "skipped") || strings.Contains(out, "## `src/backend/invite`") {
			t.Fatalf("approved-then-stale (all=%v):\n%s", all, out)
		}
	}
	if full := render(g, false); strings.Contains(full, "# Audit") {
		t.Fatalf("an approved node is not listed under # Audit while stale:\n%s", full)
	}
}
