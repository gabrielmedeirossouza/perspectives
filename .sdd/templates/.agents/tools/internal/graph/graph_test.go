package graph

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tools/internal/meta"
)

// The nodes of the fixture: a perspective spec, a global technical spec, a
// technical spec that serves the perspective, and a code file.
const (
	P = "specs/business/invite/spec.md"
	T = "src/specs/toolchain/spec.md"
	I = "src/specs/invite/spec.md"
	C = "src/backend/invite/invite.ts"
)

func metaOf(rel string) string { return rel + meta.Suffix }

func write(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
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
	write(t, root, "specs/AGENTS.md", "# Perspectives\n")
	write(t, root, "src/AGENTS.md", "# Source\n")
	return root
}

const pSpec = "# Invite\n\nPurpose.\n\n## Expiry\n\n### Invite expiry\n\nThe invite **must** expire in 7 days.\n\n### Single use\n\nAn invite **must** be used once.\n"

func fixture(t *testing.T) string {
	t.Helper()
	root := gitInit(t)
	write(t, root, "specs/business/AGENTS.md", "# Business\n")
	write(t, root, P, pSpec)
	write(t, root, T, "# Toolchain\n\nPurpose.\n\n## Tests\n\n### Vitest as test runner\n\nTests **must** run on `vitest`.\n")
	write(t, root, metaOf(T), `{"version": 2}`+"\n")
	write(t, root, I, "# Invite\n\nPurpose.\n\n## HTTP\n\n### Invite endpoint\n\n`POST /invites` **must** create one. Serves **Invite expiry**.\n")
	write(t, root, metaOf(I), `{"version": 2, "sources": [{"path": "`+P+`"}, {"path": "`+T+`"}]}`+"\n")
	write(t, root, C, "export const days = 7;\n")
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+T+`"}, {"path": "`+P+`"}]}`+"\n")
	return root
}

func load(t *testing.T, root string) *Graph {
	t.Helper()
	g, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func sync(t *testing.T, root, rel string) *Graph {
	t.Helper()
	g := load(t, root)
	n := g.Node(rel)
	if n == nil {
		t.Fatalf("no node %s", rel)
	}
	m, err := g.Sync(n, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("sync %s: %v", rel, err)
	}
	if _, err := Save(n, m); err != nil {
		t.Fatal(err)
	}
	return load(t, root)
}

func settle(t *testing.T, root string) *Graph {
	t.Helper()
	var g *Graph
	for _, rel := range []string{P, T, I, C} {
		g = sync(t, root, rel)
	}
	return g
}

func has(lines []string, want string) bool {
	for _, l := range lines {
		if strings.Contains(l, want) {
			return true
		}
	}
	return false
}

func TestUnsettledFixture(t *testing.T) {
	g := load(t, fixture(t))
	if len(g.Nodes) != 4 {
		t.Fatalf("nodes = %d", len(g.Nodes))
	}
	drift := g.Drift()
	if !has(drift, "`"+P+"` has no spec.md.metadata.json; run metadata --write "+P) || !has(drift, "`"+metaOf(I)+"` does not match") {
		t.Fatalf("drift = %v", drift)
	}
	design := g.Design()
	if !has(design, "`"+I+"` is stale against `"+P+"`: added \"Invite expiry\", \"Single use\"") {
		t.Fatalf("design = %v", design)
	}
	units := g.Units()
	if len(units) != 1 || units[0].Dir != "src/backend/invite" || units[0].Wait == "" || units[0].Rel() != "src/backend/invite/" {
		t.Fatalf("units = %+v", units)
	}
	if run, wait := g.Batch(5); len(run) != 0 || len(wait) != 1 {
		t.Fatalf("batch run=%d wait=%d", len(run), len(wait))
	}
}

func TestSettledFixtureIsQuiet(t *testing.T) {
	g := settle(t, fixture(t))
	if d := g.Drift(); len(d) > 0 {
		t.Fatalf("drift = %v", d)
	}
	if d := g.Design(); len(d) > 0 {
		t.Fatalf("design = %v", d)
	}
	if u := g.Units(); len(u) > 0 {
		t.Fatalf("units = %+v", u)
	}
	c := g.Node(C)
	if c.Meta.Sources[2].Rules["Invite expiry"] == "" || c.Meta.Sources[0].Hash != g.Node(I).Hash {
		t.Fatalf("sync did not record sources: %+v", c.Meta.Sources)
	}
	if n := g.Node(P); n.MetaRel != metaOf(P) || n.IsDir || !n.HasFile || n.Label() != "spec.md" || n.UnitDir() != "specs/business/invite" {
		t.Fatalf("spec node = %+v", n)
	}
}

func TestSpecHashIsTheHashOfItsOneEntry(t *testing.T) {
	g := load(t, fixture(t))
	if want := meta.Sum([]byte("spec.md\x00" + meta.Sum([]byte(pSpec)) + "\n")); g.Node(P).Hash != want {
		t.Fatalf("hash = %s, want %s", g.Node(P).Hash, want)
	}
}

func TestPolicyChangeReachesDesignThenCode(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, P, strings.Replace(pSpec, "7 days", "14 days", 1))
	g := load(t, root)
	if !has(g.Design(), "`"+I+"` is stale against `"+P+"`: changed \"Invite expiry\"") {
		t.Fatalf("design = %v", g.Design())
	}
	if !has(g.Design(), "— waits: `"+P+"` has drift") {
		t.Fatalf("design does not wait on the unwritten metadata of %s: %v", P, g.Design())
	}
	units := g.Units()
	if len(units) != 1 || units[0].Dir != "src/backend/invite" || !strings.Contains(units[0].Wait, "`"+P+"` has drift") {
		t.Fatalf("units = %+v", units)
	}
	if r := units[0].Rules; len(r) != 1 || r[0] != (RuleState{Title: "Invite expiry", State: "changed", Source: P}) {
		t.Fatalf("rules = %+v", r)
	}
	// /sdd-spec writes the metadata, /sdd-design acknowledges without changing the technical spec.
	sync(t, root, P)
	g = sync(t, root, I)
	if d := g.Design(); len(d) > 0 {
		t.Fatalf("design = %v", d)
	}
	run, wait := g.Batch(5)
	if len(run) != 1 || len(wait) != 0 || run[0].Dir != "src/backend/invite" || run[0].Reason != "invite.ts: stale against `"+P+"`" {
		t.Fatalf("run = %+v wait = %+v", run, wait)
	}
	if len(run[0].Nodes) != 1 || run[0].Nodes[0].Node.Rel != C || run[0].Nodes[0].Wait != "" {
		t.Fatalf("nodes = %+v", run[0].Nodes)
	}
	g = sync(t, root, C)
	if u := g.Units(); len(u) > 0 {
		t.Fatalf("units after sync = %+v", u)
	}
}

func TestRenameAndRemove(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, P, strings.Replace(pSpec, "### Single use", "### One use", 1))
	g := load(t, root)
	if !has(g.Design(), `renamed "Single use" → "One use"`) {
		t.Fatalf("design = %v", g.Design())
	}
	write(t, root, P, pSpec[:strings.Index(pSpec, "### Single use")])
	g = load(t, root)
	if !has(g.Design(), `removed "Single use"`) {
		t.Fatalf("design = %v", g.Design())
	}
	if u := g.Units(); len(u) != 1 || u[0].Rules[0].State != "removed" {
		t.Fatalf("units = %+v", u)
	}
}

