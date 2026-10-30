// Package graph loads every node under the specs root and the code root of
// .sdd/.sdd-config.json, computes the current hash and rules of each, resolves
// the sources they list, reads the gaps parked in TODO.md of the specs root,
// and derives the states of **Metadata**
// in the root AGENTS.md: drift, stale, dangling, unconsumed, fresh, waits — and
// the audit standing of each code node (audit.go).
package graph

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"tools/internal/config"
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

// Node is a directory with .metadata.json, or a directory under the specs root with spec.md.
type Node struct {
	Rel     string // root-relative directory, forward slashes
	Dir     string // absolute directory
	Kind    Kind
	HasSpec bool
	HasMeta bool
	Meta    *meta.Metadata // nil when absent or unreadable
	MetaErr error
	Hash    string            // current node hash
	Rules   map[string]string // current rule hashes; nil without spec.md
	Order   []string          // rule titles in file order
	RuleErr error
	Sources []*Edge // one per Meta.Sources entry, in order
	content []string
}

// Files lists the node's content, paths relative to the node, sorted.
func (n *Node) Files() []string { return append([]string(nil), n.content...) }

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
	topo      []*Node // technical nodes, sources first
	pos       map[*Node]int
	cycle     []string
	uncovered map[string]int
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
	g := &Graph{Root: root, Paths: cfg.Paths, byRel: map[string]*Node{}, pos: map[*Node]int{}, uncovered: map[string]int{}}
	node := func(rel string) *Node {
		if n := g.byRel[rel]; n != nil {
			return n
		}
		k, ok := g.kindOf(rel)
		if !ok {
			return nil
		}
		n := &Node{Rel: rel, Dir: filepath.Join(root, filepath.FromSlash(rel)), Kind: k}
		g.byRel[rel] = n
		g.Nodes = append(g.Nodes, n)
		return n
	}
	for _, f := range files {
		dir, base := path.Dir(f), path.Base(f)
		if base == meta.FileName {
			if n := node(dir); n != nil {
				n.HasMeta = true
			}
		} else if base == meta.SpecFile && config.Under(dir, g.Paths.Specs) {
			if n := node(dir); n != nil {
				n.HasSpec = true
			}
		}
	}
	for _, f := range files {
		if path.Base(f) == meta.SpecFile {
			if n := g.byRel[path.Dir(f)]; n != nil {
				n.HasSpec = true
			}
		}
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].Rel < g.Nodes[j].Rel })
	for _, f := range files {
		if g.isChrome(f) {
			continue
		}
		n := g.Owner(f)
		if n == nil {
			if config.Under(f, g.Paths.Src) {
				g.uncovered[path.Dir(f)]++
			}
			continue
		}
		if base := path.Base(f); base == meta.FileName || base == meta.LawFile {
			continue
		}
		n.content = append(n.content, strings.TrimPrefix(f, n.Rel+"/"))
	}
	for _, n := range g.Nodes {
		if n.Hash, err = meta.HashFiles(n.Dir, n.content); err != nil {
			return nil, err
		}
		if n.HasSpec {
			text, err := os.ReadFile(filepath.Join(n.Dir, meta.SpecFile))
			if err != nil {
				return nil, err
			}
			n.Rules, n.Order, n.RuleErr = meta.RuleHashes(string(text))
		}
		if n.HasMeta {
			n.Meta, n.MetaErr = meta.Read(n.Dir)
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

// isChrome reports whether a file directly under the code root ending in .md
// is repository chrome: no node's content, never drift.
func (g *Graph) isChrome(f string) bool {
	return path.Dir(f) == g.Paths.Src && strings.HasSuffix(f, ".md")
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

// Staged returns the nodes holding a file staged for commit, sorted by path.
// Renames are listed as a deletion and an addition, so both nodes refresh.
func (g *Graph) Staged() ([]*Node, error) {
	cmd := exec.Command("git", "-C", g.Root, "diff", "--cached", "--name-only", "--no-renames", "-z", "--relative")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff --cached: %s", strings.TrimSpace(errb.String()))
	}
	seen := map[*Node]bool{}
	var nodes []*Node
	for _, f := range strings.Split(out.String(), "\x00") {
		if f == "" {
			continue
		}
		if n := g.Owner(f); n != nil && !seen[n] {
			seen[n] = true
			nodes = append(nodes, n)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Rel < nodes[j].Rel })
	return nodes, nil
}

// kindOf is the kind of a root-relative directory by its place in the layout.
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

// order sorts the technical nodes sources first (ties by path) and records
// the members of a cycle, which come last.
func (g *Graph) order() {
	var ts []*Node
	for _, n := range g.Nodes {
		if n.Kind == Technical {
			ts = append(ts, n)
		}
	}
	indeg := map[*Node]int{}
	next := map[*Node][]*Node{}
	for _, n := range ts {
		for _, e := range n.Sources {
			if e.Valid() && e.To.Kind == Technical {
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

// Node returns the node at a root-relative directory, or nil.
func (g *Graph) Node(rel string) *Node { return g.byRel[rel] }

// Owner returns the deepest node whose directory holds the root-relative file f.
func (g *Graph) Owner(f string) *Node {
	for dir := path.Dir(f); dir != "." && dir != "/" && dir != ""; dir = path.Dir(dir) {
		if n := g.byRel[dir]; n != nil {
			return n
		}
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

// NodeDrift lists what is wrong with n's own metadata: missing, unreadable,
// not matching the content, a spec-less perspective or technical node, a
// duplicate title, a source the node may not list, a code node that lists no
// technical spec, or a perspective node the code node lists that none of its
// technical specs serves.
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
	file := n.Rel + "/" + meta.FileName
	if msg := specless(n); msg != "" {
		block(msg)
	}
	if n.RuleErr != nil {
		block(fmt.Sprintf("`%s/%s`: %v", n.Rel, meta.SpecFile, n.RuleErr))
	}
	switch {
	case !n.HasMeta:
		block(fmt.Sprintf("`%s/%s` has no %s; run metadata --write %s", n.Rel, meta.SpecFile, meta.FileName, n.Rel))
	case n.MetaErr != nil:
		block(fmt.Sprintf("`%s`: %v", file, n.MetaErr))
	case n.Meta.Hash != n.Hash || !meta.SameRules(n.Meta.Rules, n.Rules):
		block(fmt.Sprintf("`%s` does not match the content; run metadata --write %s", file, n.Rel))
	case n.Kind == Code && n.Meta.Audited == nil:
		block(fmt.Sprintf("`%s` carries no audited; run metadata --write %s", file, n.Rel))
	case n.Kind != Code && n.Meta.Audited != nil:
		block(fmt.Sprintf("`%s` carries audited, which only a code node has; run metadata --write %s", file, n.Rel))
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

// specless names a perspective or technical node without spec.md.
func specless(n *Node) string {
	switch {
	case n.Kind == Perspective && !n.HasSpec:
		return fmt.Sprintf("`%s/%s` has no %s beside it; delete it", n.Rel, meta.FileName, meta.SpecFile)
	case n.Kind == Technical && !n.HasSpec:
		return fmt.Sprintf("`%s` has no %s; a technical node holds one", n.Rel, meta.SpecFile)
	}
	return ""
}

// HasDrift reports whether NodeDrift finds anything.
func (g *Graph) HasDrift(n *Node) bool { return len(g.NodeDrift(n)) > 0 }

// Drift lists every node's drift, then files under the code root outside any node,
// technical specs that source each other in a cycle, and gaps in TODO.md
// that match no rule or no code node owing an audit.
func (g *Graph) Drift() []string {
	var d []string
	for _, n := range g.Nodes {
		d = append(d, g.NodeDrift(n)...)
	}
	dirs := make([]string, 0, len(g.uncovered))
	for dir := range g.uncovered {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		d = append(d, fmt.Sprintf("`%s/` holds %d file(s) outside any node", dir, g.uncovered[dir]))
	}
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
			if n.HasSpec && len(g.consumers(n, Technical)) == 0 {
				out = append(out, fmt.Sprintf("`%s` has no technical spec", n.Rel))
			}
		case Technical:
			for _, e := range n.Sources {
				switch {
				case e.Dangling():
					line := fmt.Sprintf("`%s` has a dangling source `%s`", n.Rel, e.Source.Path)
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

// Unit is one item of the /sdd-implement backlog: a technical spec with no code
// node, or a code node stale against a source, holding a dangling one, or
// missing a perspective node its technical specs serve.
type Unit struct {
	Node   *Node
	Reason string
	Wait   string // empty when the unit can run; otherwise what /sdd-design owes first
	Rules  []RuleState
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
	for _, c := range g.Nodes {
		if c.Kind != Code {
			continue
		}
		u, ok := g.codeUnit(c)
		if !ok {
			continue
		}
		p := -1
		for _, e := range c.Sources {
			if e.Valid() && e.To.Kind == Technical && g.pos[e.To] > p {
				p = g.pos[e.To]
			}
		}
		all = append(all, keyed{p, Code, u})
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.pos != b.pos {
			return a.pos < b.pos
		}
		if a.kind != b.kind {
			return a.kind < b.kind
		}
		return a.u.Node.Rel < b.u.Node.Rel
	})
	units := make([]Unit, len(all))
	for i, k := range all {
		units[i] = g.park(k.u)
	}
	return units
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
func (g *Graph) codeUnit(c *Node) (Unit, bool) {
	u := Unit{Node: c}
	var stale, dangling, missing []string
	var first, rest []RuleState
	for _, e := range sortedEdges(c.Sources) {
		switch {
		case e.Dangling():
			dangling = append(dangling, "`"+e.Source.Path+"`")
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
			return fmt.Sprintf("`%s` has a dangling source `%s` (/sdd-design)", t.Rel, e.Source.Path)
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
