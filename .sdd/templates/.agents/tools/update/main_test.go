package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tools/internal/config"
)

func testConfig() (config.Config, config.Config) {
	cfg := config.Default()
	cfg.Version = 0
	cfg.Paths = config.Paths{Specs: "product", Src: "apps/code", Technical: "apps/design"}
	cfg.Perspectives = []config.Perspective{{Name: "business", From: "business"}, {Name: "player", From: "user"}, {Name: "coach", From: "user"}, {Name: "ops"}}
	cfg.Skills = []string{"sdd-spec", "sdd-commit"}
	tpl := config.Default()
	tpl.Version = 7
	tpl.Perspectives = []config.Perspective{{Name: "business", From: "business"}, {Name: "user", From: "user"}, {Name: "operator", From: "operator"}}
	return cfg, tpl
}

func TestPathsMapByTheConfig(t *testing.T) {
	cfg, tpl := testConfig()
	m := newMapper(cfg, tpl)
	for p, want := range map[string][]string{
		"AGENTS.md":                          {"AGENTS.md"},
		"specs/AGENTS.md":                    {"product/AGENTS.md"},
		"specs/PERSPECTIVES.md":              {"product/PERSPECTIVES.md"},
		"specs/business/AGENTS.md":           {"product/business/AGENTS.md"},
		"specs/user/AGENTS.md":               {"product/player/AGENTS.md", "product/coach/AGENTS.md"},
		"src/AGENTS.md":                      {"apps/code/AGENTS.md"},
		"src/specs/x/spec.md":                {"apps/design/x/spec.md"},
		".agents/skills/sdd-spec/SKILL.md":   {".agents/skills/sdd-spec/SKILL.md"},
		".claude/skills/sdd-commit":          {".claude/skills/sdd-commit"},
		".githooks/pre-commit":               {".githooks/pre-commit"},
		"specs/operator/AGENTS.md":           nil,
		".agents/skills/sdd-design/SKILL.md": nil,
		".claude/skills/sdd-design":          nil,
		".sdd/.sdd-config.json":              nil,
	} {
		got, note := m.paths(p)
		if strings.Join(got, ",") != strings.Join(want, ",") || (want == nil) != (note != "") {
			t.Errorf("%s: paths = %v (%q), want %v", p, got, note, want)
		}
	}
}

func TestTokensAreReplacedOncePerPosition(t *testing.T) {
	cfg, tpl := testConfig()
	m := newMapper(cfg, tpl)
	in := "see `src/specs/x`, `specs/business/y`, `src/z`, `specs/user/w`, and specs/AGENTS.md; not srcs/ nor myspecs/."
	want := "see `apps/design/x`, `product/business/y`, `apps/code/z`, `product/user/w`, and product/AGENTS.md; not srcs/ nor myproduct/."
	if got := string(m.replace([]byte(in))); got != want {
		t.Fatalf("replace =\n%s\nwant\n%s", got, want)
	}
	cfg.Perspectives = []config.Perspective{{Name: "player", From: "user"}}
	m = newMapper(cfg, tpl)
	if got := string(m.replace([]byte("specs/user/home"))); got != "product/player/home" {
		t.Fatalf("single mapping: %s", got)
	}
	plain := newMapper(config.Default(), tpl)
	if plain.active || string(plain.replace([]byte(in))) != in {
		t.Fatalf("identity layout rewrites: %v", plain.tokens)
	}
	only := config.Default()
	only.Paths.Specs = "product"
	m = newMapper(only, tpl)
	if got := string(m.replace([]byte("`src/specs/x` and `specs/y`"))); got != "`src/specs/x` and `product/y`" {
		t.Fatalf("technical root unshielded: %s", got)
	}
}