func TestLinkToARemovedOrRenamedTitleIsDrift(t *testing.T) {
	root := fixture(t)
	iSpec := "# Invite\n\nPurpose.\n\n## HTTP\n\n### Invite endpoint\n\n`POST /invites` **must** create one. Serves [Invite expiry](../../../specs/business/invite/spec.md#invite-expiry) and [Single use](../../../" + P + "#single-use); see [HTTP](#http), [the guide](https://example.com/x#y), and `[code](gone.md)`.\n"
	write(t, root, I, iSpec)
	g := settle(t, root)
	if d := g.Drift(); len(d) > 0 {
		t.Fatalf("drift with every link resolving = %v", d)
	}
	write(t, root, P, strings.Replace(pSpec, "### Single use", "### One use", 1))
	g = sync(t, root, P)
	if d := g.NodeDrift(g.Node(I)); len(d) != 1 || !has(d, "`"+I+"` links `../../../"+P+"#single-use`, but `"+P+"` has no heading `#single-use`") || !has(d, "(/sdd-design)") {
		t.Fatalf("drift of %s after a rename = %v", I, d)
	}
	if _, err := g.Sync(g.Node(I), time.Now()); err == nil || !strings.Contains(err.Error(), "has no heading `#single-use`") {
		t.Fatalf("sync of %s with a broken link of its own = %v", I, err)
	}
	write(t, root, P, "# Invite\n\nPurpose.\n\nSee [the toolchain](../../../src/specs/missing/spec.md), [up](../../../../x.md), [case](../../../SRC/specs/toolchain/spec.md), [root](/src/specs/toolchain/spec.md?plain=1#toolchain), and [tests](/"+T+"#tests).\n")
	g = load(t, root)
	if _, err := g.Sync(g.Node(P), time.Now()); err == nil {
		t.Fatalf("sync of %s with broken links of its own went through", P)
	}
	if m, err := Refresh(g.Node(P), time.Now()); err != nil {
		t.Fatal(err)
	} else if _, err := Save(g.Node(P), m); err != nil {
		t.Fatal(err)
	}
	g = load(t, root)
	d := g.NodeDrift(g.Node(P))
	if len(d) != 3 || !has(d, "`src/specs/missing/spec.md` does not exist; fix the link (/sdd-spec)") || !has(d, "leaves the repository") || !has(d, "`SRC/specs/toolchain/spec.md` does not exist") {
		t.Fatalf("drift of %s = %v", P, d)
	}
	if _, err := g.Sync(g.Node(I), time.Now()); err == nil || strings.Contains(err.Error(), "`"+P+"` has drift") {
		t.Fatalf("sync of %s: refused for its own link only, not for the drift of %s: %v", I, P, err)
	}
}

func TestBrokenLinkOfASourceRefusesNoSyncOfItsConsumer(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, I, "# Invite\n\nPurpose.\n\n## HTTP\n\n### Invite endpoint\n\n`POST /invites` **must** create one. Serves [Invite expiry](../../../"+P+"#gone).\n")
	g := load(t, root)
	m, err := Refresh(g.Node(I), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Save(g.Node(I), m); err != nil {
		t.Fatal(err)
	}
	g = load(t, root)
	if !has(g.NodeDrift(g.Node(I)), "has no heading `#gone`") {
		t.Fatalf("drift of %s = %v", I, g.NodeDrift(g.Node(I)))
	}
	if _, err := g.Sync(g.Node(C), time.Now()); err != nil {
		t.Fatalf("sync of %s refused for a broken link of its source: %v", C, err)
	}
}

func TestDanglingSource(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	if err := os.RemoveAll(filepath.Join(root, "specs/business/invite")); err != nil {
		t.Fatal(err)
	}
	g := load(t, root)
	if !has(g.Design(), "`"+I+"` has a dangling source `"+P+"`") {
		t.Fatalf("design = %v", g.Design())
	}
	u := g.Units()
	if len(u) != 1 || !strings.Contains(u[0].Reason, "dangling source `"+P+"`") || !strings.Contains(u[0].Wait, "dangling") {
		t.Fatalf("units = %+v", u)
	}
	if _, err := g.Sync(g.Node(I), time.Now()); err == nil {
		t.Fatal("sync accepted a dangling source")
	}
}

func TestDirectorySourceIsDanglingWithTheSpecFileHint(t *testing.T) {
	root := fixture(t)
	write(t, root, metaOf(I), `{"version": 2, "sources": [{"path": "specs/business/invite"}, {"path": "`+T+`"}]}`+"\n")
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "src/specs/invite"}, {"path": "`+T+`"}, {"path": "`+P+`"}]}`+"\n")
	for _, rel := range []string{P, T} {
		sync(t, root, rel)
	}
	g := load(t, root)
	hint := " (a source names a spec file: `"
	if !has(g.Design(), "`"+I+"` has a dangling source `specs/business/invite`"+hint+P+"`)") {
		t.Fatalf("design = %v", g.Design())
	}
	u := g.Units()
	if len(u) != 2 || u[0].Dir != "src/backend/invite" || !strings.Contains(u[0].Reason, "dangling source `src/specs/invite`"+hint+I+"`)") || !strings.Contains(u[1].Wait, "dangling source `specs/business/invite`"+hint+P+"`) (/sdd-design)") {
		t.Fatalf("units = %+v", u)
	}
	if _, err := g.Sync(g.Node(C), time.Now()); err == nil || !strings.Contains(err.Error(), hint+I+"`); drop it from sources") {
		t.Fatalf("sync = %v", err)
	}
	if _, err := g.Sync(g.Node(I), time.Now()); err == nil || !strings.Contains(err.Error(), hint+P+"`)") {
		t.Fatalf("sync = %v", err)
	}
}

func TestCoverageOfCodeNode(t *testing.T) {
	root := fixture(t)
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+T+`"}]}`+"\n")
	for _, rel := range []string{P, T, I} {
		sync(t, root, rel)
	}
	g := load(t, root)
	if u := g.Units(); len(u) != 1 || !strings.Contains(u[0].Reason, "missing source `"+P+"`") || has(g.Drift(), "but not") {
		t.Fatalf("units = %+v drift = %v", u, g.Drift())
	}
	if _, err := g.Sync(g.Node(C), time.Now()); err == nil {
		t.Fatal("sync accepted a code node missing a served perspective")
	}
	write(t, root, "specs/user/home/spec.md", "# Home\n\n## S\n\n### Empty list\n\nShows nothing.\n")
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+T+`"}, {"path": "`+P+`"}, {"path": "specs/user/home/spec.md"}]}`+"\n")
	g = load(t, root)
	if !has(g.Drift(), "`"+C+"` lists `specs/user/home/spec.md`, but none of its technical specs serves it") || !has(g.Design(), "`specs/user/home/spec.md` has no technical spec") {
		t.Fatalf("drift = %v design = %v", g.Drift(), g.Design())
	}
}

