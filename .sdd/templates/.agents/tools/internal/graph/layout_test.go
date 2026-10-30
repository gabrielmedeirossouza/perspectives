package graph

import (
	"os/exec"
	"strings"
	"testing"
)

// layout is a repository whose roots are not the defaults: product specs
// under product/, code under apps/code/, technical specs beside it under
// apps/design/.
func layout(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	write(t, root, ".sdd/.sdd-config.json", `{"version": 1, "paths": {"specs": "product", "src": "apps/code", "technical": "apps/design"}}`+"\n")
	write(t, root, "product/business/invite/spec.md", pSpec)
	write(t, root, "apps/design/invite/spec.md", "# Invite\n\nHow invites are built.\n\n## Place\n\n### Invite placement\n\nCode lives under `apps/code/invite/`.\n")
	write(t, root, "apps/design/invite/spec.md.metadata.json", `{"version": 2, "sources": [{"path": "product/business/invite/spec.md"}]}`+"\n")
	write(t, root, "apps/code/invite/invite.ts", "x\n")
	write(t, root, "apps/code/invite/invite.ts.metadata.json", `{"version": 2, "sources": [{"path": "apps/design/invite/spec.md"}, {"path": "product/business/invite/spec.md"}]}`+"\n")
	write(t, root, "apps/code/README.md", "chrome\n")
	write(t, root, "apps/code/stray/x.ts", "x\n")
	return root
}

func TestLayoutComesFromTheConfig(t *testing.T) {
	root := layout(t)
	g := load(t, root)
	for rel, want := range map[string]Kind{"product/business/invite/spec.md": Perspective, "apps/design/invite/spec.md": Technical, "apps/code/invite/invite.ts": Code} {
		if n := g.Node(rel); n == nil || n.Kind != want {
			t.Fatalf("%s: node %+v, want kind %s", rel, n, want)
		}
	}
	if g.Node("product") != nil || g.Node("apps") != nil || g.Node("src/specs/invite/spec.md") != nil {
		t.Fatal("a directory outside the layout became a node")
	}
	drift := g.Drift()
	if !has(drift, "`apps/code/stray/x.ts` is in no node: give it x.ts.metadata.json, or link it in apps/code/stray/.metadata.json") || has(drift, "README") {
		t.Fatalf("uncovered and chrome: %v", drift)
	}
	for _, rel := range []string{"product/business/invite/spec.md", "apps/design/invite/spec.md", "apps/code/invite/invite.ts"} {
		sync(t, root, rel)
	}
	g = load(t, root)
	if d := g.Design(); len(d) > 0 {
		t.Fatalf("design = %v", d)
	}
	if u := g.Units(); len(u) > 0 {
		t.Fatalf("units = %+v", u)
	}
	if d := g.Drift(); len(d) != 1 || !strings.Contains(d[0], "stray") {
		t.Fatalf("drift = %v", d)
	}
	// A legacy .metadata.json beside a technical spec of a beside-layout is a stray too.
	write(t, root, "apps/design/old/spec.md", "# Old\n")
	write(t, root, "apps/design/old/.metadata.json", `{"version": 1}`+"\n")
	if d := load(t, root).Drift(); !has(d, "`apps/design/old/.metadata.json` is no node: the node of a spec is spec.md.metadata.json") {
		t.Fatalf("drift = %v", d)
	}
}

func TestGapsReadTodoOfTheSpecsRoot(t *testing.T) {
	root := layout(t)
	write(t, root, "product/TODO.md", "# TODO\n\n## Spec gap\n\n- **Invite expiry** (`business/invite`) leaves the grace period undefined.\n- **Ghost** (`business/invite`) matches nothing.\n- broken line\n\n## Audit gap\n\n- **Rate limiting** (`apps/code/invite`) code obeys a limit no technical spec states.\n")
	g := load(t, root)
	if gaps := g.gapsFor("spec", "product/business/invite/spec.md"); len(gaps) != 2 {
		t.Fatalf("spec gaps = %+v", gaps)
	}
	if s := g.AuditOf(g.Node("apps/code/invite/invite.ts")); s.Parked != "code obeys a limit no technical spec states" {
		t.Fatalf("audit state = %+v", s)
	}
	drift := g.Drift()
	if !has(drift, "product/TODO.md: spec gap out of shape: - broken line") || !has(drift, "product/TODO.md: spec gap \"Ghost\" (`product/business/invite`) matches no rule") {
		t.Fatalf("drift = %v", drift)
	}
}

func TestNamesReUsesTheCodeRoot(t *testing.T) {
	re := namesRe("apps/code/invite", "apps/code")
	for text, want := range map[string]bool{
		"Code lives under `apps/code/invite/`.": true,
		"Not apps/code/invitee.":                false,
		"Only apps/code/ in general.":           false,
		"Not code/invite either.":               false,
	} {
		if got := re.MatchString(text); got != want {
			t.Errorf("%q: names = %v, want %v", text, got, want)
		}
	}
	if !namesRe("apps/code", "apps/code").MatchString("Manifests live in `apps/code/`.") {
		t.Error("code root: apps/code/ not matched")
	}
}
