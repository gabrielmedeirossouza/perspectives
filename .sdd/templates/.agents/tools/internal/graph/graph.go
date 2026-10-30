// Package graph loads every node under the specs root and the code root of
// .sdd/.sdd-config.json, computes the current hash and rules of each, resolves
// the sources they list, reads the gaps parked in TODO.md of the specs root,
// and derives the states of **Metadata** in the root AGENTS.md: drift, stale,
// dangling, unconsumed, fresh, waits — and the audit standing of each code
// node (audit.go).
package graph

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tools/internal/config"
	"tools/internal/md"
	"tools/internal/meta"
)

// Kind is the kind of a node, given by its place in the tree.
type Kind int

const (
	Perspective Kind = iota // under the specs root: a product spec
	Technical               // under the technical root: a technical spec
	Code                    // elsewhere under the code root: code
)

func (k Kind) String() string {
	switch k {
	case Perspective:
		return "perspective"
	case Technical:
		return "technical"
	}
	return "code"
}

// MaySource reports whether a node of kind k may list a node of kind to.
func (k Kind) MaySource(to Kind) bool {
	switch k {
	case Technical, Code:
		return to == Perspective || to == Technical
	}
	return false
}

func allowed(k Kind) string {
	switch k {
	case Perspective:
		return "a perspective node lists no source"
	case Technical:
		return "a technical node sources perspective and technical nodes"
	}
	return "a code node sources technical nodes and the perspective nodes they serve"
}

// Node is a file with <file>.metadata.json beside it — a spec.md under the
// specs root is one even before the tool creates its metadata — or a
// directory under the code root whose .metadata.json links the files of the
// directory that are not code (a directory node).
type Node struct {
	Rel      string // root-relative path: the file, or the directory of a directory node
	Dir      string // absolute directory holding the node
	MetaRel  string // root-relative metadata file
	MetaPath string // absolute metadata file
	Kind     Kind
	IsDir    bool // a directory node
	HasFile  bool // a file node whose file exists; false on orphan metadata
	HasMeta  bool
	Meta     *meta.Metadata // nil when absent or unreadable
	MetaErr  error
	Hash     string            // current node hash
	Rules    map[string]string // current rule hashes; nil without a spec
	Order    []string          // rule titles in file order
	RuleErr  error
	Sources  []*Edge // one per Meta.Sources entry, in order
	content  []string
	linkErrs []finding // what is wrong with files of a directory node
	refErrs  []string  // links of a spec that name no file or no heading
	text     string    // the spec file, read for its rules and links
}

// Files lists the node's content, names relative to its directory, sorted:
// the file itself, or the files a directory node links.
func (n *Node) Files() []string { return append([]string(nil), n.content...) }

// Label names the node inside its directory: the file name, or the
// metadata file of a directory node.
func (n *Node) Label() string {
	if n.IsDir {
		return meta.FileName
	}
	return path.Base(n.Rel)
}

// UnitDir is the directory the node belongs to for /sdd-implement units and
// /sdd-audit judges: its own directory for a directory node.
func (n *Node) UnitDir() string {
	if n.IsDir {
		return n.Rel
	}
	return path.Dir(n.Rel)
}

// spec reports whether the node is a spec node: a perspective or technical
// node, which holds rules.
func (n *Node) spec() bool { return n.Kind != Code }

// Edge is one source a node lists.
type Edge struct {
	From   *Node
	Source meta.Source
	To     *Node // nil when the path is no node
	Delta  meta.Delta
}

// Dangling reports whether the listed path is no node.
func (e *Edge) Dangling() bool { return e.To == nil }

// Valid reports whether the target exists, differs from the node, and is a
// kind the node may source.
func (e *Edge) Valid() bool {
	return e.To != nil && e.To != e.From && e.From.Kind.MaySource(e.To.Kind)
}

// Stale reports whether the source's current hash differs from the one
// recorded at the last sync.
func (e *Edge) Stale() bool { return e.Valid() && e.Source.Hash != e.To.Hash }

// recorded lists the rule titles recorded for the source at the last sync, sorted.
func (e *Edge) recorded() []string {
	titles := make([]string, 0, len(e.Source.Rules))
	for t := range e.Source.Rules {
		titles = append(titles, t)
	}
	sort.Strings(titles)
	return titles
}

// Graph is every node of the repository with its edges.
type Graph struct {
	Root      string
	Paths     config.Paths // where product specs, technical specs, and code live
	Nodes     []*Node      // sorted by Rel
	byRel     map[string]*Node
	files     map[string]bool // every file git tracks or would track under the roots
	topo      []*Node         // technical nodes with metadata, sources first
	pos       map[*Node]int
	cycle     []string
	strays    []loose // metadata files that are no node, with the fix
	uncovered []loose // files in no node, with the fix
	gaps      []Gap
	gapDrift  []string
}

