// update: what must change in this project to hold framework version M — the
// files .sdd/templates/ of the framework at M add, change, or remove against
// the version N the project holds, each mapped to where this project keeps
// it — and, with --apply, the change itself. A project at version 0 is being
// bootstrapped: every file is an add.
//
// Usage: go run -C .agents/tools ./update [--apply] [--from <dir>] [--root <dir>]
//
//	--apply       write: add and update files, merge three ways where both
//	              sides changed (conflict markers stay in the file), delete
//	              what the framework removed and the project never touched,
//	              then record M as the version of .sdd/.sdd-config.json
//	--from <dir>  a checkout of the framework, relative to the project root:
//	              its working tree is M and its history holds N (default: a
//	              clone of the remote of .sdd/.sdd-config.json, removed
//	              afterwards)
//	--root <dir>  project root, absolute (default: nearest directory at or
//	              above the working directory with .sdd/.sdd-config.json;
//	              inside a framework checkout, pass it)
//
// Mapping, from the project's config: specs/<kind>/… goes to
// <paths.specs>/<name>/… for every perspective whose from is <kind> (none:
// skipped); src/specs/… to <paths.technical>/…; src/… to <paths.src>/…; a
// skill not in skills is skipped; .sdd/.sdd-config.json is never copied. In
// every .md the tokens src/specs/, specs/, src/, and specs/<kind>/ become
// the project's. The base is the framework at the commit whose config holds
// version N (the newest at or below N when N is gone), mapped the same way,
// so a three-way merge keeps what the project adapted.
//
// Output, plain text: `version N -> M` with the source and the base, then
// one line per file — add, update, merge, conflict, delete, missing (the
// project removed it; left absent), same, skip — with the template path it
// comes from when that differs, then what the agent writes by hand. Exit 0;
// 1 when the framework or the base cannot be read or a conflict is left;
// 2 on a usage error.
//
// It implements **Materialize** of TEMP_BOOTSTRAP.md and the steps of
// .agents/skills/sdd-update-framework/SKILL.md; they win over it.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"tools/internal/config"
	"tools/internal/repo"
)

const (
	usage     = "usage: go run -C .agents/tools ./update [--apply] [--from <dir>] [--root <dir>]"
	templates = ".sdd/templates"
)

var versionRe = regexp.MustCompile(`"version"\s*:\s*(\d+)`)

func main() {
	apply, from, root := false, "", ""
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--apply":
			apply = true
		case args[i] == "--from" && i+1 < len(args):
			i++
			from = args[i]
		case args[i] == "--root" && i+1 < len(args):
			i++
			root = args[i]
		default:
			fail("unknown argument: "+args[i]+"\n"+usage, 2)
		}
	}
	root, err := repo.Root(root)
	if err != nil {
		fail(err.Error(), 1)
	}
	cfg, err := config.Load(root)
	if err != nil {
		fail(err.Error(), 1)
	}
	source := from
	if from == "" {
		if cfg.Remote == "" {
			fail("no --from and no remote in "+config.File, 1)
		}
		tmp, err := os.MkdirTemp("", "sdd-update-")
		if err != nil {
			fail(err.Error(), 1)
		}
		defer os.RemoveAll(tmp)
		if out, err := exec.Command("git", "clone", "--quiet", cfg.Remote, tmp).CombinedOutput(); err != nil {
			fail(fmt.Sprintf("git clone %s: %s", cfg.Remote, strings.TrimSpace(string(out))), 1)
		}
		from, source = tmp, cfg.Remote
	}
	if !filepath.IsAbs(from) {
		from = filepath.Join(root, from) // go run -C moves the working directory; the root is what the user sees
	}
	p, err := plan(root, from, cfg)
	if err != nil {
		fail(err.Error(), 1)
	}
	p.source = source
	fmt.Print(p.render())
	if !apply {
		return
	}
	conflicts, err := p.apply(root)
	if err != nil {
		fail(err.Error(), 1)
	}
	fmt.Printf("applied: version %d", p.to)
	if conflicts > 0 {
		fmt.Printf("; %d conflict(s) to resolve\n", conflicts)
		os.Exit(1)
	}
	fmt.Println()
}

