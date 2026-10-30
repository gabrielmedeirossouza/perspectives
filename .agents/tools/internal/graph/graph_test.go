package graph

import (
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
	write(t, root, "perspectives/AGENTS.md", "# Perspectives\n")
	write(t, root, "src/AGENTS.md", "# Source\n")
	return root
}

const pSpec = "# Invite\n\nPurpose.\n\n## Expiry\n\n### Invite expiry\n\nThe invite **must** expire in 7 days.\n\n### Single use\n\nAn invite **must** be used once.\n"

func fixture(t *testing.T) string {
	t.Helper()
	root := gitInit(t)
	write(t, root, "perspectives/business/AGENTS.md", "# Business\n")
	write(t, root, "perspectives/business/invite/spec.md", pSpec)
	write(t, root, "src/specs/toolchain/spec.md", "# Toolchain\n\nPurpose.\n\n## Tests\n\n### Vitest as test runner\n\nTests **must** run on `vitest`.\n")
	write(t, root, "src/specs/toolchain/.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "src/specs/invite/spec.md", "# Invite\n\nPurpose.\n\n## HTTP\n\n### Invite endpoint\n\n`POST /invites` **must** create one. Serves **Invite expiry**.\n")
	write(t, root, "src/specs/invite/.metadata.json", `{"version": 1, "sources": [{"path": "perspectives/business/invite"}, {"path": "src/specs/toolchain"}]}`+"\n")
	write(t, root, "src/backend/invite/invite.ts", "export const days = 7;\n")
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "src/specs/toolchain"}, {"path": "perspectives/business/invite"}]}`+"\n")
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
	for _, rel := range []string{"perspectives/business/invite", "src/specs/toolchain", "src/specs/invite", "src/backend/invite"} {
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
	if !has(drift, "`perspectives/business/invite/spec.md` has no .metadata.json") || !has(drift, "`src/specs/invite/.metadata.json` does not match") {
		t.Fatalf("drift = %v", drift)
	}
	design := g.Design()
	if !has(design, "`src/specs/invite` is stale against `perspectives/business/invite`: added \"Invite expiry\", \"Single use\"") {
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
	write(t, root, "perspectives/business/invite/spec.md", strings.Replace(pSpec, "7 days", "14 days", 1))
	g := load(t, root)
	if !has(g.Design(), "`src/specs/invite` is stale against `perspectives/business/invite`: changed \"Invite expiry\"") {
		t.Fatalf("design = %v", g.Design())
	}
	units := g.Units()
	if len(units) != 1 || units[0].Node.Rel != "src/backend/invite" || !strings.Contains(units[0].Wait, "`src/specs/invite` is stale") {
		t.Fatalf("units = %+v", units)
	}
	if r := units[0].Rules; len(r) != 1 || r[0] != (RuleState{Title: "Invite expiry", State: "changed", Source: "perspectives/business/invite"}) {
		t.Fatalf("rules = %+v", r)
	}
	// /spec writes the metadata, /design acknowledges without changing the technical spec.
	sync(t, root, "perspectives/business/invite")
	g = sync(t, root, "src/specs/invite")
	if d := g.Design(); len(d) > 0 {
		t.Fatalf("design = %v", d)
	}
	run, wait := g.Batch(5)
	if len(run) != 1 || len(wait) != 0 || run[0].Node.Rel != "src/backend/invite" || run[0].Reason != "stale against `perspectives/business/invite`" {
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
	write(t, root, "perspectives/business/invite/spec.md", strings.Replace(pSpec, "### Single use", "### One use", 1))
	g := load(t, root)
	if !has(g.Design(), `renamed "Single use" → "One use"`) {
		t.Fatalf("design = %v", g.Design())
	}
	write(t, root, "perspectives/business/invite/spec.md", pSpec[:strings.Index(pSpec, "### Single use")])
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
	if err := os.RemoveAll(filepath.Join(root, "perspectives/business/invite")); err != nil {
		t.Fatal(err)
	}
	g := load(t, root)
	if !has(g.Design(), "`src/specs/invite` has a dangling source `perspectives/business/invite`") {
		t.Fatalf("design = %v", g.Design())
	}
	u := g.Units()
	if len(u) != 1 || !strings.Contains(u[0].Reason, "dangling source `perspectives/business/invite`") || !strings.Contains(u[0].Wait, "dangling") {
		t.Fatalf("units = %+v", u)
	}
	if _, err := g.Sync(g.Node("src/specs/invite"), time.Now()); err == nil {
		t.Fatal("sync accepted a dangling source")
	}
}

func TestCoverageOfCodeNode(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "src/specs/toolchain"}]}`+"\n")
	for _, rel := range []string{"perspectives/business/invite", "src/specs/toolchain", "src/specs/invite"} {
		sync(t, root, rel)
	}
	g := load(t, root)
	if u := g.Units(); len(u) != 1 || !strings.Contains(u[0].Reason, "missing source `perspectives/business/invite`") || has(g.Drift(), "but not") {
		t.Fatalf("units = %+v drift = %v", u, g.Drift())
	}
	if _, err := g.Sync(g.Node("src/backend/invite"), time.Now()); err == nil {
		t.Fatal("sync accepted a code node missing a served perspective")
	}
	write(t, root, "perspectives/user/home/spec.md", "# Home\n\n## S\n\n### Empty list\n\nShows nothing.\n")
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "src/specs/toolchain"}, {"path": "perspectives/business/invite"}, {"path": "perspectives/user/home"}]}`+"\n")
	g = load(t, root)
	if !has(g.Drift(), "`src/backend/invite` lists `perspectives/user/home`, but none of its technical specs serves it") || !has(g.Design(), "`perspectives/user/home` has no technical spec") {
		t.Fatalf("drift = %v design = %v", g.Drift(), g.Design())
	}
}

func TestInvalidEdgesUncoveredAndOrphans(t *testing.T) {
	root := gitInit(t)
	write(t, root, "perspectives/business/a/spec.md", "# A\n\n## S\n\n### R\n\nx\n")
	write(t, root, "perspectives/business/a/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/t"}]}`+"\n")
	write(t, root, "perspectives/business/b/.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "src/specs/t/spec.md", "# T\n\n## S\n\n### R\n\nx\n")
	write(t, root, "src/specs/t/.metadata.json", `{"version": 1, "sources": [{"path": "src/code"}, {"path": "src/specs/t"}]}`+"\n")
	write(t, root, "src/code/.metadata.json", `{"version": 2}`+"\n")
	write(t, root, "src/loose/x.ts", "x\n")
	write(t, root, "src/README.md", "fine\n")
	d := load(t, root).Drift()
	for _, want := range []string{
		"`perspectives/business/a` (perspective) lists `src/specs/t` (technical)",
		"`perspectives/business/b/.metadata.json` has no spec.md beside it",
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
	n := g.Node("perspectives/business/invite")
	m1, err := Refresh(n, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Save(n, m1); err != nil {
		t.Fatal(err)
	}
	g = load(t, root)
	n = g.Node("perspectives/business/invite")
	m2, _ := Refresh(n, time.Unix(200, 0))
	if written, _ := Save(n, m2); written || m2.Modified != "1970-01-01T00:01:40Z" {
		t.Fatalf("unchanged content rewrote metadata: written=%v modified=%s", written, m2.Modified)
	}
	write(t, root, "perspectives/business/invite/spec.md", pSpec+"\n### Third\n\nz\n")
	g = load(t, root)
	m3, _ := Refresh(g.Node("perspectives/business/invite"), time.Unix(300, 0))
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
	if want := "perspectives/business/invite src/specs/toolchain src/specs/invite src/backend/invite"; strings.Join(order, " ") != want {
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
	for _, rel := range []string{"perspectives/business/invite", "src/specs/toolchain", "src/specs/invite"} {
		sync(t, root, rel)
	}
	g := load(t, root)
	units := g.Units()
	if len(units) != 1 || units[0].Node.Rel != "src/backend/invite" || !strings.Contains(units[0].Reason, "missing source `perspectives/business/invite` (served by `src/specs/invite`)") {
		t.Fatalf("units = %+v", units)
	}
	if r := units[0].Rules; len(r) != 4 || r[0] != (RuleState{Title: "Invite endpoint", State: "new", Source: "src/specs/invite"}) || r[3] != (RuleState{Title: "Single use", State: "new", Source: "perspectives/business/invite"}) {
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
	write(t, root, "perspectives/business/invite/spec.md", strings.Replace(pSpec, "7 days", "14 days", 1))
	sync(t, root, "perspectives/business/invite")
	g := load(t, root)
	_, err := g.Sync(g.Node("src/backend/invite"), time.Now())
	if err == nil || !strings.Contains(err.Error(), "waits: `src/specs/invite` is stale against `perspectives/business/invite` (/design)") {
		t.Fatalf("sync accepted a waiting code node: %v", err)
	}
}

func TestDanglingSourceListsRecordedRulesAsRemoved(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	if err := os.RemoveAll(filepath.Join(root, "perspectives/business/invite")); err != nil {
		t.Fatal(err)
	}
	g := load(t, root)
	if !has(g.Design(), "`src/specs/invite` has a dangling source `perspectives/business/invite`: removed \"Invite expiry\", \"Single use\"") {
		t.Fatalf("design = %v", g.Design())
	}
	u := g.Units()
	if len(u) != 1 || len(u[0].Rules) != 2 || u[0].Rules[0] != (RuleState{Title: "Invite expiry", State: "removed", Source: "perspectives/business/invite"}) {
		t.Fatalf("units = %+v", u)
	}
}

func TestDesignGapParksRulesAndSpecGapAnnotates(t *testing.T) {
	root := fixture(t)
	settle(t, root)
	write(t, root, "src/specs/invite/spec.md", "# Invite\n\nPurpose.\n\n## HTTP\n\n### Invite endpoint\n\n`POST /invites` **must** create one. Serves **Invite expiry**.\n\n### Token storage\n\nStored somewhere.\n")
	sync(t, root, "src/specs/invite")
	write(t, root, "perspectives/TODO.md", "# TODO\n\n## Design gap\n\n- **Token storage** (`src/specs/invite`) names no persistence package.\n- [ ] **Ghost** (`src/specs/invite`) matches nothing.\n\n## Spec gap\n\n- **Invite expiry** (`business/invite`) leaves the grace period undefined.\n- broken line\n")
	g := load(t, root)
	u := g.Units()
	if len(u) != 1 || len(u[0].Rules) != 1 || u[0].Rules[0].Parked != "names no persistence package" || u[0].Wait != "every rule is parked under ## Design gap (/design)" {
		t.Fatalf("units = %+v", u)
	}
	if run, wait := g.Batch(5); len(run) != 0 || len(wait) != 1 {
		t.Fatalf("run = %d wait = %d", len(run), len(wait))
	}
	d := g.Drift()
	if !has(d, `design gap "Ghost" (`+"`src/specs/invite`"+`) matches no pending rule`) || !has(d, "spec gap out of shape: - broken line") {
		t.Fatalf("drift = %v", d)
	}
	write(t, root, "perspectives/business/invite/spec.md", strings.Replace(pSpec, "7 days", "14 days", 1))
	g = load(t, root)
	if !has(g.Design(), `changed "Invite expiry"; "Invite expiry" parked: leaves the grace period undefined (/spec)`) {
		t.Fatalf("design = %v", g.Design())
	}
}

func TestRefreshRefusesSpeclessNodes(t *testing.T) {
	root := gitInit(t)
	write(t, root, "perspectives/business/b/.metadata.json", `{"version": 1}`+"\n")
	write(t, root, "src/specs/t/.metadata.json", `{"version": 1}`+"\n")
	g := load(t, root)
	for _, rel := range []string{"perspectives/business/b", "src/specs/t"} {
		if _, err := Refresh(g.Node(rel), time.Now()); err == nil {
			t.Errorf("refresh accepted spec-less %s", rel)
		}
	}
}

func TestGlobalSpecWithoutCodeNodeNamesTheFix(t *testing.T) {
	root := fixture(t)
	write(t, root, "src/backend/invite/.metadata.json", `{"version": 1, "sources": [{"path": "src/specs/invite"}, {"path": "perspectives/business/invite"}]}`+"\n")
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