func TestInvalidEdgesStraysOrphansAndUncovered(t *testing.T) {
	root := gitInit(t)
	write(t, root, "specs/business/a/spec.md", "# A\n\n## S\n\n### R\n\nx\n")
	write(t, root, "specs/business/a/spec.md.metadata.json", `{"version": 2, "sources": [{"path": "src/specs/t/spec.md"}]}`+"\n")
	write(t, root, "specs/business/a/notes.md", "stray\n")
	write(t, root, "specs/business/b/spec.md.metadata.json", `{"version": 2}`+"\n")
	write(t, root, "specs/business/c/spec.md", "# C\n")
	write(t, root, "specs/business/c/.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "specs/business/d/x.md", "# D\n")
	write(t, root, "specs/business/d/x.md.metadata.json", `{"version": 2}`+"\n")
	write(t, root, "src/specs/t/spec.md", "# T\n\n## S\n\n### R\n\nx\n")
	write(t, root, "src/specs/t/spec.md.metadata.json", `{"version": 2, "sources": [{"path": "src/code/x.ts"}, {"path": "src/specs/t/spec.md"}]}`+"\n")
	write(t, root, "src/specs/old/spec.md", "# Old\n")
	write(t, root, "src/specs/old/.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "src/code/x.ts", "x\n")
	write(t, root, "src/code/x.ts.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "src/code/z.ts.metadata.json.metadata.json", `{"version": 2}`+"\n")
	write(t, root, "src/loose/y.ts", "x\n")
	write(t, root, "src/README.md", "fine\n")
	write(t, root, "specs/spec.md", "# Chrome\n")
	write(t, root, "src/AGENTS.md.metadata.json", `{"version": 2}`+"\n")
	write(t, root, "src/code.metadata.json", `{"version": 2}`+"\n")
	write(t, root, "src/code/.metadata.json", `{"version": 2, "sources": [{"path": "src/specs/t/spec.md"}], "files": {"tsconfig.json": ["src/specs/t/spec.md"]}}`+"\n")
	write(t, root, "src/code/tsconfig.json", "{}\n")
	write(t, root, "src/specs/dup/spec.md", "# Dup\n\n### Same\n\na\n\n### same\n\nb\n")
	write(t, root, "src/specs/dup/spec.md.metadata.json", `{"version": 2}`+"\n")
	write(t, root, "src/specs/helper/helper.ts", "x\n")
	write(t, root, "src/specs/helper/helper.ts.metadata.json", `{"version": 2}`+"\n")
	write(t, root, "src/specs/t/notes.md", "stray\n")
	g := load(t, root)
	d := g.Drift()
	for _, want := range []string{
		"`src/AGENTS.md.metadata.json` is no node: `src/AGENTS.md` is chrome; delete it",
		"`src/code.metadata.json` is no node: `src/code` is a directory, whose metadata is `src/code/.metadata.json`; delete it",
		"`src/specs/dup/spec.md`: duplicate rule title",
		"`src/specs/helper/helper.ts` is no spec.md; a technical node is spec.md; delete its metadata",
		"`src/specs/t/notes.md` is in no node: a spec directory holds spec.md and spec.md.metadata.json",
		"`specs/business/a/spec.md` (perspective) lists `src/specs/t/spec.md` (technical)",
		"`specs/business/a/notes.md` is in no node: a spec directory holds spec.md and spec.md.metadata.json",
		"`specs/business/b/spec.md.metadata.json` has no spec.md beside it; delete it, or rename it with its file",
		"`specs/business/c/.metadata.json` is no node: the node of a spec is spec.md.metadata.json",
		"`specs/business/c/spec.md` has no spec.md.metadata.json; run metadata --write specs/business/c/spec.md",
		"`specs/business/d/x.md` is no spec.md; a perspective node is spec.md; delete its metadata",
		"`src/specs/t/spec.md` (technical) lists `src/code/x.ts` (code)",
		"`src/specs/t/spec.md` lists itself",
		"`src/specs/old/.metadata.json` is no node: the node of a spec is spec.md.metadata.json",
		"`src/specs/old/spec.md` has no spec.md.metadata.json; write its skeleton `{\"version\": 2, \"sources\": [...]}` (/sdd-design)",
		"`src/code/x.ts.metadata.json`: version 1 is not supported; this tool writes version 2",
		"`src/code/z.ts.metadata.json.metadata.json` is no node: a metadata file has no metadata; delete it",
		"`src/loose/y.ts` is in no node: give it y.ts.metadata.json, or link it in src/loose/.metadata.json",
	} {
		if !has(d, want) {
			t.Errorf("drift lacks %q:\n%s", want, strings.Join(d, "\n"))
		}
	}
	if all := g.Loose(nil); len(all) != 8 || !has(all, "`src/loose/y.ts` is in no node") || !has(all, "`specs/business/c/.metadata.json` is no node") {
		t.Errorf("Loose(nil) = %v", all)
	}
	if scoped := g.Loose([]string{"src/loose", "src/code"}); len(scoped) != 2 || !has(scoped, "`src/loose/y.ts`") || !has(scoped, "`src/code/z.ts.metadata.json.metadata.json`") {
		t.Errorf("Loose(src/loose, src/code) = %v", scoped)
	}
	if none := g.Loose([]string{"src/lo"}); len(none) != 0 {
		t.Errorf("Loose(src/lo) matched a sibling by prefix: %v", none)
	}
	if has(d, "README") || has(d, "`src/AGENTS.md` is in no node") || has(d, "`specs/spec.md` is in no node") {
		t.Errorf("chrome reported: %v", d)
	}
	if g.Node("src/specs/old") != nil || g.Node("specs/business/c") != nil || g.Node("src/code/z.ts.metadata.json") != nil || g.Node("specs/spec.md") != nil || g.Node("src/AGENTS.md") != nil {
		t.Fatal("a stray metadata file or chrome became a node")
	}
	if n := g.Node("src/code"); n == nil || !n.IsDir || len(n.Files()) != 1 || has(d, "`src/code/tsconfig.json` is in no node") {
		t.Fatalf("the directory node lost to the stray beside it: %+v drift=%v", n, d)
	}
	for _, u := range g.Units() {
		if u.Node != nil && (u.Node.Rel == "src/specs/old/spec.md" || u.Node.Rel == "src/specs/dup/spec.md" || u.Node.Rel == "src/specs/helper/helper.ts") {
			t.Fatalf("a malformed technical node is no unit: %+v", u)
		}
	}
	if _, err := Refresh(g.Node("src/specs/old/spec.md"), time.Now()); err == nil || !strings.Contains(err.Error(), "write its skeleton") {
		t.Fatalf("refresh of a technical spec without skeleton: %v", err)
	}
	for _, rel := range []string{"specs/business/b/spec.md", "specs/business/d/x.md"} {
		if _, err := Refresh(g.Node(rel), time.Now()); err == nil {
			t.Errorf("refresh accepted malformed %s", rel)
		}
	}
}

func TestSiblingWithoutMetadataIsInNoNode(t *testing.T) {
	root := fixture(t)
	g := settle(t, root)
	before := g.Node(C).Hash
	write(t, root, "src/backend/invite/stray.ts", "x\n")
	g = load(t, root)
	d := g.Drift()
	if len(d) != 1 || d[0] != "`src/backend/invite/stray.ts` is in no node: give it stray.ts.metadata.json, or link it in src/backend/invite/.metadata.json" {
		t.Fatalf("drift = %v", d)
	}
	if g.Node(C).Hash != before {
		t.Fatal("a sibling moved the hash of a file node")
	}
}

func TestSpecMdUnderTheCodeRootIsCode(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "src/api/spec.md", "# Not a spec\n\n### Looks like a rule\n\nx\n")
	write(t, root, "src/api/spec.md.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}]}`+"\n")
	g := load(t, root)
	n := g.Node("src/api/spec.md")
	if n == nil || n.Kind != Code || n.Rules != nil || has(g.Drift(), "is no spec.md") {
		t.Fatalf("node = %+v drift = %v", n, g.Drift())
	}
	sync(t, root, "src/api/spec.md")
	if d := load(t, root).Drift(); len(d) > 0 {
		t.Fatalf("drift = %v", d)
	}
}

func TestDirectoryNodeLinksAndDrift(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	for _, f := range []string{"go.mod", "other.txt", "free.txt"} {
		write(t, root, "src/"+f, f+"\n")
	}
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"], "nope.txt": ["`+T+`"], "sub/x": ["`+T+`"], "go.mod.metadata.json": ["`+T+`"], "AGENTS.md": ["`+T+`"], "other.txt": ["`+I+`"], "free.txt": []}}`+"\n")
	g := load(t, root)
	n := g.Node("src")
	if n == nil || !n.IsDir || n.MetaRel != "src/.metadata.json" || n.Label() != ".metadata.json" || n.UnitDir() != "src" {
		t.Fatalf("directory node = %+v", n)
	}
	if got := strings.Join(n.Files(), ","); got != "free.txt,go.mod,other.txt" {
		t.Fatalf("content = %s", got)
	}
	if want, _ := meta.HashFiles(filepath.Join(root, "src"), []string{"free.txt", "go.mod", "other.txt"}); n.Hash != want {
		t.Fatal("hash does not cover exactly the linked files")
	}
	d := g.NodeDrift(n)
	for _, want := range []string{
		"`src/.metadata.json` links `AGENTS.md`, which is chrome",
		"`src/.metadata.json` links `go.mod.metadata.json`, a metadata file",
		"`src/.metadata.json` links `nope.txt`, which is no tracked or trackable file of the directory",
		"`src/.metadata.json` links `sub/x`, which is no file name; files names the files directly in the directory",
		"`src/.metadata.json` links `other.txt` to `" + I + "`, which it does not list as a source",
	} {
		if !has(d, want) {
			t.Errorf("drift lacks %q:\n%s", want, strings.Join(d, "\n"))
		}
	}
	if has(g.Drift(), "is in no node") {
		t.Fatalf("a linked file is covered even when its link is wrong: %v", g.Drift())
	}
	if _, err := Refresh(n, time.Now()); err == nil {
		t.Fatal("refresh accepted a directory node with a bad link")
	}
	if s := g.AuditOf(n); len(s.Blocked) == 0 {
		t.Fatalf("a bad link must block the judge: %+v", s)
	}
	// Fixed: one file linked to the spec, one linked to no spec yet.
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"], "free.txt": []}}`+"\n")
	g = load(t, root)
	if d := g.NodeDrift(g.Node("src")); len(d) != 1 || !strings.Contains(d[0], "does not match the content") {
		t.Fatalf("a file linked to no spec is no drift: %v", d)
	}
	if !has(g.Drift(), "`src/other.txt` is in no node: give it other.txt.metadata.json, or link it in src/.metadata.json") {
		t.Fatalf("drift = %v", g.Drift())
	}
	g = sync(t, root, "src")
	if s := g.AuditOf(g.Node("src")); !s.Owed() || len(g.Drift()) != 1 || !has(g.Drift(), "`src/other.txt` is in no node") {
		t.Fatalf("a file linked to no spec is the judge's, not drift: %+v %v", s, g.Drift())
	}
	if d := g.Dossier(g.Node("src")); d.Links["free.txt"] == nil || len(d.Links["free.txt"]) != 0 {
		t.Fatalf("dossier links = %+v", d.Links)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "src/.metadata.json")); !strings.Contains(string(b), `"free.txt": []`) || !strings.Contains(string(b), `"audited": 0`) {
		t.Fatalf("written = %s", b)
	}
	// A file with its own metadata is not linked; files on a file node is drift; no files is drift.
	write(t, root, "src/go.mod.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}]}`+"\n")
	write(t, root, metaOf(C), strings.Replace(string(readFile(t, root, metaOf(C))), `"sources":`, `"files": {"x": []},`+"\n  "+`"sources":`, 1))
	write(t, root, "src/backend/.metadata.json", `{"version": 2}`+"\n")
	g = load(t, root)
	d = g.Drift()
	for _, want := range []string{
		"`src/.metadata.json` links `go.mod`, which has its own metadata; drop it from files",
		"`" + metaOf(C) + "` carries files, which only a directory node has",
		"`src/backend/.metadata.json` links no file; a directory node links at least one",
	} {
		if !has(d, want) {
			t.Errorf("drift lacks %q:\n%s", want, strings.Join(d, "\n"))
		}
	}
	m, err := Refresh(g.Node("src/backend"), time.Unix(0, 0))
	if err != nil {
		t.Fatalf("refresh of an empty directory node: %v", err)
	}
	Save(g.Node("src/backend"), m)
	g = load(t, root)
	if s := g.AuditOf(g.Node("src/backend")); len(s.Blocked) > 0 || !has(g.Drift(), "links no file") {
		t.Fatalf("an empty directory node is a warning, not a block: %+v drift=%v", s, g.Drift())
	}
	if _, err := Refresh(g.Node(C), time.Now()); err == nil || !strings.Contains(err.Error(), "carries files, which only a directory node has; drop it") {
		t.Fatalf("refresh wrote files through on a file node: %v", err)
	}
}

func TestUnitsGroupByDirectoryAndCountOnce(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/backend/invite/other.ts", "y\n")
	write(t, root, metaOf("src/backend/invite/other.ts"), `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+T+`"}, {"path": "`+P+`"}, {"path": "src/specs/x/spec.md"}, {"path": "specs/user/home/spec.md"}]}`+"\n")
	write(t, root, "src/backend/invite/fixture.json", "{}\n")
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+T+`"}, {"path": "`+P+`"}], "files": {"fixture.json": ["`+I+`"]}}`+"\n")
	write(t, root, "specs/user/home/spec.md", "# Home\n\n## S\n\n### Empty list\n\nShows nothing.\n")
	write(t, root, "src/specs/x/spec.md", "# X\n\n## S\n\n### X rule\n\nx\n")
	write(t, root, metaOf("src/specs/x/spec.md"), `{"version": 2, "sources": [{"path": "specs/user/home/spec.md"}]}`+"\n")
	for _, rel := range []string{P, "specs/user/home/spec.md", T, I, "src/specs/x/spec.md", C, "src/backend/invite/other.ts", "src/backend/invite"} {
		sync(t, root, rel)
	}
	g := load(t, root)
	if d := g.Drift(); len(d) > 0 {
		t.Fatalf("drift = %v", d)
	}
	write(t, root, P, strings.Replace(pSpec, "7 days", "14 days", 1))
	write(t, root, "specs/user/home/spec.md", "# Home\n\n## S\n\n### Empty list\n\nShows a hint.\n")
	sync(t, root, P)
	sync(t, root, "specs/user/home/spec.md")
	g = sync(t, root, I)
	units := g.Units()
	if len(units) != 1 || units[0].Dir != "src/backend/invite" || len(units[0].Nodes) != 3 {
		t.Fatalf("units = %+v", units)
	}
	u := units[0]
	if u.Reason != ".metadata.json, invite.ts: stale against `"+P+"`; other.ts: stale against `"+P+"`, `specs/user/home/spec.md`" {
		t.Fatalf("reason = %s", u.Reason)
	}
	if u.Wait != "" || u.Nodes[2].Node.Rel != "src/backend/invite/other.ts" || !strings.Contains(u.Nodes[2].Wait, "`src/specs/x/spec.md` is stale against `specs/user/home/spec.md`") || u.Nodes[0].Wait != "" {
		t.Fatalf("waits = %+v", u.Nodes)
	}
	if len(u.Rules) != 2 || u.Rules[0] != (RuleState{Title: "Invite expiry", State: "changed", Source: P}) || u.Rules[1] != (RuleState{Title: "Empty list", State: "changed", Source: "specs/user/home/spec.md"}) {
		t.Fatalf("rules = %+v", u.Rules)
	}
	if run, wait := g.Batch(1); len(run) != 1 || len(wait) != 0 || len(run[0].Nodes) != 3 {
		t.Fatalf("batch run=%+v wait=%+v", run, wait)
	}
	sync(t, root, "src/specs/x/spec.md")
	g = load(t, root)
	if u := g.Units(); len(u) != 1 || u[0].Nodes[2].Wait != "" {
		t.Fatalf("units = %+v", u)
	}
}

func TestBatchOrderAndCap(t *testing.T) {
	root := gitInit(t)
	for _, n := range []string{"a", "b", "c", "d", "e", "f"} {
		write(t, root, "src/specs/"+n+"/spec.md", "# "+n+"\n\n## S\n\n### Rule "+n+"\n\nx\n")
		src := `{"version": 2}`
		if n == "a" {
			src = `{"version": 2, "sources": [{"path": "src/specs/f/spec.md"}]}`
		}
		write(t, root, "src/specs/"+n+"/spec.md.metadata.json", src+"\n")
	}
	for _, n := range []string{"f", "a", "b", "c", "d", "e"} {
		sync(t, root, "src/specs/"+n+"/spec.md")
	}
	g := load(t, root)
	run, wait := g.Batch(5)
	if len(run) != 5 || len(wait) != 0 {
		t.Fatalf("run = %d wait = %d", len(run), len(wait))
	}
	var order []string
	for _, u := range g.Units() {
		order = append(order, u.Rel())
	}
	if want := "src/specs/b/spec.md src/specs/c/spec.md src/specs/d/spec.md src/specs/e/spec.md src/specs/f/spec.md src/specs/a/spec.md"; strings.Join(order, " ") != want {
		t.Fatalf("order = %v", order)
	}
	if r := run[0].Rules; len(r) != 1 || r[0].State != "new" {
		t.Fatalf("rules = %+v", r)
	}
}

func TestRefreshIsStableAndMovesModifiedOnlyOnChange(t *testing.T) {
	root := fixture(t)
	g := load(t, root)
	n := g.Node(P)
	m1, err := Refresh(n, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Save(n, m1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(metaOf(P)))); err != nil {
		t.Fatalf("write did not create %s: %v", metaOf(P), err)
	}
	g = load(t, root)
	n = g.Node(P)
	m2, _ := Refresh(n, time.Unix(200, 0))
	if written, _ := Save(n, m2); written || m2.Modified != "1970-01-01T00:01:40Z" {
		t.Fatalf("unchanged content rewrote metadata: written=%v modified=%s", written, m2.Modified)
	}
	write(t, root, P, pSpec+"\n### Third\n\nz\n")
	g = load(t, root)
	m3, _ := Refresh(g.Node(P), time.Unix(300, 0))
	if m3.Modified != "1970-01-01T00:05:00Z" || m3.Rules["Third"] == "" {
		t.Fatalf("changed content: %+v", m3)
	}
}

func TestSaveUpdatesGraphForLaterSyncs(t *testing.T) {
	root := fixture(t)
	g := load(t, root)
	for _, n := range g.SourcesFirst(g.Nodes) {
		m, err := g.Sync(n, time.Unix(0, 0))
		if err != nil {
			t.Fatalf("sync %s in one run: %v", n.Rel, err)
		}
		if _, err := Save(n, m); err != nil {
			t.Fatal(err)
		}
	}
	if d := load(t, root).Drift(); len(d) > 0 {
		t.Fatalf("drift = %v", d)
	}
	var order []string
	for _, n := range g.SourcesFirst(g.Nodes) {
		order = append(order, n.Rel)
	}
	if want := P + " " + T + " " + I + " " + C; strings.Join(order, " ") != want {
		t.Fatalf("order = %v", order)
	}
}

func gitIn(t *testing.T, root string) func(args ...string) {
	return func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
}

func staged(t *testing.T, root string) string {
	t.Helper()
	nodes, uncovered, err := load(t, root).Staged()
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, n := range nodes {
		rels = append(rels, n.Rel)
	}
	return strings.Join(append(rels, uncovered...), " ")
}

func TestStagedFollowsFileNodesMetadataAndLinks(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	git := gitIn(t, root)
	write(t, root, "src/go.mod", "module x\n")
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"]}}`+"\n")
	sync(t, root, "src")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	// A rename with its metadata: only the new node is staged.
	git("mv", C, "src/backend/invite/moved.ts")
	git("mv", metaOf(C), metaOf("src/backend/invite/moved.ts"))
	if got := staged(t, root); got != "src/backend/invite/moved.ts" {
		t.Fatalf("staged = %q", got)
	}
	g := load(t, root)
	if _, err := Refresh(g.Node("src/backend/invite/moved.ts"), time.Now()); err != nil {
		t.Fatalf("refresh of the moved node: %v", err)
	}
	// A linked file and a directory node's metadata stage the directory node.
	write(t, root, "src/go.mod", "module y\n")
	git("add", "src/go.mod")
	if got := staged(t, root); got != "src src/backend/invite/moved.ts" {
		t.Fatalf("staged = %q", got)
	}
	git("commit", "-q", "-m", "moved")
	git("add", "src/.metadata.json")
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"]}}`+"\n")
	git("add", "src/.metadata.json")
	if got := staged(t, root); got != "src" {
		t.Fatalf("staged = %q", got)
	}
	// A rename without its metadata leaves an orphan, which refresh refuses, and a file in no node.
	git("commit", "-q", "-m", "linked")
	git("mv", "src/backend/invite/moved.ts", "src/backend/invite/again.ts")
	if got := staged(t, root); got != "src/backend/invite/moved.ts `src/backend/invite/again.ts` is in no node: give it again.ts.metadata.json, or link it in src/backend/invite/.metadata.json" {
		t.Fatalf("staged = %q", got)
	}
	g = load(t, root)
	if _, err := Refresh(g.Node("src/backend/invite/moved.ts"), time.Now()); err == nil || !strings.Contains(err.Error(), "has no moved.ts beside it; delete it, or rename it with its file") {
		t.Fatalf("refresh of an orphan: %v", err)
	}
	if !has(g.Drift(), "`src/backend/invite/again.ts` is in no node") {
		t.Fatalf("drift = %v", g.Drift())
	}
	// A staged deletion of metadata alone, and a staged new file without metadata, are reported.
	git("mv", "src/backend/invite/again.ts", "src/backend/invite/moved.ts")
	git("rm", "-q", metaOf("src/backend/invite/moved.ts"))
	write(t, root, "src/backend/invite/new.ts", "n\n")
	git("add", "src/backend/invite/new.ts")
	if got := staged(t, root); got != "`src/backend/invite/moved.ts` is in no node: give it moved.ts.metadata.json, or link it in src/backend/invite/.metadata.json `src/backend/invite/new.ts` is in no node: give it new.ts.metadata.json, or link it in src/backend/invite/.metadata.json" {
		t.Fatalf("staged = %q", got)
	}
}

func TestOwner(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "src/go.mod", "module x\n")
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"]}}`+"\n")
	g := load(t, root)
	for f, want := range map[string]string{
		C: C, metaOf(C): C, P: P, metaOf(P): P,
		"src/go.mod": "src", "src/.metadata.json": "src",
		"src/AGENTS.md": "", "src/other.txt": "", "src/backend/invite/stray.ts": "", "src/backend/.metadata.json": "",
	} {
		n := g.Owner(f)
		switch {
		case want == "" && n != nil:
			t.Errorf("Owner(%s) = %s, want none", f, n.Rel)
		case want != "" && (n == nil || n.Rel != want):
			t.Errorf("Owner(%s) = %v, want %s", f, n, want)
		}
	}
}

func TestResolve(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "src/go.mod", "module x\n")
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"]}}`+"\n")
	g := load(t, root)
	rels := func(nodes []*Node) string {
		var out []string
		for _, n := range nodes {
			out = append(out, n.Rel)
		}
		return strings.Join(out, " ")
	}
	for target, want := range map[string]string{
		C: C, metaOf(C): C, "src/go.mod": "src", "src/.metadata.json": "src",
		"src/backend": C, "src": "src " + C,
		"src.metadata.json": "", "src/backend.metadata.json": "", "src/missing": "",
	} {
		if got := rels(g.CodeNodes(target)); got != want {
			t.Errorf("CodeNodes(%s) = %q, want %q", target, got, want)
		}
	}
	if got := rels(g.Resolve("src/specs")); got != I+" "+T {
		t.Errorf("Resolve(src/specs) = %q", got)
	}
}

func TestMissingServedPerspectiveIsAnImplementUnit(t *testing.T) {
	root := fixture(t)
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+T+`"}]}`+"\n")
	for _, rel := range []string{P, T, I} {
		sync(t, root, rel)
	}
	g := load(t, root)
	units := g.Units()
	if len(units) != 1 || units[0].Dir != "src/backend/invite" || !strings.Contains(units[0].Reason, "invite.ts: stale against `"+I+"`, `"+T+"`; missing source `"+P+"` (served by `"+I+"`)") {
		t.Fatalf("units = %+v", units)
	}
	if r := units[0].Rules; len(r) != 4 || r[0] != (RuleState{Title: "Invite endpoint", State: "new", Source: I}) || r[3] != (RuleState{Title: "Single use", State: "new", Source: P}) {
		t.Fatalf("rules = %+v", r)
	}
	if has(g.Drift(), "missing") {
		t.Fatalf("a missing served perspective is pending work, not drift: %v", g.Drift())
	}
	if _, err := g.Sync(g.Node(C), time.Now()); err == nil || !strings.Contains(err.Error(), "add it to sources") {
		t.Fatalf("sync did not refuse with guidance: %v", err)
	}
}