// entry is one file of a tree: a regular file, an executable, or a symlink.
type entry struct {
	exec bool
	link string // target when a symlink
	data []byte
}

func (e entry) equal(o entry) bool {
	return e.exec == o.exec && e.link == o.link && bytes.Equal(e.data, o.data)
}

type tree map[string]entry

// action is one file of the plan.
type action struct {
	verb string // add, update, merge, conflict, delete, missing, same, skip
	path string // project-relative
	from string // template path, when it differs from path
	note string
	want *entry // what to write, when the verb writes
}

type planned struct {
	from, to  int
	source    string
	base      string // commit of the base, or "none"
	actions   []action
	byHand    []string
	templates tree
}

// plan reads the framework at from, the base at the project's version, and
// the project, and decides every file.
func plan(root, from string, cfg config.Config) (*planned, error) {
	theirs, tplCfg, err := readTemplates(from)
	if err != nil {
		return nil, err
	}
	p := &planned{from: cfg.Version, to: tplCfg.Version, base: "none", templates: theirs}
	base := tree{}
	if cfg.Version > 0 {
		var sha string
		base, sha, err = readBase(from, cfg.Version, p)
		if err != nil {
			return nil, err
		}
		p.base = sha
	}
	m := newMapper(cfg, tplCfg)
	ourSide := m.project(theirs)
	baseSide := m.project(base)
	paths := map[mapped]bool{}
	for k := range ourSide {
		paths[k] = true
	}
	for k := range baseSide {
		paths[k] = true
	}
	for _, mp := range sortedKeys(paths) {
		t, hasT := ourSide[mp]
		b, hasB := baseSide[mp]
		o, hasO, err := readOurs(root, mp.path)
		if err != nil {
			return nil, err
		}
		a := action{path: mp.path}
		if mp.from != mp.path {
			a.from = mp.from
		}
		switch {
		case hasT && !hasB:
			switch {
			case !hasO:
				a.verb, a.want = "add", &t
			case o.equal(t):
				a.verb = "same"
			default:
				a.verb, a.note = "conflict", "exists in the project with other content; kept"
			}
		case !hasT && hasB:
			switch {
			case !hasO:
				a.verb = "same"
			case o.equal(b):
				a.verb = "delete"
			default:
				a.verb, a.note = "conflict", "removed by the framework, changed by the project; kept"
			}
		default:
			switch {
			case !hasO:
				a.verb, a.note = "missing", "removed by the project; left absent"
			case o.equal(t):
				a.verb = "same"
			case o.equal(b):
				a.verb, a.want = "update", &t
			case b.equal(t):
				a.verb = "same"
			case t.link != "" || o.link != "" || b.link != "" || t.exec != o.exec:
				a.verb, a.note = "conflict", "changed on both sides; kept"
			default:
				merged, clean, err := merge3(o.data, b.data, t.data)
				if err != nil {
					return nil, err
				}
				e := entry{exec: t.exec, data: merged}
				if clean {
					a.verb, a.want = "merge", &e
				} else {
					a.verb, a.want, a.note = "conflict", &e, "changed on both sides; markers left"
				}
			}
		}
		p.actions = append(p.actions, a)
	}
	for _, s := range m.skipped {
		p.actions = append(p.actions, action{verb: "skip", path: s.path, note: s.note})
	}
	sort.SliceStable(p.actions, func(i, j int) bool { return p.actions[i].path < p.actions[j].path })
	for _, ps := range cfg.Perspectives {
		if ps.From != "" {
			continue
		}
		law := cfg.Paths.Specs + "/" + ps.Name + "/" + "AGENTS.md"
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(law))); err != nil {
			p.byHand = append(p.byHand, fmt.Sprintf("write %s: the law of perspective %s, by analogy to a shipped one", law, ps.Name))
		}
	}
	return p, nil
}

