package graph

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
	write(t, root, "specs/business/invite/spec.md", pSpec)
	write(t, root, "src/specs/toolchain/spec.md", "# Toolchain\n\nPurpose.\n\n## Tests\n\n### Vitest as test runner\n\nTests **must** run on `vitest`.\n")
	write(t, root, "src/specs/toolchain/.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "src/specs/invite/spec.md", "# Invite\n\nPurpose.\n\n## HTTP\n\n### Invite endpoint\n\n`POST /invites` **must** create one. Serves **Invite expiry**.\n")
	write(t, root, "src/specs/invite/.metadata.json", `{"version": 1, "sources": [{"path": "specs/business/invite"}, {"path": "src/specs/toolchain"}]}`+"\n")
	write(t, root, "src/backend/invite/invite.ts", "export const days = 7;\n")
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "src/specs/toolchain"}, {"path": "specs/business/invite"}]}`+"\n")
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
	for _, rel := range []string{"specs/business/invite", "src/specs/toolchain", "src/specs/invite", "src/backend/invite"} {
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
	if !has(drift, "`specs/business/invite/spec.md` has no .metadata.json") || !has(drift, "`src/specs/invite/.metadata.json` does not match") {
		t.Fatalf("drift = %v", drift)
	}
	design := g.Design()
	if !has(design, "`src/specs/invite` is stale against `specs/business/invite`: added \"Invite expiry\", \"Single use\"") {
		t.Fatalf("design = %v", design)
	}
	units := g.Units()
	if len(units) != 1 || units[0].Node.Rel != "src/backend/invite" || units[0].Wait == "" {
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
	c := g.Node("src/backend/invite")
	if c.Meta.Sources[2].Rules["Invite expiry"] == "" || c.Meta.Sources[0].Hash != g.Node("src/specs/invite").Hash {
		t.Fatalf("sync did not record sources: %+v", c.Meta.Sources)
	}
}

func TestPolicyChangeReachesDesignThenCode(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "specs/business/invite/spec.md", strings.Replace(pSpec, "7 days", "14 days", 1))
	g := load(t, root)
	if !has(g.Design(), "`src/specs/invite` is stale against `specs/business/invite`: changed \"Invite expiry\"") {
		t.Fatalf("design = %v", g.Design())
	}
	units := g.Units()
	if len(units) != 1 || units[0].Node.Rel != "src/backend/invite" || !strings.Contains(units[0].Wait, "`src/specs/invite` is stale") {
		t.Fatalf("units = %+v", units)
	}
	if r := units[0].Rules; len(r) != 1 || r[0] != (RuleState{Title: "Invite expiry", State: "changed", Source: "specs/business/invite"}) {
		t.Fatalf("rules = %+v", r)
	}
	// /sdd-spec writes the metadata, /sdd-design acknowledges without changing the technical spec.
	sync(t, root, "specs/business/invite")
	g = sync(t, root, "src/specs/invite")
	if d := g.Design(); len(d) > 0 {
		t.Fatalf("design = %v", d)
	}
	run, wait := g.Batch(5)
	if len(run) != 1 || len(wait) != 0 || run[0].Node.Rel != "src/backend/invite" || run[0].Reason != "stale against `specs/business/invite`" {
		t.Fatalf("run = %+v wait = %+v", run, wait)
	}
	g = sync(t, root, "src/backend/invite")
	if u := g.Units(); len(u) > 0 {
		t.Fatalf("units after sync = %+v", u)
	}
}

func TestRenameAndRemove(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "specs/business/invite/spec.md", strings.Replace(pSpec, "### Single use", "### One use", 1))
	g := load(t, root)
	if !has(g.Design(), `renamed "Single use" → "One use"`) {
		t.Fatalf("design = %v", g.Design())
	}
	write(t, root, "specs/business/invite/spec.md", pSpec[:strings.Index(pSpec, "### Single use")])
	g = load(t, root)
	if !has(g.Design(), `removed "Single use"`) {
		t.Fatalf("design = %v", g.Design())
	}
	if u := g.Units(); len(u) != 1 || u[0].Rules[0].State != "removed" {
		t.Fatalf("units = %+v", u)
	}
}

func TestDanglingSource(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	if err := os.RemoveAll(filepath.Join(root, "specs/business/invite")); err != nil {
		t.Fatal(err)
	}
	g := load(t, root)
	if !has(g.Design(), "`src/specs/invite` has a dangling source `specs/business/invite`") {
		t.Fatalf("design = %v", g.Design())
	}
	u := g.Units()
	if len(u) != 1 || !strings.Contains(u[0].Reason, "dangling source `specs/business/invite`") || !strings.Contains(u[0].Wait, "dangling") {
		t.Fatalf("units = %+v", u)
	}
	if _, err := g.Sync(g.Node("src/specs/invite"), time.Now()); err == nil {
		t.Fatal("sync accepted a dangling source")
	}
}

func TestCoverageOfCodeNode(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "src/specs/toolchain"}]}`+"\n")
	for _, rel := range []string{"specs/business/invite", "src/specs/toolchain", "src/specs/invite"} {
		sync(t, root, rel)
	}
	g := load(t, root)
	if u := g.Units(); len(u) != 1 || !strings.Contains(u[0].Reason, "missing source `specs/business/invite`") || has(g.Drift(), "but not") {
		t.Fatalf("units = %+v drift = %v", u, g.Drift())
	}
	if _, err := g.Sync(g.Node("src/backend/invite"), time.Now()); err == nil {
		t.Fatal("sync accepted a code node missing a served perspective")
	}
	write(t, root, "specs/user/home/spec.md", "# Home\n\n## S\n\n### Empty list\n\nShows nothing.\n")
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "src/specs/toolchain"}, {"path": "specs/business/invite"}, {"path": "specs/user/home"}]}`+"\n")
	g = load(t, root)
	if !has(g.Drift(), "`src/backend/invite` lists `specs/user/home`, but none of its technical specs serves it") || !has(g.Design(), "`specs/user/home` has no technical spec") {
		t.Fatalf("drift = %v design = %v", g.Drift(), g.Design())
	}
}

func TestInvalidEdgesUncoveredAndOrphans(t *testing.T) {
	root := gitInit(t)
	write(t, root, "specs/business/a/spec.md", "# A\n\n## S\n\n### R\n\nx\n")
	write(t, root, "specs/business/a/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/t"}]}`+"\n")
	write(t, root, "specs/business/b/.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "src/specs/t/spec.md", "# T\n\n## S\n\n### R\n\nx\n")
	write(t, root, "src/specs/t/.metadata.json", `{"version": 1, "sources": [{"path": "src/code"}, {"path": "src/specs/t"}]}`+"\n")
	write(t, root, "src/code/.metadata.json", `{"version": 2}`+"\n")
	write(t, root, "src/loose/x.ts", "x\n")
	write(t, root, "src/README.md", "fine\n")
	d := load(t, root).Drift()
	for _, want := range []string{
		"`specs/business/a` (perspective) lists `src/specs/t` (technical)",
		"`specs/business/b/.metadata.json` has no spec.md beside it",
		"`src/specs/t` (technical) lists `src/code` (code)",
		"`src/specs/t` lists itself",
		"`src/code/.metadata.json`: version 2 is not supported",
		"`src/loose/` holds 1 file(s) outside any node",
	} {
		if !has(d, want) {
			t.Errorf("drift lacks %q:\n%s", want, strings.Join(d, "\n"))
		}
	}
	if has(d, "README") {
		t.Errorf("src/*.md reported as uncovered: %v", d)
	}
}