func TestSyncRefusesACodeNodeThatWaits(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, P, strings.Replace(pSpec, "7 days", "14 days", 1))
	sync(t, root, P)
	g := load(t, root)
	_, err := g.Sync(g.Node(C), time.Now())
	if err == nil || !strings.Contains(err.Error(), "waits: `"+I+"` is stale against `"+P+"` (/sdd-design)") {
		t.Fatalf("sync accepted a waiting code node: %v", err)
	}
}

func TestDanglingSourceListsRecordedRulesAsRemoved(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	if err := os.RemoveAll(filepath.Join(root, "specs/business/invite")); err != nil {
		t.Fatal(err)
	}
	g := load(t, root)
	if !has(g.Design(), "`"+I+"` has a dangling source `"+P+"`: removed \"Invite expiry\", \"Single use\"") {
		t.Fatalf("design = %v", g.Design())
	}
	u := g.Units()
	if len(u) != 1 || len(u[0].Rules) != 2 || u[0].Rules[0] != (RuleState{Title: "Invite expiry", State: "removed", Source: P}) {
		t.Fatalf("units = %+v", u)
	}
}

func TestDesignGapParksRulesAndSpecGapAnnotates(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, I, "# Invite\n\nPurpose.\n\n## HTTP\n\n### Invite endpoint\n\n`POST /invites` **must** create one. Serves **Invite expiry**.\n\n### Token storage\n\nStored somewhere.\n")
	sync(t, root, I)
	write(t, root, "specs/TODO.md", "# TODO\n\n## Design gap\n\n- **Token storage** (`src/specs/invite`) names no persistence package.\n- [ ] **Ghost** (`src/specs/invite`) matches nothing.\n\n## Spec gap\n\n- **Invite expiry** (`business/invite`) leaves the grace period undefined.\n- broken line\n")
	g := load(t, root)
	u := g.Units()
	if len(u) != 1 || len(u[0].Rules) != 1 || u[0].Rules[0].Parked != "names no persistence package" || u[0].Wait != "every rule is parked under ## Design gap (/sdd-design)" {
		t.Fatalf("units = %+v", u)
	}
	if run, wait := g.Batch(5); len(run) != 0 || len(wait) != 1 {
		t.Fatalf("run = %d wait = %d", len(run), len(wait))
	}
	d := g.Drift()
	if !has(d, `design gap "Ghost" (`+"`src/specs/invite`"+`) matches no pending rule`) || !has(d, "spec gap out of shape: - broken line") {
		t.Fatalf("drift = %v", d)
	}
	write(t, root, P, strings.Replace(pSpec, "7 days", "14 days", 1))
	g = load(t, root)
	if !has(g.Design(), `changed "Invite expiry"; "Invite expiry" parked: leaves the grace period undefined (/sdd-spec)`) {
		t.Fatalf("design = %v", g.Design())
	}
	// An owner may name the spec file itself.
	write(t, root, "specs/TODO.md", "# TODO\n\n## Spec gap\n\n- **Invite expiry** (`"+P+"`) leaves the grace period undefined.\n")
	if g = load(t, root); !has(g.Design(), `"Invite expiry" parked:`) || has(g.Drift(), "matches no rule") {
		t.Fatalf("design = %v drift = %v", g.Design(), g.Drift())
	}
}