func (p *planned) render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "version %d -> %d (%s), base %s\n", p.from, p.to, p.source, p.base)
	counts := map[string]int{}
	for _, a := range p.actions {
		counts[a.verb]++
		if a.verb == "same" {
			continue
		}
		fmt.Fprintf(&b, "%-8s %s", a.verb, a.path)
		if a.from != "" {
			fmt.Fprintf(&b, " (from %s)", a.from)
		}
		if a.note != "" {
			fmt.Fprintf(&b, " — %s", a.note)
		}
		b.WriteString("\n")
	}
	var summary []string
	for _, v := range []string{"add", "update", "merge", "conflict", "delete", "missing", "same", "skip"} {
		if counts[v] > 0 {
			summary = append(summary, fmt.Sprintf("%d %s", counts[v], v))
		}
	}
	if len(summary) > 0 {
		b.WriteString(strings.Join(summary, ", ") + "\n")
	}
	for _, h := range p.byHand {
		b.WriteString("by hand: " + h + "\n")
	}
	return b.String()
}

// apply writes the plan and records the version; it returns the conflicts left.
func (p *planned) apply(root string) (int, error) {
	conflicts := 0
	for _, a := range p.actions {
		abs := filepath.Join(root, filepath.FromSlash(a.path))
		switch a.verb {
		case "add", "update", "merge", "conflict":
			if a.verb == "conflict" {
				conflicts++
			}
			if a.want == nil {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return conflicts, err
			}
			if a.want.link != "" {
				os.Remove(abs)
				if err := os.Symlink(filepath.FromSlash(a.want.link), abs); err != nil {
					return conflicts, err
				}
				continue
			}
			mode := os.FileMode(0o644)
			if a.want.exec {
				mode = 0o755
			}
			os.Remove(abs) // a symlink or a stale mode in the way
			if err := os.WriteFile(abs, a.want.data, mode); err != nil {
				return conflicts, err
			}
		case "delete":
			if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
				return conflicts, err
			}
		}
	}
	file := filepath.Join(root, filepath.FromSlash(config.File))
	raw, err := os.ReadFile(file)
	if err != nil {
		return conflicts, err
	}
	m := versionRe.FindSubmatchIndex(raw)
	if m == nil {
		return conflicts, errors.New(config.File + ": no \"version\" field to record")
	}
	out := append([]byte(nil), raw[:m[2]]...)
	out = append(out, strconv.Itoa(p.to)...)
	out = append(out, raw[m[3]:]...)
	return conflicts, os.WriteFile(file, out, 0o644)
}

// mapper turns template paths and tokens into the project's.
type mapper struct {
	cfg, tpl config.Config
	kinds    map[string][]string // template kind → project perspective names
	tokens   [][2]string         // old → new, longest first; an identity pair protects its token
	active   bool                // some token changes
	skipped  []skipped
}

type skipped struct{ path, note string }

// mapped is a project path with the template path it comes from.
type mapped struct{ path, from string }

func newMapper(cfg, tpl config.Config) *mapper {
	m := &mapper{cfg: cfg, tpl: tpl, kinds: map[string][]string{}}
	for _, k := range tpl.Perspectives {
		m.kinds[k.Name] = nil
	}
	for _, ps := range cfg.Perspectives {
		if ps.From != "" {
			m.kinds[ps.From] = append(m.kinds[ps.From], ps.Name)
		}
	}
	add := func(old, new string) {
		m.tokens = append(m.tokens, [2]string{old, new})
		if old != new {
			m.active = true
		}
	}
	for kind, names := range m.kinds {
		if len(names) == 1 {
			add(tpl.Paths.Specs+"/"+kind+"/", cfg.Paths.Specs+"/"+names[0]+"/")
		}
	}
	add(tpl.Paths.Technical+"/", cfg.Paths.Technical+"/")
	add(tpl.Paths.Specs+"/", cfg.Paths.Specs+"/")
	add(tpl.Paths.Src+"/", cfg.Paths.Src+"/")
	sort.SliceStable(m.tokens, func(i, j int) bool { return len(m.tokens[i][0]) > len(m.tokens[j][0]) })
	return m
}