func TestBatchOrderAndCap(t *testing.T) {
	root := gitInit(t)
	for _, n := range []string{"a", "b", "c", "d", "e", "f"} {
		write(t, root, "src/specs/"+n+"/spec.md", "# "+n+"\n\n## S\n\n### Rule "+n+"\n\nx\n")
		src := `{"version": 1}`
		if n == "a" {
			src = `{"version": 1, "sources": [{"path": "src/specs/f"}]}`
		}
		write(t, root, "src/specs/"+n+"/.metadata.json", src+"\n")
	}
	for _, n := range []string{"f", "a", "b", "c", "d", "e"} {
		sync(t, root, "src/specs/"+n)
	}
	g := load(t, root)
	run, wait := g.Batch(5)
	if len(run) != 5 || len(wait) != 0 {
		t.Fatalf("run = %d wait = %d", len(run), len(wait))
	}
	var order []string
	for _, u := range g.Units() {
		order = append(order, u.Node.Rel)
	}
	if want := "src/specs/b src/specs/c src/specs/d src/specs/e src/specs/f src/specs/a"; strings.Join(order, " ") != want {
		t.Fatalf("order = %v", order)
	}
	if r := run[0].Rules; len(r) != 1 || r[0].State != "new" {
		t.Fatalf("rules = %+v", r)
	}
}