func TestGlobalSpecWithoutCodeNodeNamesTheFix(t *testing.T) {
	root := fixture(t)
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+P+`"}]}`+"\n")
	g := settle(t, root)
	u := g.Units()
	if len(u) != 1 || u[0].Node.Rel != T || u[0].Reason != "no code node lists it; list it from the code nodes it constrains" {
		t.Fatalf("units = %+v", u)
	}
}

func TestChromeUnderSrcIsNoContent(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"]}}`+"\n")
	write(t, root, "src/go.mod", "module x\n")
	g := settle(t, root)
	sync(t, root, "src")
	g = load(t, root)
	before := g.Node("src").Hash
	write(t, root, "src/BUG-AUTOPSIES.md", "# Bug autopsies\n\nnew entry\n")
	g = load(t, root)
	if g.Node("src").Hash != before || len(g.Units()) > 0 || len(g.Drift()) > 0 {
		t.Fatalf("chrome moved the hash or drifted: units=%+v drift=%v", g.Units(), g.Drift())
	}
}

func approve(t *testing.T, root, rel string) *Graph {
	t.Helper()
	g := load(t, root)
	m, err := g.Approve(g.Node(rel), time.Unix(0, 0))
	if err != nil {
		t.Fatalf("approve %s: %v", rel, err)
	}
	if _, err := Save(g.Node(rel), m); err != nil {
		t.Fatal(err)
	}
	return load(t, root)
}