// Load reads the layout of .sdd/.sdd-config.json, lists the files git tracks
// or would track under its roots, finds the nodes among them, computes every
// hash and edge, and reads the gaps parked in TODO.md of the specs root.
func Load(root string) (*Graph, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	files, err := listFiles(root, cfg.Paths)
	if err != nil {
		return nil, err
	}
	g := &Graph{Root: root, Paths: cfg.Paths, byRel: map[string]*Node{}, files: map[string]bool{}, pos: map[*Node]int{}}
	dirs := map[string]bool{}
	for _, f := range files {
		g.files[f] = true
		for d := path.Dir(f); d != "." && d != "" && !dirs[d]; d = path.Dir(d) {
			dirs[d] = true
		}
	}
	node := func(rel string, dir bool) *Node {
		if n := g.byRel[rel]; n != nil {
			return n
		}
		k, ok := g.kindOf(rel)
		if !ok {
			return nil
		}
		n := &Node{Rel: rel, Kind: k, IsDir: dir, MetaRel: meta.MetaRel(rel, dir)}
		if dir {
			n.Dir = filepath.Join(root, filepath.FromSlash(rel))
		} else {
			n.Dir = filepath.Join(root, filepath.FromSlash(path.Dir(rel)))
		}
		n.MetaPath = filepath.Join(root, filepath.FromSlash(n.MetaRel))
		g.byRel[rel] = n
		g.Nodes = append(g.Nodes, n)
		return n
	}
	for _, f := range files {
		dir, base := path.Dir(f), path.Base(f)
		switch {
		case base == meta.FileName:
			if k, ok := g.kindOf(dir); ok && k == Code {
				node(dir, true).HasMeta = true
			} else {
				g.strays = append(g.strays, loose{f, fmt.Sprintf("`%s` is no node: the node of a spec is %s.%s", f, meta.SpecFile, strings.TrimPrefix(meta.Suffix, "."))})
			}
		case strings.HasSuffix(base, meta.Suffix):
			rel := strings.TrimSuffix(f, meta.Suffix)
			switch {
			case strings.HasSuffix(rel, meta.Suffix):
				g.strays = append(g.strays, loose{f, fmt.Sprintf("`%s` is no node: a metadata file has no metadata; delete it", f)})
			case g.isChrome(rel):
				g.strays = append(g.strays, loose{f, fmt.Sprintf("`%s` is no node: `%s` is chrome; delete it", f, rel)})
			case dirs[rel]:
				g.strays = append(g.strays, loose{f, fmt.Sprintf("`%s` is no node: `%s` is a directory, whose metadata is `%s/%s`; delete it", f, rel, rel, meta.FileName)})
			default:
				if n := node(rel, false); n != nil {
					n.HasMeta = true
				}
			}
		case base == meta.SpecFile && !g.isChrome(f) && (config.Under(dir, g.Paths.Specs) || config.Under(dir, g.Paths.Technical)):
			node(f, false)
		}
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].Rel < g.Nodes[j].Rel })
	for _, n := range g.Nodes {
		if n.HasMeta {
			n.Meta, n.MetaErr = meta.Read(n.MetaPath)
		}
	}
	for _, n := range g.Nodes {
		if n.IsDir {
			g.loadLinks(n)
		}
	}
	for _, f := range files {
		base := path.Base(f)
		if base == meta.FileName || strings.HasSuffix(base, meta.Suffix) || g.isChrome(f) {
			continue
		}
		if n := g.byRel[f]; n != nil {
			n.HasFile = true
			n.content = []string{base}
			continue
		}
		if d := g.byRel[path.Dir(f)]; d != nil && d.IsDir && d.links(base) {
			continue // content of the directory node, taken from files
		}
		g.uncovered = append(g.uncovered, loose{f, g.uncoveredFix(f)})
	}
	for _, n := range g.Nodes {
		if n.Hash, err = meta.HashFiles(n.Dir, n.content); err != nil {
			return nil, err
		}
		if n.spec() && n.HasFile && path.Base(n.Rel) == meta.SpecFile {
			text, err := os.ReadFile(filepath.Join(n.Dir, meta.SpecFile))
			if err != nil {
				return nil, err
			}
			n.text = string(text)
			n.Rules, n.Order, n.RuleErr = meta.RuleHashes(n.text)
		}
	}
	anchors, listings := map[string]map[string]bool{}, map[string]map[string]bool{}
	for _, n := range g.Nodes {
		if n.text != "" {
			n.refErrs = g.brokenLinks(n, anchors, listings)
			n.text = ""
		}
	}
	for _, n := range g.Nodes {
		if n.Meta == nil {
			continue
		}
		for _, s := range n.Meta.Sources {
			e := &Edge{From: n, Source: s, To: g.byRel[cleanRel(s.Path)]}
			if e.Valid() {
				e.Delta = meta.Diff(s.Rules, e.To.Rules, e.To.Order)
			}
			n.Sources = append(n.Sources, e)
		}
	}
	g.order()
	g.gaps, g.gapDrift = loadGaps(root, g.Paths)
	return g, nil
}