func TestRefreshIsStableAndMovesModifiedOnlyOnChange(t *testing.T) {
	root := fixture(t)
	g := load(t, root)
	n := g.Node("specs/business/invite")
	m1, err := Refresh(n, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Save(n, m1); err != nil {
		t.Fatal(err)
	}
	g = load(t, root)
	n = g.Node("specs/business/invite")
	m2, _ := Refresh(n, time.Unix(200, 0))
	if written, _ := Save(n, m2); written || m2.Modified != "1970-01-01T00:01:40Z" {
		t.Fatalf("unchanged content rewrote metadata: written=%v modified=%s", written, m2.Modified)
	}
	write(t, root, "specs/business/invite/spec.md", pSpec+"\n### Third\n\nz\n")
	g = load(t, root)
	m3, _ := Refresh(g.Node("specs/business/invite"), time.Unix(300, 0))
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
	if want := "specs/business/invite src/specs/toolchain src/specs/invite src/backend/invite"; strings.Join(order, " ") != want {
		t.Fatalf("order = %v", order)
	}
}

func TestStagedListsBothNodesOfARename(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	write(t, root, "src/backend/other/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/toolchain"}]}`+"\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	git("mv", "src/backend/invite/invite.ts", "src/backend/other/invite.ts")
	g := load(t, root)
	nodes, err := g.Staged()
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, n := range nodes {
		rels = append(rels, n.Rel)
	}
	if want := "src/backend/invite src/backend/other"; strings.Join(rels, " ") != want {
		t.Fatalf("staged = %v, want %s", rels, want)
	}
}

func TestMissingServedPerspectiveIsAnImplementUnit(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "src/specs/toolchain"}]}`+"\n")
	for _, rel := range []string{"specs/business/invite", "src/specs/toolchain", "src/specs/invite"} {
		sync(t, root, rel)
	}
	g := load(t, root)
	units := g.Units()
	if len(units) != 1 || units[0].Node.Rel != "src/backend/invite" || !strings.Contains(units[0].Reason, "missing source `specs/business/invite` (served by `src/specs/invite`)") {
		t.Fatalf("units = %+v", units)
	}
	if r := units[0].Rules; len(r) != 4 || r[0] != (RuleState{Title: "Invite endpoint", State: "new", Source: "src/specs/invite"}) || r[3] != (RuleState{Title: "Single use", State: "new", Source: "specs/business/invite"}) {
		t.Fatalf("rules = %+v", r)
	}
	if has(g.Drift(), "missing") {
		t.Fatalf("a missing served perspective is pending work, not drift: %v", g.Drift())
	}
	if _, err := g.Sync(g.Node("src/backend/invite"), time.Now()); err == nil || !strings.Contains(err.Error(), "add it to sources") {
		t.Fatalf("sync did not refuse with guidance: %v", err)
	}
}

func TestSyncRefusesACodeNodeThatWaits(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "specs/business/invite/spec.md", strings.Replace(pSpec, "7 days", "14 days", 1))
	sync(t, root, "specs/business/invite")
	g := load(t, root)
	_, err := g.Sync(g.Node("src/backend/invite"), time.Now())
	if err == nil || !strings.Contains(err.Error(), "waits: `src/specs/invite` is stale against `specs/business/invite` (/sdd-design)") {
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
	if !has(g.Design(), "`src/specs/invite` has a dangling source `specs/business/invite`: removed \"Invite expiry\", \"Single use\"") {
		t.Fatalf("design = %v", g.Design())
	}
	u := g.Units()
	if len(u) != 1 || len(u[0].Rules) != 2 || u[0].Rules[0] != (RuleState{Title: "Invite expiry", State: "removed", Source: "specs/business/invite"}) {
		t.Fatalf("units = %+v", u)
	}
}