func readFile(t *testing.T, root, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCodeNodeWithoutTechnicalSpecIsDrift(t *testing.T) {
	root := fixture(t)
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "`+P+`"}]}`+"\n")
	for _, rel := range []string{P, T, I} {
		sync(t, root, rel)
	}
	g := load(t, root)
	d := g.Drift()
	if !has(d, "`"+C+"` lists no technical spec; a code node lists at least one") || has(d, "none of its technical specs serves it") {
		t.Fatalf("drift = %v", d)
	}
	if _, err := g.Sync(g.Node(C), time.Now()); err == nil || !strings.Contains(err.Error(), "lists no technical spec") || strings.Contains(err.Error(), "none of its technical specs serves it") {
		t.Fatalf("sync on a code node with no technical spec: %v", err)
	}
	if s := g.AuditOf(g.Node(C)); s.Unit == "" || len(s.Blocked) == 0 {
		t.Fatalf("an unsynced perspective is /sdd-implement's first and a hand-written file is drift: %+v", s)
	}
	// With no source at all, once written, nothing blocks the judge: it proposes the spec.
	write(t, root, metaOf(C), `{"version": 2}`+"\n")
	g = load(t, root)
	m, err := Refresh(g.Node(C), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	Save(g.Node(C), m)
	g = load(t, root)
	if s := g.AuditOf(g.Node(C)); len(s.Blocked) > 0 || !s.Owed() || !has(g.Drift(), "lists no technical spec") {
		t.Fatalf("no technical spec must not block the judge: %+v drift=%v", s, g.Drift())
	}
	// A dangling technical source is reported as dangling only.
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "src/specs/gone/spec.md"}, {"path": "`+P+`"}]}`+"\n")
	g = load(t, root)
	if d := g.Drift(); has(d, "no technical spec") || has(d, "none of its technical specs serves it") {
		t.Fatalf("dangling technical source reported as another drift: %v", d)
	}
	m, _ = Refresh(g.Node(C), time.Unix(0, 0))
	Save(g.Node(C), m)
	g = load(t, root)
	if _, err := g.Approve(g.Node(C), time.Now()); err == nil || !strings.Contains(err.Error(), "dangling source `src/specs/gone/spec.md`") {
		t.Fatalf("approve on a dangling source: %v", err)
	}
	if u, ok := g.codeUnit(g.Node(C)); !ok || !strings.Contains(u.Reason, "dangling source `src/specs/gone/spec.md`") {
		t.Fatalf("units = %+v", u)
	}
	if s := g.AuditOf(g.Node(C)); s.Unit == "" || s.Owed() {
		t.Fatalf("a dangling source is /sdd-implement's first: %+v", s)
	}
}

func TestAuditedStartsAtZeroAndResetsOnContentChangeOrNewSource(t *testing.T) {
	root := fixture(t)
	g := settle(t, root)
	c := g.Node(C)
	if c.Meta.Audited == nil || *c.Meta.Audited != 0 || g.Node(I).Meta.Audited != nil || g.Node(P).Meta.Audited != nil {
		t.Fatalf("audited after settle: code=%v technical=%v", c.Meta.Audited, g.Node(I).Meta.Audited)
	}
	if s := g.AuditOf(c); !s.Owed() {
		t.Fatalf("settled code node not owed: %+v", s)
	}
	g = approve(t, root, C)
	if a := g.Node(C).Meta.Audited; a == nil || *a != 1 {
		t.Fatalf("audited after approve = %v", a)
	}
	if s := g.AuditOf(g.Node(C)); s.Owed() || s.Audited != 1 {
		t.Fatalf("approved node still owed: %+v", s)
	}
	m, _ := Refresh(g.Node(C), time.Unix(50, 0))
	if written, _ := Save(g.Node(C), m); written || *m.Audited != 1 {
		t.Fatalf("refresh without change touched audited: written=%v audited=%d", written, *m.Audited)
	}
	write(t, root, C, "export const days = 14;\n")
	g = load(t, root)
	if s := g.AuditOf(g.Node(C)); s.Audited != 0 {
		t.Fatalf("changed content still counts as audited: %+v", s)
	}
	m, _ = Refresh(g.Node(C), time.Unix(60, 0))
	if *m.Audited != 0 {
		t.Fatalf("refresh after change kept audited = %d", *m.Audited)
	}
	Save(g.Node(C), m)
	g = approve(t, root, C)
	// A source listed since the last sync returns audited to 0 when recorded.
	write(t, root, "src/specs/extra/spec.md", "# Extra\n\nPurpose.\n\n## S\n\n### Extra rule\n\nx\n")
	write(t, root, "src/specs/extra/spec.md.metadata.json", `{"version": 2}`+"\n")
	sync(t, root, "src/specs/extra/spec.md")
	file := string(readFile(t, root, metaOf(C)))
	write(t, root, metaOf(C), strings.Replace(file, `"path": "`+I+`"`, `"path": "src/specs/extra/spec.md"`+"\n    },\n    {\n      "+`"path": "`+I+`"`, 1))
	g = load(t, root)
	if a := g.Node(C).Meta.Audited; a == nil || *a != 1 {
		t.Fatalf("editing sources moved audited: %v", a)
	}
	if _, err := g.Approve(g.Node(C), time.Now()); err == nil || !strings.Contains(err.Error(), "/sdd-implement unit") {
		t.Fatalf("approve accepted a node with an unsynced source: %v", err)
	}
	g = sync(t, root, C)
	if a := g.Node(C).Meta.Audited; a == nil || *a != 0 {
		t.Fatalf("sync of a new source kept audited = %v", a)
	}
	approve(t, root, C)
	g = sync(t, root, C)
	if a := g.Node(C).Meta.Audited; *a != 1 {
		t.Fatalf("sync without a new source moved audited = %d", *a)
	}
}

func TestLinkEditMovesTheHashOfADirectoryNode(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	for _, f := range []string{"go.mod", "go.sum"} {
		write(t, root, "src/"+f, f+"\n")
	}
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"]}}`+"\n")
	sync(t, root, "src")
	g := approve(t, root, "src")
	before := g.Node("src").Hash
	write(t, root, "src/.metadata.json", strings.Replace(string(readFile(t, root, "src/.metadata.json")), `"go.mod": [`, `"go.sum": [], "go.mod": [`, 1))
	g = load(t, root)
	if g.Node("src").Hash == before || auditedOf(g.Node("src")) != 0 || !has(g.Drift(), "does not match the content") {
		t.Fatalf("linking a file must move the hash: %+v", g.AuditOf(g.Node("src")))
	}
	m, err := Refresh(g.Node("src"), time.Unix(0, 0))
	if err != nil || *m.Audited != 0 || m.Files["go.sum"] == nil {
		t.Fatalf("refresh after a link edit: %v %+v", err, m)
	}
}

func TestApproveRefusals(t *testing.T) {
	root := fixture(t)
	g := settle(t, root)
	if _, err := g.Approve(g.Node(I), time.Now()); err == nil || !strings.Contains(err.Error(), "only a code node") {
		t.Fatalf("approve accepted a technical node: %v", err)
	}
	write(t, root, C, "changed\n")
	g = load(t, root)
	if _, err := g.Approve(g.Node(C), time.Now()); err == nil || !strings.Contains(err.Error(), "changed since its metadata was last written") {
		t.Fatalf("approve accepted a moved hash: %v", err)
	}
	write(t, root, C, "export const days = 7;\n")
	write(t, root, P, strings.Replace(pSpec, "7 days", "14 days", 1))
	sync(t, root, P)
	g = load(t, root)
	if _, err := g.Approve(g.Node(C), time.Now()); err == nil || !strings.Contains(err.Error(), "/sdd-implement unit (stale against") {
		t.Fatalf("approve accepted a stale node: %v", err)
	}
}