func shell(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const tplConfig = `{
  "version": %d,
  "remote": "r",
  "paths": { "specs": "specs", "src": "src", "technical": "src/specs" },
  "perspectives": [
    { "name": "business", "from": "business" },
    { "name": "user", "from": "user" },
    { "name": "operator", "from": "operator" }
  ]
}
`

// upstream builds a framework repository with two versions on main and
// returns its root and the commit of version 1.
func upstream(t *testing.T) (string, string) {
	t.Helper()
	up := t.TempDir()
	shell(t, up, "init", "-q", "-b", "main")
	shell(t, up, "config", "user.email", "t@t")
	shell(t, up, "config", "user.name", "t")
	tpl := func(rel string) string { return ".sdd/templates/" + rel }
	write(t, up, tpl(".sdd/.sdd-config.json"), strings.ReplaceAll(tplConfig, "%d", "1"))
	write(t, up, tpl("AGENTS.md"), "# Law\n\nProduct specs (`specs/`) and code (`src/`).\n\nLine two.\n")
	write(t, up, tpl("specs/AGENTS.md"), "# Specs\n\nTree of `specs/business/` and `specs/user/`.\n")
	write(t, up, tpl("specs/user/AGENTS.md"), "# User\n\nSurface.\n")
	write(t, up, tpl("specs/operator/AGENTS.md"), "# Operator\n")
	write(t, up, tpl("OLD.md"), "gone later\n")
	write(t, up, tpl(".agents/skills/sdd-spec/SKILL.md"), "spec skill v1\n")
	write(t, up, tpl(".agents/skills/sdd-design/SKILL.md"), "design skill\n")
	write(t, up, tpl(".githooks/pre-commit"), "#!/bin/sh\nexit 0\n")
	os.Chmod(filepath.Join(up, tpl(".githooks/pre-commit")), 0o755)
	os.MkdirAll(filepath.Join(up, tpl(".claude/skills")), 0o755)
	os.Symlink("../../.agents/skills/sdd-spec", filepath.Join(up, tpl(".claude/skills/sdd-spec")))
	shell(t, up, "add", "-A")
	shell(t, up, "commit", "-q", "-m", "v1")
	v1 := shell(t, up, "rev-parse", "HEAD")
	write(t, up, tpl(".sdd/.sdd-config.json"), strings.ReplaceAll(tplConfig, "%d", "2"))
	write(t, up, tpl("AGENTS.md"), "# Law v2\n\nProduct specs (`specs/`) and code (`src/`).\n\nLine two.\n")
	write(t, up, tpl("specs/AGENTS.md"), "# Specs v2\n\nTree of `specs/business/` and `specs/user/`.\n")
	write(t, up, tpl("src/AGENTS.md"), "# Source\n\nCode under `src/`, specs under `src/specs/`.\n")
	write(t, up, tpl(".agents/skills/sdd-spec/SKILL.md"), "spec skill v2\n")
	os.Remove(filepath.Join(up, tpl("OLD.md")))
	shell(t, up, "add", "-A")
	shell(t, up, "commit", "-q", "-m", "v2")
	return up, v1
}

func TestBootstrapThenUpdateMergesThreeWays(t *testing.T) {
	up, v1 := upstream(t)
	proj := t.TempDir()
	write(t, proj, ".sdd/.sdd-config.json", `{"version": 0, "remote": "r", "paths": {"specs": "product"}, "perspectives": [{"name": "business", "from": "business"}, {"name": "member", "from": "user"}, {"name": "ops"}], "skills": ["sdd-spec"]}`+"\n")
	cfg, err := config.Load(proj)
	if err != nil {
		t.Fatal(err)
	}
	// Bootstrap against version 1: the working tree of a checkout at v1.
	shell(t, up, "checkout", "-q", v1)
	p, err := plan(proj, up, cfg)
	if err != nil {
		t.Fatal(err)
	}
	out := p.render()
	for _, want := range []string{"version 0 -> 1", "base none", "add      product/member/AGENTS.md (from specs/user/AGENTS.md)", "skip     specs/operator/AGENTS.md — no perspective from operator", "skip     .agents/skills/sdd-design/SKILL.md — skill not listed", "by hand: write product/ops/AGENTS.md"} {
		if !strings.Contains(out, want) {
			t.Fatalf("plan lacks %q:\n%s", want, out)
		}
	}
	if conflicts, err := p.apply(proj); err != nil || conflicts != 0 {
		t.Fatalf("apply: %d, %v", conflicts, err)
	}
	if got := read(t, proj, "product/AGENTS.md"); got != "# Specs\n\nTree of `product/business/` and `product/member/`.\n" {
		t.Fatalf("tokens: %q", got)
	}
	if st, err := os.Stat(filepath.Join(proj, ".githooks/pre-commit")); err != nil || st.Mode()&0o111 == 0 {
		t.Fatalf("hook mode: %v %v", st, err)
	}
	if target, err := os.Readlink(filepath.Join(proj, ".claude/skills/sdd-spec")); err != nil || target != "../../.agents/skills/sdd-spec" {
		t.Fatalf("symlink: %q %v", target, err)
	}
	if !strings.Contains(read(t, proj, ".sdd/.sdd-config.json"), `"version": 1,`) {
		t.Fatalf("version not recorded: %s", read(t, proj, ".sdd/.sdd-config.json"))
	}
	if _, err := os.Stat(filepath.Join(proj, ".agents/skills/sdd-design")); err == nil {
		t.Fatal("skipped skill was written")
	}
	// The project adapts: a line appended to the law, the member law rewritten.
	write(t, proj, "AGENTS.md", read(t, proj, "AGENTS.md")+"\nProject line.\n")
	write(t, proj, "product/member/AGENTS.md", "# Member\n\nWhat a member sees.\n")
	// Update to version 2.
	shell(t, up, "checkout", "-q", "main")
	cfg, _ = config.Load(proj)
	p, err = plan(proj, up, cfg)
	if err != nil {
		t.Fatal(err)
	}
	out = p.render()
	for _, want := range []string{"version 1 -> 2", "base " + v1[:12], "merge    AGENTS.md", "update   product/AGENTS.md", "add      apps/code/AGENTS.md"} {
		if !strings.Contains(out, want) && !strings.Contains(out, strings.Replace(want, "apps/code", "src", 1)) {
			t.Fatalf("plan lacks %q:\n%s", want, out)
		}
	}
	for _, want := range []string{"delete   OLD.md", "update   .agents/skills/sdd-spec/SKILL.md"} {
		if !strings.Contains(out, want) {
			t.Fatalf("plan lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "product/member/AGENTS.md") {
		t.Fatalf("unchanged upstream file planned:\n%s", out)
	}
	if conflicts, err := p.apply(proj); err != nil || conflicts != 0 {
		t.Fatalf("apply: %d, %v\n%s", conflicts, err, out)
	}
	if got := read(t, proj, "AGENTS.md"); got != "# Law v2\n\nProduct specs (`product/`) and code (`src/`).\n\nLine two.\n\nProject line.\n" {
		t.Fatalf("merge: %q", got)
	}
	if got := read(t, proj, "product/member/AGENTS.md"); got != "# Member\n\nWhat a member sees.\n" {
		t.Fatalf("adapted law lost: %q", got)
	}
	if _, err := os.Stat(filepath.Join(proj, "OLD.md")); err == nil {
		t.Fatal("OLD.md not deleted")
	}
	if got := read(t, proj, "src/AGENTS.md"); got != "# Source\n\nCode under `src/`, specs under `src/specs/`.\n" {
		t.Fatalf("added: %q", got)
	}
	if !strings.Contains(read(t, proj, ".sdd/.sdd-config.json"), `"version": 2,`) {
		t.Fatal("version 2 not recorded")
	}
	// A conflict: both sides change the same line.
	write(t, proj, "AGENTS.md", strings.Replace(read(t, proj, "AGENTS.md"), "Line two.", "Line two, ours.", 1))
	write(t, up, ".sdd/templates/AGENTS.md", strings.Replace(read(t, up, ".sdd/templates/AGENTS.md"), "Line two.", "Line two, theirs.", 1))
	write(t, up, ".sdd/templates/.sdd/.sdd-config.json", strings.ReplaceAll(tplConfig, "%d", "3"))
	shell(t, up, "add", "-A")
	shell(t, up, "commit", "-q", "-m", "v3")
	cfg, _ = config.Load(proj)
	p, err = plan(proj, up, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.render(), "conflict AGENTS.md") {
		t.Fatalf("no conflict:\n%s", p.render())
	}
	if conflicts, err := p.apply(proj); err != nil || conflicts != 1 {
		t.Fatalf("apply: %d, %v", conflicts, err)
	}
	if got := read(t, proj, "AGENTS.md"); !strings.Contains(got, "<<<<<<< project") || !strings.Contains(got, ">>>>>>> framework") {
		t.Fatalf("markers: %q", got)
	}
}

func TestConfigDecidesWhatComesBackAndWhatIsRefused(t *testing.T) {
	up, _ := upstream(t)
	proj := t.TempDir()
	// Perspectives left out: every shipped kind; a skill removed by hand comes back.
	write(t, proj, ".sdd/.sdd-config.json", `{"version": 0, "remote": "r"}`+"\n")
	cfg, _ := config.Load(proj)
	p, err := plan(proj, up, cfg)
	if err != nil {
		t.Fatal(err)
	}
	out := p.render()
	for _, want := range []string{"add      specs/user/AGENTS.md", "add      specs/operator/AGENTS.md", "add      .agents/skills/sdd-design/SKILL.md"} {
		if !strings.Contains(out, want) {
			t.Fatalf("plan lacks %q:\n%s", want, out)
		}
	}
	if _, err := p.apply(proj); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(filepath.Join(proj, ".agents/skills/sdd-design"))
	cfg, _ = config.Load(proj)
	p, _ = plan(proj, up, cfg)
	if out := p.render(); !strings.Contains(out, "add      .agents/skills/sdd-design/SKILL.md (from") && !strings.Contains(out, "add      .agents/skills/sdd-design/SKILL.md — the config keeps it") {
		t.Fatalf("a removed skill the config keeps did not come back:\n%s", out)
	}
	// A dropped skill is dropped in the config: its directory goes, and so does the empty parent.
	write(t, proj, ".sdd/.sdd-config.json", strings.Replace(read(t, proj, ".sdd/.sdd-config.json"), `"remote": "r"`, `"remote": "r", "skills": ["sdd-spec"]`, 1))
	cfg, _ = config.Load(proj)
	p, _ = plan(proj, up, cfg)
	if _, err := p.apply(proj); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(proj, ".agents/skills/sdd-design")); err == nil {
		t.Fatal("dropped skill directory kept")
	}
	// Unknown from and unknown skill are refused; an older framework too.
	for _, text := range []string{`{"version": 2, "remote": "r", "perspectives": [{"name": "x", "from": "customer"}]}`, `{"version": 2, "remote": "r", "skills": ["sdd-nope"]}`, `{"version": 9, "remote": "r"}`} {
		write(t, proj, ".sdd/.sdd-config.json", text+"\n")
		cfg, _ = config.Load(proj)
		if _, err := plan(proj, up, cfg); err == nil {
			t.Fatalf("%s: accepted", text)
		}
	}
	// A kind with two perspectives leaves its token and names the mapped project file.
	write(t, proj, ".sdd/.sdd-config.json", `{"version": 0, "remote": "r", "paths": {"specs": "product"}, "perspectives": [{"name": "player", "from": "user"}, {"name": "coach", "from": "user"}]}`+"\n")
	cfg, _ = config.Load(proj)
	p, _ = plan(proj, up, cfg)
	if out := p.render(); !strings.Contains(out, "by hand: review product/AGENTS.md: it names a perspective kind") {
		t.Fatalf("ambiguous kind not named:\n%s", out)
	}
}

func TestBaseFallsBackToTheNewestVersionBelow(t *testing.T) {
	up, v1 := upstream(t)
	proj := t.TempDir()
	write(t, proj, ".sdd/.sdd-config.json", `{"version": 1, "remote": "r"}`+"\n")
	cfg, _ := config.Load(proj)
	p := &planned{}
	_, sha, err := readBase(up, 1, p)
	if err != nil || sha != v1[:12] || len(p.byHand) != 0 {
		t.Fatalf("base = %s, %v, %v", sha, err, p.byHand)
	}
	cfg.Version = 5
	p = &planned{}
	if _, sha, err = readBase(up, 5, p); err != nil || !strings.HasPrefix(shell(t, up, "rev-parse", "HEAD"), sha) || len(p.byHand) != 1 {
		t.Fatalf("fallback = %s, %v, %v", sha, err, p.byHand)
	}
	if _, _, err = readBase(up, 0, p); err == nil {
		t.Fatal("version 0 found a base")
	}
}