// project maps a template tree to project paths, mapping tokens in .md files.
func (m *mapper) project(t tree) map[mapped]entry {
	out := map[mapped]entry{}
	seen := map[string]bool{}
	for _, p := range sortedTree(t) {
		targets, note := m.paths(p)
		if note != "" {
			if !seen[p] {
				seen[p] = true
				m.skipped = append(m.skipped, skipped{p, note})
			}
			continue
		}
		e := t[p]
		if strings.HasSuffix(p, ".md") && e.link == "" {
			e.data = m.replace(e.data)
		}
		for _, target := range targets {
			out[mapped{target, p}] = e
		}
	}
	return out
}

// paths returns where template path p goes in the project, or why it is skipped.
func (m *mapper) paths(p string) ([]string, string) {
	if p == config.File {
		return nil, "the project's own"
	}
	if s := skillOf(p); s != "" && len(m.cfg.Skills) > 0 && !contains(m.cfg.Skills, s) {
		return nil, "skill not listed in " + config.File
	}
	tp, cp := m.tpl.Paths, m.cfg.Paths
	switch {
	case config.Under(p, tp.Technical):
		return []string{cp.Technical + strings.TrimPrefix(p, tp.Technical)}, ""
	case config.Under(p, tp.Specs):
		rest := strings.TrimPrefix(p, tp.Specs+"/")
		kind, tail, _ := strings.Cut(rest, "/")
		if names, isKind := m.kinds[kind]; isKind && tail != "" {
			if len(names) == 0 {
				return nil, "no perspective from " + kind
			}
			var out []string
			for _, n := range names {
				out = append(out, cp.Specs+"/"+n+"/"+tail)
			}
			return out, ""
		}
		return []string{cp.Specs + "/" + rest}, ""
	case config.Under(p, tp.Src):
		return []string{cp.Src + strings.TrimPrefix(p, tp.Src)}, ""
	}
	return []string{p}, ""
}

// replace rewrites every token in one pass, longest token first at each
// position, so a replacement is never replaced again and a longer root
// (src/specs/) shields the shorter one it contains (specs/).
func (m *mapper) replace(text []byte) []byte {
	if !m.active {
		return text
	}
	var out []byte
	for i := 0; i < len(text); {
		hit := false
		for _, t := range m.tokens {
			if bytes.HasPrefix(text[i:], []byte(t[0])) {
				out = append(out, t[1]...)
				i += len(t[0])
				hit = true
				break
			}
		}
		if !hit {
			out = append(out, text[i])
			i++
		}
	}
	return out
}

// skillOf names the skill a template path belongs to, or "".
func skillOf(p string) string {
	for _, dir := range []string{".agents/skills/", ".claude/skills/"} {
		if strings.HasPrefix(p, dir) {
			s, _, _ := strings.Cut(strings.TrimPrefix(p, dir), "/")
			return s
		}
	}
	return ""
}

// readTemplates reads .sdd/templates/ of a framework checkout as git sees
// its working tree, and the framework's config.
func readTemplates(from string) (tree, config.Config, error) {
	dir := filepath.Join(from, filepath.FromSlash(templates))
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil, config.Config{}, errors.New("no " + templates + " under " + from)
	}
	out, err := git(from, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", templates)
	if err != nil {
		return nil, config.Config{}, err
	}
	t := tree{}
	for _, f := range strings.Split(out, "\x00") {
		if f == "" {
			continue
		}
		rel := strings.TrimPrefix(f, templates+"/")
		abs := filepath.Join(from, filepath.FromSlash(f))
		st, err := os.Lstat(abs)
		if err != nil {
			continue
		}
		var e entry
		switch {
		case st.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(abs)
			if err != nil {
				return nil, config.Config{}, err
			}
			e.link = filepath.ToSlash(target)
		case st.Mode().IsRegular():
			if e.data, err = os.ReadFile(abs); err != nil {
				return nil, config.Config{}, err
			}
			e.exec = st.Mode()&0o111 != 0
		default:
			continue
		}
		t[rel] = e
	}
	raw, ok := t[config.File]
	if !ok {
		return nil, config.Config{}, errors.New("no " + templates + "/" + config.File + " under " + from)
	}
	cfg, err := config.Parse(raw.data)
	if err != nil {
		return nil, config.Config{}, fmt.Errorf("%s/%v", templates, err)
	}
	return t, cfg, nil
}