func TestApproveAcceptsANodeThatOnlyWaits(t *testing.T) {
	root := fixture(t)
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+P+`"}]}`+"\n")
	settle(t, root)
	write(t, root, T, "# Toolchain\n\nPurpose.\n\n## Tests\n\n### Vitest as test runner\n\nTests **must** run on `vitest` 2.\n")
	sync(t, root, T)
	g := load(t, root)
	if !has(g.Design(), "`"+I+"` is stale against `"+T+"`") {
		t.Fatalf("design = %v", g.Design())
	}
	if u, ok := g.codeUnit(g.Node(C)); ok {
		t.Fatalf("a node whose own sources are fresh is no unit: %+v", u)
	}
	if _, err := g.Approve(g.Node(C), time.Now()); err != nil {
		t.Fatalf("approve refused a node that only waits: %v", err)
	}
}

func TestDossierTiersCandidatesAndNamesCorrections(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/specs/http-client/spec.md", "# HTTP client\n\nHow HTTP calls are made.\n\n## Client\n\n### Single client\n\nCode under `src/backend/invite/` **must** use `acme-http`.\n")
	write(t, root, "src/specs/http-client/spec.md.metadata.json", `{"version": 2}`+"\n")
	write(t, root, "src/specs/other/spec.md", "# Other\n\nSomething else entirely.\n\n## S\n\n### Other rule\n\nx\n")
	write(t, root, "src/specs/other/spec.md.metadata.json", `{"version": 2, "sources": [{"path": "`+P+`"}]}`+"\n")
	write(t, root, "specs/user/home/spec.md", "# Home\n\nPurpose.\n\n## S\n\n### Empty list\n\nShows nothing.\n")
	write(t, root, metaOf(I), `{"version": 2, "sources": [{"path": "`+P+`"}, {"path": "specs/user/home/spec.md"}, {"path": "`+T+`"}]}`+"\n")
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+P+`"}]}`+"\n")
	for _, rel := range []string{P, "specs/user/home/spec.md", T, "src/specs/http-client/spec.md", "src/specs/other/spec.md", I} {
		sync(t, root, rel)
	}
	g := load(t, root)
	d := g.Dossier(g.Node(C))
	if strings.Join(d.Files, ",") != "invite.ts" || d.Links != nil || len(d.Technical) != 1 || d.Technical[0].Rel != I || len(d.Perspectives) != 1 {
		t.Fatalf("dossier = %+v", d)
	}
	var cands []string
	for _, c := range d.Candidates {
		cands = append(cands, fmt.Sprintf("%s:%d:%v:%v:%s", c.Node.Rel, c.Tier(), c.Names, c.Global, c.Purpose))
	}
	want := "src/specs/http-client/spec.md:1:true:true:How HTTP calls are made. " + T + ":2:false:true:Purpose. src/specs/other/spec.md:3:false:false:Something else entirely."
	if strings.Join(cands, " ") != want {
		t.Fatalf("candidates = %v", cands)
	}
	if len(d.DropPerspectives) != 0 {
		t.Fatalf("drop = %+v", d.DropPerspectives)
	}
	if s := d.State; s.Unit == "" || s.Owed() {
		t.Fatalf("a missing served perspective is /sdd-implement's first: %+v", s)
	}
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+P+`"}, {"path": "specs/user/home/spec.md"}]}`+"\n")
	sync(t, root, C)
	write(t, root, metaOf(I), `{"version": 2, "sources": [{"path": "`+P+`"}, {"path": "`+T+`"}]}`+"\n")
	sync(t, root, I)
	g = load(t, root)
	d = g.Dossier(g.Node(C))
	if len(d.DropPerspectives) != 1 || d.DropPerspectives[0] != "`specs/user/home/spec.md` (none of its technical specs serves it)" {
		t.Fatalf("drop = %+v", d.DropPerspectives)
	}
	if nodes := g.CodeNodes("src/backend"); len(nodes) != 1 || nodes[0].Rel != C || len(g.CodeNodes("src/specs")) != 0 {
		t.Fatalf("CodeNodes = %+v", nodes)
	}
	if nodes := g.CodeNodes(C); len(nodes) != 1 || nodes[0].Rel != C || len(g.CodeNodes("src/backend/invite/invite")) != 0 {
		t.Fatalf("CodeNodes on a file = %+v", nodes)
	}
}

func TestDossierOfADirectoryNodeCarriesItsLinks(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "src/go.mod", "module x\n")
	write(t, root, "src/free.txt", "x\n")
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"], "free.txt": []}}`+"\n")
	g := sync(t, root, "src")
	d := g.Dossier(g.Node("src"))
	if strings.Join(d.Files, ",") != "free.txt,go.mod" || len(d.Links["go.mod"]) != 1 || d.Links["go.mod"][0] != T || len(d.Links["free.txt"]) != 0 {
		t.Fatalf("dossier = %+v", d)
	}
	if nodes := g.CodeNodes("src"); len(nodes) != 2 || nodes[0].Rel != "src" || nodes[1].Rel != C {
		t.Fatalf("CodeNodes(src) = %+v", nodes)
	}
}

func TestNamesReMatchesWholePathTokens(t *testing.T) {
	re := namesRe("src/backend/invite", "src")
	for text, want := range map[string]bool{
		"Code under `src/backend/invite/` lives here.": true,
		"Everything under src/backend.":                true,
		"Everything under src/backend-old/ is gone.":   false,
		"Not src/backend/invitee.":                     false,
		"Not src/backend/invite/deeper either.":        false,
		"Only src/ in general.":                        false,
		"Links to specs/business/invite.":              false,
	} {
		if got := re.MatchString(text); got != want {
			t.Errorf("%q: names = %v, want %v", text, got, want)
		}
	}
	if !namesRe("src", "src").MatchString("Manifests live in `src/`.") {
		t.Error("src node: src/ not matched")
	}
	re = namesRe("src/api/server.ts", "src")
	for text, want := range map[string]bool{
		"`src/api/server.ts` **must** export `build`.": true,
		"Keep src/api/server.ts.bak around.":           false,
		"Everything under `src/api/`.":                 true,
		"Not src/api/auth/ though.":                    false,
		"Only src/ in general.":                        false,
	} {
		if got := re.MatchString(text); got != want {
			t.Errorf("%q: names = %v, want %v", text, got, want)
		}
	}
}

func TestAuditGapParksEveryCodeNodeUnderItsOwner(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "specs/TODO.md", "# TODO\n\n## Audit gap\n\n- **Rate limiting** (`src/backend/invite`) code obeys a limit no technical spec states.\n- **Ghost** (`src/nowhere`) nothing.\n")
	g := load(t, root)
	s := g.AuditOf(g.Node(C))
	if s.Parked != "code obeys a limit no technical spec states" || s.Owed() {
		t.Fatalf("state = %+v", s)
	}
	d := g.Drift()
	if !has(d, "audit gap \"Ghost\" (`src/nowhere`) names no code node that owes an audit") || has(d, "Rate limiting") {
		t.Fatalf("drift = %v", d)
	}
	approve(t, root, C)
	if d := load(t, root).Drift(); !has(d, "audit gap \"Rate limiting\" (`src/backend/invite`) names no code node that owes an audit") {
		t.Fatalf("drift after approve = %v", d)
	}
	// The owner may be the node itself, or the root code node, which parks everything under src.
	write(t, root, "src/go.mod", "module x\n")
	write(t, root, "src/.metadata.json", `{"version": 2, "sources": [{"path": "`+T+`"}], "files": {"go.mod": ["`+T+`"]}}`+"\n")
	sync(t, root, "src")
	write(t, root, "specs/TODO.md", "# TODO\n\n## Audit gap\n\n- **Root manifests** (`src`) nothing decides them.\n- **Limit** (`"+C+"`) nothing states it.\n")
	write(t, root, C, "y\n")
	g = load(t, root)
	m, _ := Refresh(g.Node(C), time.Unix(0, 0))
	Save(g.Node(C), m)
	g = load(t, root)
	if s := g.AuditOf(g.Node("src")); s.Parked != "nothing decides them" {
		t.Fatalf("state = %+v", s)
	}
	if s := g.AuditOf(g.Node(C)); s.Parked != "nothing decides them" {
		t.Fatalf("state = %+v", s)
	}
	if has(g.Drift(), "Root manifests") || has(g.Drift(), "Limit") {
		t.Fatalf("drift = %v", g.Drift())
	}
	// A linked file, or the metadata file of a directory node, parks that node alone.
	for _, owner := range []string{"src/go.mod", "src/.metadata.json"} {
		write(t, root, "specs/TODO.md", "# TODO\n\n## Audit gap\n\n- **Manifest** (`"+owner+"`) nothing decides it.\n")
		g = load(t, root)
		if s := g.AuditOf(g.Node("src")); s.Parked != "nothing decides it" {
			t.Fatalf("owner %s: state = %+v", owner, s)
		}
		if s := g.AuditOf(g.Node(C)); s.Parked != "" {
			t.Fatalf("owner %s parked a sibling: %+v", owner, s)
		}
		if has(g.Drift(), "Manifest") {
			t.Fatalf("owner %s: drift = %v", owner, g.Drift())
		}
	}
	write(t, root, "specs/TODO.md", "# TODO\n\n## Audit gap\n\n- **Ghost** (`src/other.txt`) nothing.\n- **Spec** (`"+T+"`) nothing.\n")
	if d := load(t, root).Drift(); !has(d, "audit gap \"Ghost\" (`src/other.txt`) names no code node") || !has(d, "audit gap \"Spec\" (`"+T+"`) names no code node") {
		t.Fatalf("drift = %v", d)
	}
}