func TestDesignGapParksRulesAndSpecGapAnnotates(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "src/specs/invite/spec.md", "# Invite\n\nPurpose.\n\n## HTTP\n\n### Invite endpoint\n\n`POST /invites` **must** create one. Serves **Invite expiry**.\n\n### Token storage\n\nStored somewhere.\n")
	sync(t, root, "src/specs/invite")
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
	write(t, root, "specs/business/invite/spec.md", strings.Replace(pSpec, "7 days", "14 days", 1))
	g = load(t, root)
	if !has(g.Design(), `changed "Invite expiry"; "Invite expiry" parked: leaves the grace period undefined (/sdd-spec)`) {
		t.Fatalf("design = %v", g.Design())
	}
}

func TestRefreshRefusesSpeclessNodes(t *testing.T) {
	root := gitInit(t)
	write(t, root, "specs/business/b/.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "src/specs/t/.metadata.json", `{"version": 1}`+"\n")
	g := load(t, root)
	for _, rel := range []string{"specs/business/b", "src/specs/t"} {
		if _, err := Refresh(g.Node(rel), time.Now()); err == nil {
			t.Errorf("refresh accepted spec-less %s", rel)
		}
	}
}

func TestGlobalSpecWithoutCodeNodeNamesTheFix(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "specs/business/invite"}]}`+"\n")
	g := settle(t, root)
	u := g.Units()
	if len(u) != 1 || u[0].Node.Rel != "src/specs/toolchain" || u[0].Reason != "no code node lists it; list it from the code nodes it constrains" {
		t.Fatalf("units = %+v", u)
	}
}

func TestChromeUnderSrcIsNoContent(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/toolchain"}]}`+"\n")
	write(t, root, "src/go.mod", "module x\n")
	g := settle(t, root)
	sync(t, root, "src")
	g = load(t, root)
	before := g.Node("src").Hash
	write(t, root, "src/BUG-AUTOPSIES.md", "# Bug autopsies\n\nnew entry\n")
	g = load(t, root)
	if g.Node("src").Hash != before || len(g.Units()) > 0 || has(g.Drift(), "src/") {
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
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "specs/business/invite"}]}`+"\n")
	for _, rel := range []string{"specs/business/invite", "src/specs/toolchain", "src/specs/invite"} {
		sync(t, root, rel)
	}
	g := load(t, root)
	d := g.Drift()
	if !has(d, "`src/backend/invite` lists no technical spec; a code node lists at least one") || has(d, "none of its technical specs serves it") {
		t.Fatalf("drift = %v", d)
	}
	if _, err := g.Sync(g.Node("src/backend/invite"), time.Now()); err == nil || !strings.Contains(err.Error(), "lists no technical spec") || strings.Contains(err.Error(), "none of its technical specs serves it") {
		t.Fatalf("sync on a code node with no technical spec: %v", err)
	}
	if s := g.AuditOf(g.Node("src/backend/invite")); s.Unit == "" || len(s.Blocked) == 0 {
		t.Fatalf("an unsynced perspective is /sdd-implement's first and a hand-written file is drift: %+v", s)
	}
	// With no source at all, once written, nothing blocks the judge: it proposes the spec.
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1}`+"\n")
	g = load(t, root)
	m, err := Refresh(g.Node("src/backend/invite"), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	Save(g.Node("src/backend/invite"), m)
	g = load(t, root)
	if s := g.AuditOf(g.Node("src/backend/invite")); len(s.Blocked) > 0 || !s.Owed() || !has(g.Drift(), "lists no technical spec") {
		t.Fatalf("no technical spec must not block the judge: %+v drift=%v", s, g.Drift())
	}
	// A dangling technical source is reported as dangling only.
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/gone"}, {"path": "specs/business/invite"}]}`+"\n")
	g = load(t, root)
	if d := g.Drift(); has(d, "no technical spec") || has(d, "none of its technical specs serves it") {
		t.Fatalf("dangling technical source reported as another drift: %v", d)
	}
	m, _ = Refresh(g.Node("src/backend/invite"), time.Unix(0, 0))
	Save(g.Node("src/backend/invite"), m)
	g = load(t, root)
	if _, err := g.Approve(g.Node("src/backend/invite"), time.Now()); err == nil || !strings.Contains(err.Error(), "dangling source `src/specs/gone`") {
		t.Fatalf("approve on a dangling source: %v", err)
	}
	if u, ok := g.codeUnit(g.Node("src/backend/invite")); !ok || !strings.Contains(u.Reason, "dangling source `src/specs/gone`") {
		t.Fatalf("units = %+v", u)
	}
	if s := g.AuditOf(g.Node("src/backend/invite")); s.Unit == "" || s.Owed() {
		t.Fatalf("a dangling source is /sdd-implement's first: %+v", s)
	}
}