// brokenLinks lists the links of a spec that name nothing in the repository,
// or a heading the markdown file they name does not hold, its slug computed
// as Anchors of specs/AGENTS.md says. A path is resolved against the spec's
// directory, or against the repository root when it starts with `/`; a query
// is ignored; every segment must match a name on disk exactly, case included,
// as on a case-sensitive host. A link with a scheme is not followed. anchors
// caches the slugs of each file read, listings the names of each directory.
func (g *Graph) brokenLinks(n *Node, anchors, listings map[string]map[string]bool) []string {
	owner := "/sdd-spec"
	if n.Kind == Technical {
		owner = "/sdd-design"
	}
	var out []string
	for _, link := range md.Links(n.text) {
		if schemeRe.MatchString(link) || strings.HasPrefix(link, "//") {
			continue
		}
		target, frag, _ := strings.Cut(link, "#")
		target, _, _ = strings.Cut(target, "?")
		if t, err := url.PathUnescape(target); err == nil {
			target = t
		}
		if f, err := url.PathUnescape(frag); err == nil {
			frag = f
		}
		rel := n.Rel
		switch {
		case strings.HasPrefix(target, "/"):
			rel = path.Clean(strings.TrimLeft(target, "/"))
		case target != "":
			rel = path.Clean(path.Join(path.Dir(n.Rel), target))
		}
		if rel == ".." || strings.HasPrefix(rel, "../") {
			out = append(out, fmt.Sprintf("`%s` links `%s`, which leaves the repository; fix the link (%s)", n.Rel, link, owner))
			continue
		}
		if !g.exists(rel, listings) {
			out = append(out, fmt.Sprintf("`%s` links `%s`, but `%s` does not exist; fix the link (%s)", n.Rel, link, rel, owner))
			continue
		}
		if frag == "" || !strings.HasSuffix(rel, ".md") {
			continue
		}
		slugs, ok := anchors[rel]
		if !ok {
			slugs = map[string]bool{}
			if text, err := os.ReadFile(filepath.Join(g.Root, filepath.FromSlash(rel))); err == nil {
				for _, s := range md.Anchors(string(text)) {
					slugs[s] = true
				}
			}
			anchors[rel] = slugs
		}
		if !slugs[strings.ToLower(frag)] {
			out = append(out, fmt.Sprintf("`%s` links `%s`, but `%s` has no heading `#%s`: a removed or renamed title; fix the link (%s)", n.Rel, link, rel, frag, owner))
		}
	}
	return out
}

// exists reports whether a root-relative path names a file or directory whose
// every segment matches a name on disk exactly.
func (g *Graph) exists(rel string, listings map[string]map[string]bool) bool {
	if rel == "." {
		return true
	}
	dir := "."
	for _, seg := range strings.Split(rel, "/") {
		names, ok := listings[dir]
		if !ok {
			names = map[string]bool{}
			entries, _ := os.ReadDir(filepath.Join(g.Root, filepath.FromSlash(dir)))
			for _, e := range entries {
				names[e.Name()] = true
			}
			listings[dir] = names
		}
		if !names[seg] {
			return false
		}
		dir = path.Join(dir, seg)
	}
	return true
}

var schemeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// loadLinks reads files of a directory node: each key is the name of a
// tracked or trackable file directly in the directory that is neither chrome,
// a metadata file, nor a node of its own; each value lists sources the node
// lists. Valid keys become the node's content; the rest is drift.
func (g *Graph) loadLinks(n *Node) {
	if n.Meta == nil {
		return
	}
	block := func(msg string) { n.linkErrs = append(n.linkErrs, finding{msg, true}) }
	warn := func(msg string) { n.linkErrs = append(n.linkErrs, finding{msg, false}) }
	if len(n.Meta.Files) == 0 {
		warn(fmt.Sprintf("`%s` links no file; a directory node links at least one", n.MetaRel))
		return
	}
	sources := map[string]bool{}
	for _, s := range n.Meta.Sources {
		sources[cleanRel(s.Path)] = true
	}
	keys := make([]string, 0, len(n.Meta.Files))
	for k := range n.Meta.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f := n.Rel + "/" + k
		switch {
		case k == "" || k == "." || k == ".." || strings.Contains(k, "/") || strings.Contains(k, "\\"):
			block(fmt.Sprintf("`%s` links `%s`, which is no file name; files names the files directly in the directory", n.MetaRel, k))
			continue
		case k == meta.FileName || strings.HasSuffix(k, meta.Suffix):
			block(fmt.Sprintf("`%s` links `%s`, a metadata file", n.MetaRel, k))
			continue
		case g.isChrome(f):
			block(fmt.Sprintf("`%s` links `%s`, which is chrome", n.MetaRel, k))
			continue
		case !g.files[f]:
			block(fmt.Sprintf("`%s` links `%s`, which is no tracked or trackable file of the directory", n.MetaRel, k))
			continue
		case g.byRel[f] != nil:
			block(fmt.Sprintf("`%s` links `%s`, which has its own metadata; drop it from files", n.MetaRel, k))
			continue
		}
		n.content = append(n.content, k)
		for _, p := range n.Meta.Files[k] {
			if !sources[cleanRel(p)] {
				block(fmt.Sprintf("`%s` links `%s` to `%s`, which it does not list as a source", n.MetaRel, k, p))
			}
		}
	}
}

// links reports whether directory node n links a file of its directory by name.
func (n *Node) links(name string) bool {
	for _, c := range n.content {
		if c == name {
			return true
		}
	}
	return false
}

// uncoveredFix names a file in no node and how to cover it.
func (g *Graph) uncoveredFix(f string) string {
	if config.Under(f, g.Paths.Specs) || config.Under(f, g.Paths.Technical) {
		return fmt.Sprintf("`%s` is in no node: a spec directory holds %s and %s.%s", f, meta.SpecFile, meta.SpecFile, strings.TrimPrefix(meta.Suffix, "."))
	}
	return fmt.Sprintf("`%s` is in no node: give it %s%s, or link it in %s/%s", f, path.Base(f), meta.Suffix, path.Dir(f), meta.FileName)
}

