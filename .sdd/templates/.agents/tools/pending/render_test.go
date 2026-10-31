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

const (
	P = "specs/business/invite/spec.md"
	T = "src/specs/toolchain/spec.md"
	I = "src/specs/invite/spec.md"
	C = "src/backend/invite/invite.ts"
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
	write(t, root, P, "# Invite\n\n## S\n\n### R\n\nx\n")
	g := load(t, root)
	out := render(g, true)
	for _, want := range []string{"# Next batch: none", "Nothing to implement; /sdd-design owes:", "- `" + P + "` has no technical spec", "# Drift"} {
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
// owes an audit (src/backend/invite/invite.ts), with a global spec it does not list.
func auditFixture(t *testing.T) string {
	t.Helper()
	root := gitInit(t)
	write(t, root, P, pSpec)
	write(t, root, T, "# Toolchain\n\nThe toolchain of every module.\n\n## Place\n\n### Backend placement\n\nModules live under `src/backend/`.\n")
	write(t, root, T+".metadata.json", `{"version": 2}`+"\n")
	write(t, root, I, "# Invite\n\nHow invites are built.\n\n## HTTP\n\n### Invite endpoint\n\nx\n")
	write(t, root, I+".metadata.json", `{"version": 2, "sources": [{"path": "`+P+`"}]}`+"\n")
	write(t, root, C, "x\n")
	write(t, root, C+".metadata.json", `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+P+`"}]}`+"\n")
	write(t, root, "src/go.mod", "module x\n")
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"]}}`+"\n")
	sync(t, root, P, T, I, C, "src")
	approve(t, root, "src")
	return root
}

func TestImplementListsDirectoryUnits(t *testing.T) {
	root := auditFixture(t)
	write(t, root, P, strings.Replace(pSpec, "\nx\n", "\ny\n", 1))
	sync(t, root, P, I)
	g := load(t, root)
	for _, batch := range []bool{false, true} {
		out := render(g, batch)
		if strings.Count(out, "Invite expiry — changed ("+P+")") != 1 || !strings.Contains(out, "invite.ts: stale against `"+P+"`") {
			t.Fatalf("batch=%v must describe each rule once:\n%s", batch, out)
		}
		marker := map[bool]string{false: "- ", true: "1. "}[batch]
		if !strings.Contains(out, "# Next batch: 1 unit\n\n"+marker+"`src/backend/invite/`") {
			t.Fatalf("batch=%v lost the next unit:\n%s", batch, out)
		}
		if strings.Contains(out, "# Implement") == batch {
			t.Fatalf("batch=%v has the wrong backlog view:\n%s", batch, out)
		}
	}
}

func TestBatchKeepsWaitingReasonsWithoutRepeatingTheBacklog(t *testing.T) {
	root := auditFixture(t)
	write(t, root, P, strings.Replace(pSpec, "\nx\n", "\ny\n", 1))
	sync(t, root, P) // invite waits for design
	write(t, root, T, "# Toolchain\n\n## Place\n\n### Backend placement\n\nUpdated placement.\n")
	sync(t, root, T) // root manifests can run
	g := load(t, root)
	full, batch := render(g, false), render(g, true)
	if strings.Count(full, "Invite expiry — changed ("+P+")") != 1 || strings.Contains(full, "Waiting:") {
		t.Fatalf("full view repeated waiting details:\n%s", full)
	}
	if !strings.Contains(batch, "# Next batch: 1 unit") || !strings.Contains(batch, "Waiting:\n- `src/backend/invite/`") {
		t.Fatalf("batch lost runnable or waiting work:\n%s", batch)
	}
	if !strings.Contains(full, "# Next batch: 1 unit\n\n- `src/`") {
		t.Fatalf("full view lost the runnable unit:\n%s", full)
	}
	// With the runnable unit synced, both modes still explain why no batch runs.
	sync(t, root, "src")
	g = load(t, root)
	for _, batch := range []bool{false, true} {
		out := render(g, batch)
		if !strings.Contains(out, "# Next batch: none") || !strings.Contains(out, "`src/backend/invite/`") || !strings.Contains(out, "stale") {
			t.Fatalf("batch=%v lost the blocker:\n%s", batch, out)
		}
	}
}

func TestAuditSectionAndDossier(t *testing.T) {
	root := auditFixture(t)
	g := load(t, root)
	full := render(g, false)
	if !strings.Contains(full, "Nothing stale, dangling, or unconsumed.") || strings.Contains(full, "No pending work.") || !strings.Contains(full, "# Audit\n\n- `"+C+"` owes an audit (/sdd-audit)") || strings.Contains(full, "`src` ") {
		t.Fatalf("full output:\n%s", full)
	}
	if batch := render(g, true); strings.Contains(batch, "# Audit") || strings.Contains(batch, "No pending work.") {
		t.Fatalf("batch output carries the audit section:\n%s", batch)
	}
	targets, err := codeNodes(g, nil)
	if err != nil || len(targets) != 2 || targets[0].Rel != "src" {
		t.Fatalf("targets = %v %v", targets, err)
	}
	out := renderAudit(g, false, targets)
	for _, want := range []string{
		"# Audit\n\n## `src/`\n\n- `src` audited 1; skipped (--all re-audits)\n\n## `src/backend/invite/`\n\n### `" + C + "` — audited 0",
		"File: " + C,
		"Listed technical specs:\n- `" + I + "`: \"Invite endpoint\"",
		"Listed perspective specs:\n- `" + P + "`",
		"Candidates:\n- `" + T + "` — names this node; global; The toolchain of every module.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("dossier lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Perspectives to drop") || strings.Contains(out, "# Drift") {
		t.Fatalf("dossier:\n%s", out)
	}
	if all := renderAudit(g, true, targets); !strings.Contains(all, "### `src` — audited 1\n\nFiles:\n- go.mod → `"+T+"`") {
		t.Fatalf("directory node dossier:\n%s", all)
	}
	if _, err := codeNodes(g, []string{"src/specs"}); err == nil {
		t.Fatal("a directory with no code node resolved")
	}
	if abs, err := codeNodes(g, []string{filepath.Join(root, "src", "backend") + "/"}); err != nil || len(abs) != 1 || abs[0].Rel != C {
		t.Fatalf("absolute dir: %v %v", abs, err)
	}
	if one, err := codeNodes(g, []string{C}); err != nil || len(one) != 1 || one[0].Rel != C {
		t.Fatalf("file node: %v %v", one, err)
	}
	if one, err := codeNodes(g, []string{"src/.metadata.json", C + ".metadata.json"}); err != nil || len(one) != 2 || one[0].Rel != "src" || one[1].Rel != C {
		t.Fatalf("metadata files: %v %v", one, err)
	}
	if _, err := codeNodes(g, []string{I + ".metadata.json"}); err == nil {
		t.Fatal("a technical node's metadata resolved as a code node")
	}
	if one, err := codeNodes(g, []string{C, C + ".metadata.json", filepath.Join(root, C)}); err != nil || len(one) != 1 || one[0].Rel != C {
		t.Fatalf("aliases must select one node: %v %v", one, err)
	}
	if nodes, err := codeNodes(g, []string{"src"}); err != nil || len(nodes) != 2 || nodes[0].Rel != "src" || nodes[1].Rel != C {
		t.Fatalf("directory must select code nodes only: %v %v", nodes, err)
	}
	if _, err := codeNodes(g, []string{"src.metadata.json"}); err == nil {
		t.Fatal("a directory sidecar must not alias its .metadata.json")
	}
	approve(t, root, C)
	g = load(t, root)
	if full := render(g, false); strings.Contains(full, "# Audit") {
		t.Fatalf("approved node still owed:\n%s", full)
	}
	targets, _ = codeNodes(g, []string{"src/backend"})
	if out := renderAudit(g, false, targets); !strings.Contains(out, "- `"+C+"` audited 1; skipped (--all re-audits)") || strings.Contains(out, "### `"+C+"`") {
		t.Fatalf("skipped output:\n%s", out)
	}
	if out := renderAudit(g, true, targets); !strings.Contains(out, "### `"+C+"` — audited 1") {
		t.Fatalf("--all output:\n%s", out)
	}
	// Parked: content changed and written, a gap names the directory.
	write(t, root, "specs/TODO.md", "# TODO\n\n## Audit gap\n\n- **Ghost** (`src/backend/invite`) nothing fits.\n")
	write(t, root, C, "y\n")
	refresh(t, root, C)
	g = load(t, root)
	if full := render(g, false); !strings.Contains(full, "- `"+C+"` parked: nothing fits (/sdd-design)") {
		t.Fatalf("parked node:\n%s", full)
	}
	targets, _ = codeNodes(g, []string{"src/backend"})
	if out := renderAudit(g, false, targets); !strings.Contains(out, "- `"+C+"` parked: nothing fits (/sdd-design)") || strings.Contains(out, "### `"+C+"`") {
		t.Fatalf("parked, not judged:\n%s", out)
	}
	if out := renderAudit(g, true, targets); !strings.Contains(out, "### `"+C+"` — audited 0\n\nParked: nothing fits (/sdd-design)") {
		t.Fatalf("parked judged under --all:\n%s", out)
	}
	// Blocked wins over parked: content changed again, not written.
	write(t, root, C, "z\n")
	g = load(t, root)
	if full := render(g, false); !strings.Contains(full, "- `"+C+"` owes an audit — blocked: `"+C+".metadata.json` does not match the content; run metadata --write "+C) {
		t.Fatalf("blocked node:\n%s", full)
	}
	targets, _ = codeNodes(g, []string{"src/backend"})
	if out := renderAudit(g, true, targets); !strings.Contains(out, "- `"+C+"` not judged — blocked:") || strings.Contains(out, "### `"+C+"`") {
		t.Fatalf("blocked --all output:\n%s", out)
	}
	// Approved, then its spec changed: after /sdd-implement, never "skipped".
	write(t, root, "specs/TODO.md", "# TODO\n")
	refresh(t, root, C)
	approve(t, root, C)
	write(t, root, P, strings.Replace(pSpec, "\nx\n", "\ny\n", 1))
	sync(t, root, P)
	g = load(t, root)
	targets, _ = codeNodes(g, []string{"src/backend"})
	for _, all := range []bool{false, true} {
		out := renderAudit(g, all, targets)
		if !strings.Contains(out, "- `"+C+"` not judged — after /sdd-implement: stale against `"+P+"`") || strings.Contains(out, "skipped") || strings.Contains(out, "### `"+C+"`") {
			t.Fatalf("approved-then-stale (all=%v):\n%s", all, out)
		}
	}
	if full := render(g, false); strings.Contains(full, "# Audit") {
		t.Fatalf("an approved node is not listed under # Audit while stale:\n%s", full)
	}
}

func TestNoPendingWorkOnlyWhenNothingIsOwed(t *testing.T) {
	root := gitInit(t)
	if out := render(load(t, root), false); out != "No pending work." {
		t.Fatalf("empty project:\n%s", out)
	}
	write(t, root, "src/loose.ts", "export {};\n")
	if out := render(load(t, root), false); !strings.HasPrefix(out, "Nothing stale, dangling, or unconsumed.\n# Drift") {
		t.Fatalf("project with drift only:\n%s", out)
	}
}