func TestAuditedStartsAtZeroAndResetsOnContentChangeOrNewSource(t *testing.T) {
	root := fixture(t)
	g := settle(t, root)
	c := g.Node("src/backend/invite")
	if c.Meta.Audited == nil || *c.Meta.Audited != 0 || g.Node("src/specs/invite").Meta.Audited != nil || g.Node("specs/business/invite").Meta.Audited != nil {
		t.Fatalf("audited after settle: code=%v technical=%v", c.Meta.Audited, g.Node("src/specs/invite").Meta.Audited)
	}
	if s := g.AuditOf(c); !s.Owed() {
		t.Fatalf("settled code node not owed: %+v", s)
	}
	g = approve(t, root, "src/backend/invite")
	if a := g.Node("src/backend/invite").Meta.Audited; a == nil || *a != 1 {
		t.Fatalf("audited after approve = %v", a)
	}
	if s := g.AuditOf(g.Node("src/backend/invite")); s.Owed() || s.Audited != 1 {
		t.Fatalf("approved node still owed: %+v", s)
	}
	m, _ := Refresh(g.Node("src/backend/invite"), time.Unix(50, 0))
	if written, _ := Save(g.Node("src/backend/invite"), m); written || *m.Audited != 1 {
		t.Fatalf("refresh without change touched audited: written=%v audited=%d", written, *m.Audited)
	}
	write(t, root, "src/backend/invite/invite.ts", "export const days = 14;\n")
	g = load(t, root)
	if s := g.AuditOf(g.Node("src/backend/invite")); s.Audited != 0 {
		t.Fatalf("changed content still counts as audited: %+v", s)
	}
	m, _ = Refresh(g.Node("src/backend/invite"), time.Unix(60, 0))
	if *m.Audited != 0 {
		t.Fatalf("refresh after change kept audited = %d", *m.Audited)
	}
	Save(g.Node("src/backend/invite"), m)
	g = approve(t, root, "src/backend/invite")
	// A source listed since the last sync returns audited to 0 when recorded.
	write(t, root, "src/specs/extra/spec.md", "# Extra\n\nPurpose.\n\n## S\n\n### Extra rule\n\nx\n")
	write(t, root, "src/specs/extra/.metadata.json", `{"version": 1}`+"\n")
	sync(t, root, "src/specs/extra")
	file := string(readFile(t, root, "src/backend/invite/.metadata.json"))
	write(t, root, "src/backend/invite/.metadata.json", strings.Replace(file, `"path": "src/specs/invite"`, `"path": "src/specs/extra"`+"\n    },\n    {\n      "+`"path": "src/specs/invite"`, 1))
	g = load(t, root)
	if a := g.Node("src/backend/invite").Meta.Audited; a == nil || *a != 1 {
		t.Fatalf("editing sources moved audited: %v", a)
	}
	if _, err := g.Approve(g.Node("src/backend/invite"), time.Now()); err == nil || !strings.Contains(err.Error(), "/sdd-implement unit") {
		t.Fatalf("approve accepted a node with an unsynced source: %v", err)
	}
	g = sync(t, root, "src/backend/invite")
	if a := g.Node("src/backend/invite").Meta.Audited; a == nil || *a != 0 {
		t.Fatalf("sync of a new source kept audited = %v", a)
	}
	approve(t, root, "src/backend/invite")
	g = sync(t, root, "src/backend/invite")
	if a := g.Node("src/backend/invite").Meta.Audited; *a != 1 {
		t.Fatalf("sync without a new source moved audited = %d", *a)
	}
}