// isChrome reports whether a file is repository chrome — no node's content,
// never drift: AGENTS.md anywhere, the files directly under the specs root,
// and the .md files directly under the code root.
func (g *Graph) isChrome(f string) bool {
	dir := path.Dir(f)
	return path.Base(f) == meta.LawFile || dir == g.Paths.Specs || (dir == g.Paths.Src && strings.HasSuffix(f, ".md"))
}

// listFiles returns the root-relative regular files git tracks or would track
// (not ignored) under the specs root, the code root, and the technical root
// when it sits beside the code root, sorted.
func listFiles(root string, p config.Paths) ([]string, error) {
	args := []string{"-C", root, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", p.Specs, p.Src}
	if !config.Under(p.Technical, p.Src) {
		args = append(args, p.Technical)
	}
	cmd := exec.Command("git", args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git ls-files in %s: %s", root, strings.TrimSpace(errb.String()))
	}
	seen := map[string]bool{}
	var files []string
	for _, f := range strings.Split(out.String(), "\x00") {
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		st, err := os.Stat(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil || !st.Mode().IsRegular() {
			continue
		}
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

// Staged returns the nodes holding a file staged for commit, sorted by path,
// and the staged files that belong to no node: a file added under a root
// without metadata, or a file whose metadata is staged for deletion. Renames
// are listed as a deletion and an addition; the deleted file's node is an
// orphan until its metadata moves with it.
func (g *Graph) Staged() (nodes []*Node, uncovered []string, err error) {
	cmd := exec.Command("git", "-C", g.Root, "diff", "--cached", "--name-only", "--no-renames", "-z", "--relative")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("git diff --cached: %s", strings.TrimSpace(errb.String()))
	}
	seen := map[*Node]bool{}
	for _, f := range strings.Split(out.String(), "\x00") {
		if f == "" {
			continue
		}
		if n := g.Owner(f); n != nil {
			if !seen[n] {
				seen[n] = true
				nodes = append(nodes, n)
			}
			continue
		}
		switch {
		case g.files[f] && !g.isChrome(f) && !strings.HasSuffix(f, meta.Suffix):
			if _, in := g.kindOf(f); in {
				uncovered = append(uncovered, g.uncoveredFix(f))
			}
		case strings.HasSuffix(f, meta.Suffix) && path.Base(f) != meta.FileName && g.files[strings.TrimSuffix(f, meta.Suffix)]:
			uncovered = append(uncovered, g.uncoveredFix(strings.TrimSuffix(f, meta.Suffix)))
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Rel < nodes[j].Rel })
	sort.Strings(uncovered)
	return nodes, uncovered, nil
}

// kindOf is the kind of a root-relative path by its place in the layout.
func (g *Graph) kindOf(rel string) (Kind, bool) {
	switch {
	case config.Under(rel, g.Paths.Specs):
		return Perspective, true
	case config.Under(rel, g.Paths.Technical):
		return Technical, true
	case config.Under(rel, g.Paths.Src):
		return Code, true
	}
	return 0, false
}

func cleanRel(p string) string {
	return strings.TrimPrefix(path.Clean(strings.ReplaceAll(p, "\\", "/")), "./")
}

// hint names the spec file a source should have named, when the listed path
// is the directory of a spec node.
func (g *Graph) hint(p string) string {
	if n := g.byRel[cleanRel(p)+"/"+meta.SpecFile]; n != nil {
		return fmt.Sprintf(" (a source names a spec file: `%s`)", n.Rel)
	}
	return ""
}

// usable reports whether a technical node can be ordered and realized: it
// has readable metadata, is a spec.md that exists, and its rules parse. The
// rest is drift owed to whoever wrote it, never an /sdd-implement unit.
func (n *Node) usable() bool {
	return n.Kind == Technical && n.HasMeta && n.MetaErr == nil && n.RuleErr == nil && malformed(n) == ""
}

// order sorts the usable technical nodes sources first (ties by path) and
// records the members of a cycle, which come last. A technical spec without
// metadata is no one's source yet and joins once its skeleton exists.
func (g *Graph) order() {
	var ts []*Node
	for _, n := range g.Nodes {
		if n.usable() {
			ts = append(ts, n)
		}
	}
	indeg := map[*Node]int{}
	next := map[*Node][]*Node{}
	for _, n := range ts {
		for _, e := range n.Sources {
			if e.Valid() && e.To.usable() {
				indeg[n]++
				next[e.To] = append(next[e.To], n)
			}
		}
	}
	var ready []*Node
	for _, n := range ts {
		if indeg[n] == 0 {
			ready = append(ready, n)
		}
	}
	for len(ready) > 0 {
		sort.Slice(ready, func(i, j int) bool { return ready[i].Rel < ready[j].Rel })
		n := ready[0]
		ready = ready[1:]
		g.pos[n] = len(g.topo)
		g.topo = append(g.topo, n)
		for _, m := range next[n] {
			if indeg[m]--; indeg[m] == 0 {
				ready = append(ready, m)
			}
		}
	}
	for _, n := range ts {
		if _, done := g.pos[n]; !done {
			g.cycle = append(g.cycle, n.Rel)
			g.pos[n] = len(g.topo)
			g.topo = append(g.topo, n)
		}
	}
}

// Node returns the node at a root-relative path, or nil.
func (g *Graph) Node(rel string) *Node { return g.byRel[rel] }

// loose is a file under the roots that belongs to no node, with its fix.
type loose struct{ file, msg string }

// Loose lists the metadata files that are no node, then the files in no
// node, each with its fix: every one when under is empty, else those at or
// under one of the root-relative paths of under.
func (g *Graph) Loose(under []string) []string {
	var out []string
	for _, l := range append(append([]loose(nil), g.strays...), g.uncovered...) {
		if len(under) == 0 {
			out = append(out, l.msg)
			continue
		}
		for _, u := range under {
			if l.file == u || strings.HasPrefix(l.file, u+"/") {
				out = append(out, l.msg)
				break
			}
		}
	}
	return out
}

// NodesUnder lists the nodes at or under a root-relative path, by path: the
// directory node of that directory, when it has one, and every node below.
func (g *Graph) NodesUnder(rel string) []*Node {
	var out []*Node
	for _, n := range g.Nodes {
		if n.Rel == rel || strings.HasPrefix(n.Rel, rel+"/") {
			out = append(out, n)
		}
	}
	return out
}

// Owner returns the node a root-relative file belongs to: the file's own
// node, the node a metadata file belongs to, or the directory node that
// links the file; nil when none.
func (g *Graph) Owner(f string) *Node {
	dir, base := path.Dir(f), path.Base(f)
	switch {
	case base == meta.FileName:
		if n := g.byRel[dir]; n != nil && n.IsDir {
			return n
		}
		return nil
	case strings.HasSuffix(base, meta.Suffix):
		if n := g.byRel[strings.TrimSuffix(f, meta.Suffix)]; n != nil && !n.IsDir {
			return n
		}
		return nil
	}
	if n := g.byRel[f]; n != nil && !n.IsDir {
		return n
	}
	if d := g.byRel[dir]; d != nil && d.IsDir && d.links(base) {
		return d
	}
	return nil
}

// consumers returns the nodes of kind k with a valid edge to n.
func (g *Graph) consumers(n *Node, k Kind) []*Node {
	var out []*Node
	for _, m := range g.Nodes {
		if m.Kind != k {
			continue
		}
		for _, e := range m.Sources {
			if e.Valid() && e.To == n {
				out = append(out, m)
				break
			}
		}
	}
	return out
}

// served maps every perspective node the code node's technical specs list to
// the first technical spec that serves it, and reports which nodes it lists.
func (g *Graph) served(c *Node) (served map[*Node]*Node, listed map[*Node]bool) {
	served, listed = map[*Node]*Node{}, map[*Node]bool{}
	for _, e := range c.Sources {
		if !e.Valid() {
			continue
		}
		listed[e.To] = true
		if e.To.Kind != Technical {
			continue
		}
		for _, te := range e.To.Sources {
			if te.Valid() && te.To.Kind == Perspective {
				if _, ok := served[te.To]; !ok {
					served[te.To] = e.To
				}
			}
		}
	}
	return served, listed
}

// missing lists the perspective nodes a code node's technical specs serve
// that the node does not list, sorted by path.
func (g *Graph) missing(c *Node) []*Node {
	served, listed := g.served(c)
	var out []*Node
	for _, p := range sortedNodes(served) {
		if !listed[p] {
			out = append(out, p)
		}
	}
	return out
}

// finding is one drift line; blocking when a judge cannot read the node
// until it is fixed (its content or metadata is not what it claims).
type finding struct {
	msg      string
	blocking bool
}

// NodeDrift lists what is wrong with n's own metadata: missing, orphan,
// unreadable, not matching the content, a spec node that is not spec.md, a
// duplicate title, a link of a spec that names no file or no heading, a bad
// link of a directory node, a source the node may not
// list, a code node that lists no technical spec, or a perspective node the
// code node lists that none of its technical specs serves.
func (g *Graph) NodeDrift(n *Node) []string {
	var d []string
	for _, f := range g.findings(n) {
		d = append(d, f.msg)
	}
	return d
}

func (g *Graph) findings(n *Node) []finding {
	var d []finding
	block := func(msg string) { d = append(d, finding{msg, true}) }
	warn := func(msg string) { d = append(d, finding{msg, false}) }
	if msg := malformed(n); msg != "" {
		block(msg)
	}
	if n.RuleErr != nil {
		block(fmt.Sprintf("`%s`: %v", n.Rel, n.RuleErr))
	}
	switch {
	case !n.HasMeta && n.Kind == Technical:
		block(fmt.Sprintf("`%s` has no %s; write its skeleton `{\"version\": %d, \"sources\": [...]}` (/sdd-design)", n.Rel, path.Base(n.MetaRel), meta.Version))
	case !n.HasMeta:
		block(fmt.Sprintf("`%s` has no %s; run metadata --write %s", n.Rel, path.Base(n.MetaRel), n.Rel))
	case n.MetaErr != nil:
		block(fmt.Sprintf("`%s`: %v", n.MetaRel, n.MetaErr))
	case n.Meta.Hash != n.Hash || !meta.SameRules(n.Meta.Rules, n.Rules):
		block(fmt.Sprintf("`%s` does not match the content; run metadata --write %s", n.MetaRel, n.Rel))
	case n.Kind == Code && n.Meta.Audited == nil:
		block(fmt.Sprintf("`%s` carries no audited; run metadata --write %s", n.MetaRel, n.Rel))
	case n.Kind != Code && n.Meta.Audited != nil:
		block(fmt.Sprintf("`%s` carries audited, which only a code node has; run metadata --write %s", n.MetaRel, n.Rel))
	}
	if n.Meta != nil && !n.IsDir && len(n.Meta.Files) > 0 {
		block(fmt.Sprintf("`%s` carries files, which only a directory node has", n.MetaRel))
	}
	d = append(d, n.linkErrs...)
	for _, msg := range n.refErrs {
		warn(msg)
	}
	for _, e := range n.Sources {
		switch {
		case e.To == nil:
		case e.To == n:
			block(fmt.Sprintf("`%s` lists itself as a source", n.Rel))
		case !n.Kind.MaySource(e.To.Kind):
			block(fmt.Sprintf("`%s` (%s) lists `%s` (%s) as a source; %s", n.Rel, n.Kind, e.To.Rel, e.To.Kind, allowed(n.Kind)))
		}
	}
	if n.Kind == Code && n.Meta != nil {
		if msg := g.noTechnical(n); msg != "" {
			warn(msg)
		} else if !g.danglingTechnical(n) { // which perspectives are served is unknown until /sdd-implement drops it
			served, _ := g.served(n)
			for _, e := range n.Sources {
				if e.Valid() && e.To.Kind == Perspective {
					if _, ok := served[e.To]; !ok {
						warn(fmt.Sprintf("`%s` lists `%s`, but none of its technical specs serves it", n.Rel, e.To.Rel))
					}
				}
			}
		}
	}
	return d
}

// danglingTechnical reports whether a code node lists a technical path that is no node.
func (g *Graph) danglingTechnical(n *Node) bool {
	for _, e := range n.Sources {
		if k, ok := g.kindOf(cleanRel(e.Source.Path)); ok && k == Technical && e.Dangling() {
			return true
		}
	}
	return false
}

// noTechnical names a code node whose sources hold no technical path. It reads
// the kind of each path, not the validity of its edge, so a node whose only
// technical source is dangling is reported as dangling and not also here.
func (g *Graph) noTechnical(n *Node) string {
	if n.Kind != Code || n.Meta == nil {
		return ""
	}
	for _, s := range n.Meta.Sources {
		if k, ok := g.kindOf(cleanRel(s.Path)); ok && k == Technical {
			return ""
		}
	}
	return fmt.Sprintf("`%s` lists no technical spec; a code node lists at least one", n.Rel)
}

// malformed names a node that cannot be what it claims: orphan metadata, or
// a perspective or technical node that is not spec.md.
func malformed(n *Node) string {
	switch {
	case !n.IsDir && !n.HasFile:
		return fmt.Sprintf("`%s` has no %s beside it; delete it, or rename it with its file", n.MetaRel, path.Base(n.Rel))
	case n.spec() && path.Base(n.Rel) != meta.SpecFile:
		return fmt.Sprintf("`%s` is no %s; a %s node is %s; delete its metadata", n.Rel, meta.SpecFile, n.Kind, meta.SpecFile)
	}
	return ""
}

// blocking lists the drift of n that makes its content or metadata not what
// it claims.
func (g *Graph) blocking(n *Node) []string {
	var out []string
	for _, f := range g.findings(n) {
		if f.blocking {
			out = append(out, f.msg)
		}
	}
	return out
}

// HasDrift reports whether NodeDrift finds anything.
func (g *Graph) HasDrift(n *Node) bool { return len(g.NodeDrift(n)) > 0 }

// Drift lists every node's drift, then metadata files that are no node,
// files under the specs root or the code root in no node, technical specs
// that source each other in a cycle, and gaps in TODO.md that match no rule
// or no code node owing an audit.
func (g *Graph) Drift() []string {
	var d []string
	for _, n := range g.Nodes {
		d = append(d, g.NodeDrift(n)...)
	}
	d = append(d, g.Loose(nil)...)
	if len(g.cycle) > 0 {
		d = append(d, "technical specs source each other in a cycle: `"+strings.Join(g.cycle, "`, `")+"`")
	}
	d = append(d, g.gapDrift...)
	d = append(d, g.unmatchedGaps()...)
	return d
}

// Design lists what /sdd-design owes: perspective specs no technical spec serves,
// technical specs stale against a source (with the rules that changed, and
// the spec gap that parks one), and dangling sources of technical specs with
// the rules recorded for them.
func (g *Graph) Design() []string {
	var out []string
	for _, n := range g.Nodes {
		switch n.Kind {
		case Perspective:
			if n.HasFile && len(g.consumers(n, Technical)) == 0 {
				out = append(out, fmt.Sprintf("`%s` has no technical spec", n.Rel))
			}
		case Technical:
			for _, e := range n.Sources {
				switch {
				case e.Dangling():
					line := fmt.Sprintf("`%s` has a dangling source `%s`%s", n.Rel, e.Source.Path, g.hint(e.Source.Path))
					if titles := e.recorded(); len(titles) > 0 {
						line += ": removed " + quoted(titles)
					}
					out = append(out, line)
				case e.Stale():
					line := fmt.Sprintf("`%s` is stale against `%s`: %s", n.Rel, e.To.Rel, e.Delta)
					for _, gap := range g.gapsFor("spec", e.To.Rel) {
						if _, ok := e.To.Rules[gap.Title]; ok || contains(e.Delta.Removed, gap.Title) {
							line += fmt.Sprintf("; %q parked: %s (/sdd-spec)", gap.Title, gap.Reason)
						}
					}
					out = append(out, line)
				}
			}
		}
	}
	return out
}

// RuleState is one rule a unit has to run, and why.
type RuleState struct {
	Title  string
	State  string // new, changed, removed, renamed from "<previous title>"
	Source string // the node that owns the rule
	Parked string // the reason of the ## Design gap item that parks it, if any
}

// NodeUnit is one pending code node inside a directory unit.
type NodeUnit struct {
	Node   *Node
	Reason string
	Wait   string // empty when the node can run
	Rules  []RuleState
}

// Unit is one item of the /sdd-implement backlog: a technical spec with no code
// node, or a directory with pending code nodes — stale against a source,
// holding a dangling one, or missing a perspective node their technical specs
// serve — with the union of their rules.
type Unit struct {
	Node   *Node      // the technical spec; nil for a directory unit
	Dir    string     // the directory of the code nodes; empty for a technical spec
	Nodes  []NodeUnit // the pending code nodes of Dir, by path
	Reason string
	Wait   string // empty when the unit can run; otherwise what /sdd-design owes first
	Rules  []RuleState
}

// Rel names the unit: the technical spec, or the directory with a trailing slash.
func (u Unit) Rel() string {
	if u.Node != nil {
		return u.Node.Rel
	}
	return u.Dir + "/"
}

// Units lists the /sdd-implement backlog, sources first: a unit sits after every
// technical spec it realizes or depends on.
func (g *Graph) Units() []Unit {
	type keyed struct {
		pos  int
		kind Kind
		u    Unit
	}
	var all []keyed
	for _, t := range g.topo {
		if len(g.consumers(t, Code)) == 0 {
			all = append(all, keyed{g.pos[t], Technical, g.newUnit(t)})
		}
	}
	dirs := map[string]*keyed{}
	var order []string
	for _, c := range g.Nodes {
		if c.Kind != Code {
			continue
		}
		nu, ok := g.codeUnit(c)
		if !ok {
			continue
		}
		p := -1
		for _, e := range c.Sources {
			if e.Valid() && e.To.Kind == Technical && g.pos[e.To] > p {
				p = g.pos[e.To]
			}
		}
		dir := c.UnitDir()
		k := dirs[dir]
		if k == nil {
			k = &keyed{p, Code, Unit{Dir: dir}}
			dirs[dir] = k
			order = append(order, dir)
		}
		if p > k.pos {
			k.pos = p
		}
		k.u.Nodes = append(k.u.Nodes, nu)
	}
	for _, dir := range order {
		k := dirs[dir]
		k.u.Reason, k.u.Wait, k.u.Rules = groupUnit(k.u.Nodes)
		all = append(all, *k)
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.pos != b.pos {
			return a.pos < b.pos
		}
		if a.kind != b.kind {
			return a.kind < b.kind
		}
		return a.u.Rel() < b.u.Rel()
	})
	units := make([]Unit, len(all))
	for i, k := range all {
		units[i] = g.park(k.u)
	}
	return units
}

// groupUnit folds the pending nodes of one directory: the reasons grouped by
// the nodes that share them, the wait when every node waits, and the union
// of their rules in the order of first appearance.
func groupUnit(nodes []NodeUnit) (reason, wait string, rules []RuleState) {
	var reasons []string
	labels := map[string][]string{}
	waits := 0
	seen := map[[2]string]bool{}
	for _, nu := range nodes {
		if _, ok := labels[nu.Reason]; !ok {
			reasons = append(reasons, nu.Reason)
		}
		labels[nu.Reason] = append(labels[nu.Reason], nu.Node.Label())
		if nu.Wait != "" {
			waits++
		}
		for _, r := range nu.Rules {
			key := [2]string{r.Source, r.Title}
			if !seen[key] {
				seen[key] = true
				rules = append(rules, r)
			}
		}
	}
	parts := make([]string, len(reasons))
	for i, r := range reasons {
		parts[i] = strings.Join(labels[r], ", ") + ": " + r
	}
	reason = strings.Join(parts, "; ")
	if len(nodes) > 0 && waits == len(nodes) {
		wait = nodes[0].Wait
	}
	return reason, wait, rules
}

// newUnit is a technical spec no code node realizes: every rule it and the
// perspective specs it serves hold is new. A spec that serves no perspective
// is a global constraint: the fix is to list it from the code it constrains.
func (g *Graph) newUnit(t *Node) Unit {
	u := Unit{Node: t, Reason: "no code node lists it; list it from the code nodes it constrains", Wait: g.wait(t, nil)}
	for _, e := range sortedEdges(t.Sources) {
		if e.Valid() && e.To.Kind == Perspective {
			u.Reason = "no code node"
			for _, title := range e.To.Order {
				u.Rules = append(u.Rules, RuleState{Title: title, State: "new", Source: e.To.Rel})
			}
		}
	}
	for _, title := range t.Order {
		u.Rules = append(u.Rules, RuleState{Title: title, State: "new", Source: t.Rel})
	}
	return u
}

// codeUnit is a code node with a stale or dangling source, or missing a
// perspective node its technical specs serve: removed and renamed rules
// first, then new and changed ones in file order, perspective sources before
// technical ones, then the rules of the missing perspective nodes.
func (g *Graph) codeUnit(c *Node) (NodeUnit, bool) {
	u := NodeUnit{Node: c}
	var stale, dangling, missing []string
	var first, rest []RuleState
	for _, e := range sortedEdges(c.Sources) {
		switch {
		case e.Dangling():
			dangling = append(dangling, "`"+e.Source.Path+"`"+g.hint(e.Source.Path))
			for _, t := range e.recorded() {
				first = append(first, RuleState{Title: t, State: "removed", Source: e.Source.Path})
			}
		case e.Stale():
			stale = append(stale, "`"+e.To.Rel+"`")
			for _, t := range e.Delta.Removed {
				first = append(first, RuleState{Title: t, State: "removed", Source: e.To.Rel})
			}
			for _, r := range e.Delta.Renamed {
				first = append(first, RuleState{Title: r[1], State: fmt.Sprintf("renamed from %q", r[0]), Source: e.To.Rel})
			}
			states := map[string]string{}
			for _, t := range e.Delta.Added {
				states[t] = "new"
			}
			for _, t := range e.Delta.Changed {
				states[t] = "changed"
			}
			for _, t := range e.To.Order {
				if s := states[t]; s != "" {
					rest = append(rest, RuleState{Title: t, State: s, Source: e.To.Rel})
				}
			}
		}
	}
	served, _ := g.served(c)
	for _, p := range g.missing(c) {
		missing = append(missing, fmt.Sprintf("`%s` (served by `%s`)", p.Rel, served[p].Rel))
		for _, title := range p.Order {
			rest = append(rest, RuleState{Title: title, State: "new", Source: p.Rel})
		}
	}
	if len(stale)+len(dangling)+len(missing) == 0 {
		return u, false
	}
	var reasons []string
	if len(stale) > 0 {
		reasons = append(reasons, "stale against "+strings.Join(stale, ", "))
	}
	if len(dangling) > 0 {
		reasons = append(reasons, "dangling source "+strings.Join(dangling, ", "))
	}
	if len(missing) > 0 {
		reasons = append(reasons, "missing source "+strings.Join(missing, ", "))
	}
	u.Reason = strings.Join(reasons, "; ")
	u.Rules = append(first, rest...)
	u.Wait = g.codeWait(c)
	return u, true
}

// codeWait names what /sdd-design owes before code node c can run.
func (g *Graph) codeWait(c *Node) string {
	for _, e := range c.Sources {
		if e.Valid() && e.To.Kind == Technical {
			if w := g.wait(e.To, nil); w != "" {
				return w
			}
		}
	}
	return ""
}

// wait names what /sdd-design owes before code against t can run: t or a
// technical spec upstream of it is stale or has a dangling source.
func (g *Graph) wait(t *Node, seen map[*Node]bool) string {
	if seen == nil {
		seen = map[*Node]bool{}
	}
	if seen[t] {
		return ""
	}
	seen[t] = true
	for _, e := range t.Sources {
		if e.Dangling() {
			return fmt.Sprintf("`%s` has a dangling source `%s`%s (/sdd-design)", t.Rel, e.Source.Path, g.hint(e.Source.Path))
		}
		if e.Stale() {
			return fmt.Sprintf("`%s` is stale against `%s` (/sdd-design)", t.Rel, e.To.Rel)
		}
	}
	for _, e := range t.Sources {
		if e.Valid() && e.To.Kind == Technical {
			if w := g.wait(e.To, seen); w != "" {
				return w
			}
		}
	}
	return ""
}

// Batch splits the backlog into the units /sdd-implement continue runs now — the
// first max that can run, sources first — and the units that wait on /sdd-design.
func (g *Graph) Batch(max int) (run, wait []Unit) {
	for _, u := range g.Units() {
		switch {
		case u.Wait != "":
			wait = append(wait, u)
		case len(run) < max:
			run = append(run, u)
		}
	}
	return run, wait
}

// sortedEdges orders edges: valid perspective targets, valid technical
// targets, then the rest; ties by path.
func sortedEdges(edges []*Edge) []*Edge {
	out := append([]*Edge(nil), edges...)
	rank := func(e *Edge) int {
		if !e.Valid() {
			return 2
		}
		if e.To.Kind == Perspective {
			return 0
		}
		return 1
	}
	sort.SliceStable(out, func(i, j int) bool {
		if rank(out[i]) != rank(out[j]) {
			return rank(out[i]) < rank(out[j])
		}
		return out[i].Source.Path < out[j].Source.Path
	})
	return out
}

// SourcesFirst orders nodes so every node comes after its sources: perspective
// nodes, then technical nodes sources first, then code nodes; ties by path.
func (g *Graph) SourcesFirst(nodes []*Node) []*Node {
	out := append([]*Node(nil), nodes...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Kind == Technical && g.pos[a] != g.pos[b] {
			return g.pos[a] < g.pos[b]
		}
		return a.Rel < b.Rel
	})
	return out
}

func sortedNodes(m map[*Node]*Node) []*Node {
	out := make([]*Node, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out
}

func quoted(titles []string) string {
	q := make([]string, len(titles))
	for i, t := range titles {
		q[i] = fmt.Sprintf("%q", t)
	}
	return strings.Join(q, ", ")
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