func TestApproveOnANeverWrittenSkeletonNamesTheDrift(t *testing.T) {
	root := fixture(t)
	for _, rel := range []string{P, T, I} {
		sync(t, root, rel)
	}
	g := load(t, root)
	_, err := g.Approve(g.Node(C), time.Now())
	if err == nil || !strings.Contains(err.Error(), "does not match the content; run metadata --write") || strings.Contains(err.Error(), "changed since") {
		t.Fatalf("approve on a skeleton: %v", err)
	}
}

func TestAuditedPlacementIsDrift(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	code := string(readFile(t, root, metaOf(C)))
	write(t, root, metaOf(C), strings.Replace(code, "  \"audited\": 0,\n", "", 1))
	tech := string(readFile(t, root, metaOf(I)))
	write(t, root, metaOf(I), strings.Replace(tech, "  \"rules\":", "  \"audited\": 0,\n  \"rules\":", 1))
	g := load(t, root)
	d := g.Drift()
	if !has(d, "`"+metaOf(C)+"` carries no audited; run metadata --write "+C) || !has(d, "`"+metaOf(I)+"` carries audited, which only a code node has") {
		t.Fatalf("drift = %v", d)
	}
	if s := g.AuditOf(g.Node(C)); len(s.Blocked) == 0 {
		t.Fatalf("a missing audited must block the judge: %+v", s)
	}
	for _, rel := range []string{I, C} {
		m, err := Refresh(g.Node(rel), time.Unix(0, 0))
		if err != nil {
			t.Fatal(err)
		}
		if written, _ := Save(g.Node(rel), m); !written {
			t.Fatalf("write did not repair %s", rel)
		}
	}
	if d := load(t, root).Drift(); len(d) > 0 {
		t.Fatalf("drift after write = %v", d)
	}
}

func TestNamesReRejectsFilesBesideTheDirectory(t *testing.T) {
	re := namesRe("src/backend/invite", "src")
	for text, want := range map[string]bool{
		"The entry is src/backend/invite.ts.":        false,
		"See src/backend.md for the layout.":         false,
		"Code under ./src/backend/invite/ is x.":     true,
		"Placement: (src/backend).":                  true,
		"Everything in `src/backend/invite`, period": true,
	} {
		if got := re.MatchString(text); got != want {
			t.Errorf("%q: names = %v, want %v", text, got, want)
		}
	}
}

func TestPurposeJoinsWrappedLinesAndSkipsFences(t *testing.T) {
	for text, want := range map[string]string{
		"# T\n\nA purpose that wraps\nonto a second line.\n\n## S\n\n### R\n\nx\n": "A purpose that wraps onto a second line.",
		"# T\n\n```\ncode\n```\n\n## S\n":                                          "",
		"# T\n\n## S\n\n### R\n\nx\n":                                              "",
		"# T\r\n\r\nCRLF purpose.\r\n\r\n## S\r\n":                                 "CRLF purpose.",
	} {
		if got := purpose(text); got != want {
			t.Errorf("%q: purpose = %q, want %q", text, got, want)
		}
	}
}

func TestLinkToAnIgnoredFileIsDrift(t *testing.T) {
	root := fixture(t)
	write(t, root, ".gitignore", "dist/\n")
	write(t, root, "src/dist/out.js", "x\n")
	write(t, root, P, pSpec+"\nSee [the build](../../../src/dist/out.js) and [the root](/AGENTS.md#root).\n")
	g := load(t, root)
	if d := g.NodeDrift(g.Node(P)); len(d) != 2 || !has(d, "`src/dist/out.js` does not exist") {
		t.Fatalf("drift of %s = %v", P, d)
	}
}

func TestCodeWaitsOnTheDriftOfASource(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, I, "# Invite\n\nPurpose.\n\n## HTTP\n\n### Invite endpoint\n\n`POST /invites` **must** create one and answer `201`. Serves **Invite expiry**.\n")
	g := load(t, root)
	run, wait := g.Batch(5)
	if len(run) != 0 || len(wait) != 1 || !strings.Contains(wait[0].Wait, "`"+I+"` has drift") {
		t.Fatalf("run = %+v wait = %+v", run, wait)
	}
}

func TestDesignAndCodeWaitOnTheDriftOfAFreshSource(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, T, "# Toolchain\n\nPurpose.\n\n## Tests\n\n### Vitest as test runner\n\nTests **must** run on `vitest`.\n\n### Lint\n\nLint **must** pass.\n")
	sync(t, root, T)
	if err := os.Remove(filepath.Join(root, metaOf(P))); err != nil {
		t.Fatal(err)
	}
	g := load(t, root)
	if !has(g.Design(), "`"+I+"` is stale against `"+T+"`: added \"Lint\" — waits: `"+P+"` has drift") {
		t.Fatalf("design = %v", g.Design())
	}
	if u := g.Units(); len(u) != 1 || !strings.Contains(u[0].Wait, "`"+P+"` has drift") {
		t.Fatalf("units = %+v", u)
	}
}

func TestUpstreamPerspectiveDriftDoesNotMakeCodeWait(t *testing.T) {
	root := fixture(t)
	const O = "specs/business/other/spec.md"
	write(t, root, O, "# Other\n\nPurpose.\n\n## A\n\n### Other rule\n\nIt **must** hold.\n")
	write(t, root, metaOf(T), `{"version": 2, "sources": [{"path": "`+O+`"}]}`+"\n")
	write(t, root, metaOf(C), `{"version": 2, "sources": [{"path": "`+I+`"}, {"path": "`+P+`"}]}`+"\n")
	sync(t, root, O)
	settle(t, root)
	write(t, root, I, "# Invite\n\nPurpose.\n\n## HTTP\n\n### Invite endpoint\n\n`POST /invites` **must** create one and answer `201`.\n")
	sync(t, root, I)
	if err := os.Remove(filepath.Join(root, metaOf(O))); err != nil {
		t.Fatal(err)
	}
	g := load(t, root)
	run, wait := g.Batch(5)
	if len(wait) != 0 || len(run) != 2 || run[1].Dir != "src/backend/invite" {
		t.Fatalf("run = %+v wait = %+v", run, wait)
	}
	if _, err := g.Sync(g.Node(C), time.Now()); err != nil {
		t.Fatalf("sync of %s refused for the drift of a perspective upstream: %v", C, err)
	}
}

func TestLinkIntoAFileDeletedButStillTrackedIsBroken(t *testing.T) {
	root := fixture(t)
	const O = "specs/business/other/spec.md"
	write(t, root, O, "# Other\n\nSee [invite](../invite/spec.md) and [single use](../invite/spec.md#single-use).\n")
	if out, err := exec.Command("git", "-C", root, "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v %s", err, out)
	}
	if err := os.RemoveAll(filepath.Join(root, "specs/business/invite")); err != nil {
		t.Fatal(err)
	}
	g := load(t, root)
	if d := g.NodeDrift(g.Node(O)); has(d, "has no heading") || !has(d, "links `../invite/spec.md`, but `specs/business/invite/spec.md` does not exist") || !has(d, "links `../invite/spec.md#single-use`, but `specs/business/invite/spec.md` does not exist") {
		t.Fatalf("drift of %s = %v", O, d)
	}
}