// readBase reads .sdd/templates/ at the newest commit of the framework whose
// config holds version n, or the newest below it, warning in the plan.
func readBase(from string, n int, p *planned) (tree, string, error) {
	list, err := git(from, "rev-list", "HEAD", "--", templates+"/"+config.File)
	if err != nil {
		return nil, "", err
	}
	var sha string
	var v int
	for _, c := range strings.Fields(list) {
		raw, err := git(from, "show", c+":"+templates+"/"+config.File)
		if err != nil {
			continue
		}
		m := versionRe.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		if cv, _ := strconv.Atoi(m[1]); cv <= n {
			sha, v = c, cv
			break
		}
	}
	if sha == "" {
		return nil, "", fmt.Errorf("no commit of the framework holds version %d or below; the project's version is not in this history", n)
	}
	if v != n {
		p.byHand = append(p.byHand, fmt.Sprintf("version %d is not in the framework history; the base is version %d (%s)", n, v, sha[:12]))
	}
	out, err := git(from, "ls-tree", "-r", "-z", sha, "--", templates)
	if err != nil {
		return nil, "", err
	}
	t := tree{}
	for _, line := range strings.Split(out, "\x00") {
		if line == "" {
			continue
		}
		meta, f, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		fields := strings.Fields(meta)
		if len(fields) != 3 || fields[1] != "blob" {
			continue
		}
		content, err := git(from, "cat-file", "blob", fields[2])
		if err != nil {
			return nil, "", err
		}
		e := entry{exec: fields[0] == "100755"}
		if fields[0] == "120000" {
			e.link = content
		} else {
			e.data = []byte(content)
		}
		t[strings.TrimPrefix(f, templates+"/")] = e
	}
	return t, sha[:12], nil
}

// readOurs reads one project file, a symlink included.
func readOurs(root, rel string) (entry, bool, error) {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	st, err := os.Lstat(abs)
	if os.IsNotExist(err) {
		return entry{}, false, nil
	}
	if err != nil {
		return entry{}, false, err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(abs)
		return entry{link: filepath.ToSlash(target)}, true, err
	}
	if !st.Mode().IsRegular() {
		return entry{}, false, fmt.Errorf("%s is neither a file nor a symlink", rel)
	}
	data, err := os.ReadFile(abs)
	return entry{exec: st.Mode()&0o111 != 0, data: data}, true, err
}

// merge3 merges theirs onto ours over base with git merge-file; clean is
// false when markers are left.
func merge3(ours, base, theirs []byte) ([]byte, bool, error) {
	dir, err := os.MkdirTemp("", "sdd-merge-")
	if err != nil {
		return nil, false, err
	}
	defer os.RemoveAll(dir)
	names := []string{"ours", "base", "theirs"}
	for i, data := range [][]byte{ours, base, theirs} {
		if err := os.WriteFile(filepath.Join(dir, names[i]), data, 0o644); err != nil {
			return nil, false, err
		}
	}
	cmd := exec.Command("git", "merge-file", "-p", "-L", "project", "-L", "base", "-L", "framework", filepath.Join(dir, "ours"), filepath.Join(dir, "base"), filepath.Join(dir, "theirs"))
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err = cmd.Run()
	if err == nil {
		return out.Bytes(), true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() > 0 && exit.ExitCode() < 128 {
		return out.Bytes(), false, nil
	}
	return nil, false, fmt.Errorf("git merge-file: %v %s", err, strings.TrimSpace(errb.String()))
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s in %s: %s", args[0], dir, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

func sortedTree(t tree) []string {
	out := make([]string, 0, len(t))
	for k := range t {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeys(m map[mapped]bool) []mapped {
	out := make([]mapped, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].path != out[j].path {
			return out[i].path < out[j].path
		}
		return out[i].from < out[j].from
	})
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func fail(msg string, code int) {
	fmt.Fprintf(os.Stderr, "update: %s\n", msg)
	os.Exit(code)
}