func TestApproveRefusals(t *testing.T) {
	root := fixture(t)
	g := settle(t, root)
	if _, err := g.Approve(g.Node("src/specs/invite"), time.Now()); err == nil || !strings.Contains(err.Error(), "only a code node") {
		t.Fatalf("approve accepted a technical node: %v", err)
	}
	write(t, root, "src/backend/invite/invite.ts", "changed\n")
	g = load(t, root)
	if _, err := g.Approve(g.Node("src/backend/invite"), time.Now()); err == nil || !strings.Contains(err.Error(), "changed since its metadata was last written") {
		t.Fatalf("approve accepted a moved hash: %v", err)
	}
	write(t, root, "src/backend/invite/invite.ts", "export const days = 7;\n")
	write(t, root, "specs/business/invite/spec.md", strings.Replace(pSpec, "7 days", "14 days", 1))
	sync(t, root, "specs/business/invite")
	g = load(t, root)
	if _, err := g.Approve(g.Node("src/backend/invite"), time.Now()); err == nil || !strings.Contains(err.Error(), "/sdd-implement unit (stale against") {
		t.Fatalf("approve accepted a stale node: %v", err)
	}
}

func TestApproveAcceptsANodeThatOnlyWaits(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "specs/business/invite"}]}`+"\n")
	settle(t, root)
	write(t, root, "src/specs/toolchain/spec.md", "# Toolchain\n\nPurpose.\n\n## Tests\n\n### Vitest as test runner\n\nTests **must** run on `vitest` 2.\n")
	sync(t, root, "src/specs/toolchain")
	g := load(t, root)
	if !has(g.Design(), "`src/specs/invite` is stale against `src/specs/toolchain`") {
		t.Fatalf("design = %v", g.Design())
	}
	if u, ok := g.codeUnit(g.Node("src/backend/invite")); ok {
		t.Fatalf("a node whose own sources are fresh is no unit: %+v", u)
	}
	if _, err := g.Approve(g.Node("src/backend/invite"), time.Now()); err != nil {
		t.Fatalf("approve refused a node that only waits: %v", err)
	}
}

func TestDossierTiersCandidatesAndNamesCorrections(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/specs/http-client/spec.md", "# HTTP client\n\nHow HTTP calls are made.\n\n## Client\n\n### Single client\n\nCode under `src/backend/invite/` **must** use `acme-http`.\n")
	write(t, root, "src/specs/http-client/.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "src/specs/other/spec.md", "# Other\n\nSomething else entirely.\n\n## S\n\n### Other rule\n\nx\n")
	write(t, root, "src/specs/other/.metadata.json", `{"version": 1, "sources": [{"path": "specs/business/invite"}]}`+"\n")
	write(t, root, "specs/user/home/spec.md", "# Home\n\nPurpose.\n\n## S\n\n### Empty list\n\nShows nothing.\n")
	write(t, root, "src/specs/invite/.metadata.json", `{"version": 1, "sources": [{"path": "specs/business/invite"}, {"path": "specs/user/home"}, {"path": "src/specs/toolchain"}]}`+"\n")
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "specs/business/invite"}]}`+"\n")
	for _, rel := range []string{"specs/business/invite", "specs/user/home", "src/specs/toolchain", "src/specs/http-client", "src/specs/other", "src/specs/invite"} {
		sync(t, root, rel)
	}
	g := load(t, root)
	d := g.Dossier(g.Node("src/backend/invite"))
	if strings.Join(d.Files, ",") != "invite.ts" || len(d.Technical) != 1 || d.Technical[0].Rel != "src/specs/invite" || len(d.Perspectives) != 1 {
		t.Fatalf("dossier = %+v", d)
	}
	var cands []string
	for _, c := range d.Candidates {
		cands = append(cands, fmt.Sprintf("%s:%d:%v:%v:%s", c.Node.Rel, c.Tier(), c.Names, c.Global, c.Purpose))
	}
	want := "src/specs/http-client:1:true:true:How HTTP calls are made. src/specs/toolchain:2:false:true:Purpose. src/specs/other:3:false:false:Something else entirely."
	if strings.Join(cands, " ") != want {
		t.Fatalf("candidates = %v", cands)
	}
	if len(d.DropPerspectives) != 0 {
		t.Fatalf("drop = %+v", d.DropPerspectives)
	}
	if s := d.State; s.Unit == "" || s.Owed() {
		t.Fatalf("a missing served perspective is /sdd-implement's first: %+v", s)
	}
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "specs/business/invite"}, {"path": "specs/user/home"}]}`+"\n")
	sync(t, root, "src/backend/invite")
	write(t, root, "src/specs/invite/.metadata.json", `{"version": 1, "sources": [{"path": "specs/business/invite"}, {"path": "src/specs/toolchain"}]}`+"\n")
	sync(t, root, "src/specs/invite")
	g = load(t, root)
	d = g.Dossier(g.Node("src/backend/invite"))
	if len(d.DropPerspectives) != 1 || d.DropPerspectives[0] != "`specs/user/home` (none of its technical specs serves it)" {
		t.Fatalf("drop = %+v", d.DropPerspectives)
	}
	if nodes := g.CodeNodesUnder("src/backend"); len(nodes) != 1 || nodes[0].Rel != "src/backend/invite" || len(g.CodeNodesUnder("src/specs")) != 0 {
		t.Fatalf("CodeNodesUnder = %+v", nodes)
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
		"Links to specs/business/invite.":       false,
	} {
		if got := re.MatchString(text); got != want {
			t.Errorf("%q: names = %v, want %v", text, got, want)
		}
	}
	if !namesRe("src", "src").MatchString("Manifests live in `src/`.") {
		t.Error("src node: src/ not matched")
	}
}

func TestAuditGapParksANodeAndOrphansAreDrift(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "specs/TODO.md", "# TODO\n\n## Audit gap\n\n- **Rate limiting** (`src/backend/invite`) code obeys a limit no technical spec states.\n- **Ghost** (`src/nowhere`) nothing.\n")
	g := load(t, root)
	s := g.AuditOf(g.Node("src/backend/invite"))
	if s.Parked != "code obeys a limit no technical spec states" || s.Owed() {
		t.Fatalf("state = %+v", s)
	}
	d := g.Drift()
	if !has(d, "audit gap \"Ghost\" (`src/nowhere`) names no code node that owes an audit") || has(d, "Rate limiting") {
		t.Fatalf("drift = %v", d)
	}
	approve(t, root, "src/backend/invite")
	if d := load(t, root).Drift(); !has(d, "audit gap \"Rate limiting\" (`src/backend/invite`) names no code node that owes an audit") {
		t.Fatalf("drift after approve = %v", d)
	}
}

func TestAuditGapMayNameTheRootCodeNode(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/toolchain"}]}`+"\n")
	write(t, root, "src/go.mod", "module x\n")
	settle(t, root)
	sync(t, root, "src")
	write(t, root, "specs/TODO.md", "# TODO\n\n## Audit gap\n\n- **Root manifests** (`src`) nothing decides them.\n")
	g := load(t, root)
	if s := g.AuditOf(g.Node("src")); s.Parked != "nothing decides them" {
		t.Fatalf("state = %+v", s)
	}
	if has(g.Drift(), "Root manifests") {
		t.Fatalf("drift = %v", g.Drift())
	}
}

func TestApproveOnANeverWrittenSkeletonNamesTheDrift(t *testing.T) {
	root := fixture(t)
	for _, rel := range []string{"specs/business/invite", "src/specs/toolchain", "src/specs/invite"} {
		sync(t, root, rel)
	}
	g := load(t, root)
	_, err := g.Approve(g.Node("src/backend/invite"), time.Now())
	if err == nil || !strings.Contains(err.Error(), "does not match the content; run metadata --write") || strings.Contains(err.Error(), "changed since") {
		t.Fatalf("approve on a skeleton: %v", err)
	}
}

func TestAuditedPlacementIsDrift(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	code := string(readFile(t, root, "src/backend/invite/.metadata.json"))
	write(t, root, "src/backend/invite/.metadata.json", strings.Replace(code, "  \"audited\": 0,\n", "", 1))
	tech := string(readFile(t, root, "src/specs/invite/.metadata.json"))
	write(t, root, "src/specs/invite/.metadata.json", strings.Replace(tech, "  \"rules\":", "  \"audited\": 0,\n  \"rules\":", 1))
	g := load(t, root)
	d := g.Drift()
	if !has(d, "`src/backend/invite/.metadata.json` carries no audited; run metadata --write src/backend/invite") || !has(d, "`src/specs/invite/.metadata.json` carries audited, which only a code node has") {
		t.Fatalf("drift = %v", d)
	}
	if s := g.AuditOf(g.Node("src/backend/invite")); len(s.Blocked) == 0 {
		t.Fatalf("a missing audited must block the judge: %+v", s)
	}
	for _, rel := range []string{"src/specs/invite", "src/backend/invite"} {
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
