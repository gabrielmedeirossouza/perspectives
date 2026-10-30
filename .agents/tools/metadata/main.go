// metadata: does the metadata of each node reflect its content? It also
// writes it, records which version of each source a node validated against,
// and records the approvals of /sdd-audit on a code node.
//
// Usage: go run -C .agents/tools ./metadata [--write | --sync | --approve | --staged] [--root <dir>] [node|dir...]
//
//	node|dir...   root-relative targets (default: every node; --sync and
//	              --approve require them). A node is a file path
//	              (specs/business/invite/spec.md, src/api/auth/db.ts) or the
//	              path of its metadata file (db.ts.metadata.json,
//	              src/api/.metadata.json); a directory names its directory
//	              node, when it has one, and every node at or under it.
//	              --sync and --approve take one node per target and refuse a
//	              directory that names several
//	(no flag)     check: list every drift of the targeted nodes; fail if any
//	--write       recompute hash and rules; move modified only when the hash
//	              changed; on a code node, return audited to 0 when the hash
//	              changed; create spec.md.metadata.json for a spec.md of the
//	              specs root; never touch sources or files
//	--sync        --write, then record the current hash and rules of every
//	              source, returning audited to 0 when a source is recorded for
//	              the first time; refuses a dangling or invalid source, one
//	              whose own metadata has drift, a code node with no technical
//	              spec or whose sources do not match the perspective nodes its
//	              technical specs serve, and a code node that waits on /sdd-design
//	--approve     --write, then add one approval to audited: a judge confirmed
//	              the sources of the code node; refuses a node that is no code
//	              node, has drift, changed since the judge read it, or is an
//	              /sdd-implement unit (stale, dangling, missing a served
//	              perspective)
//	--staged      --write on every node holding a staged file, then git add
//	              the metadata of every such node; the pre-commit hook runs
//	              this. It hashes the working tree, so a partially staged file
//	              is hashed as it is on disk. A renamed file whose metadata
//	              did not move with it leaves an orphan, which is refused, as
//	              is a staged file that belongs to no node
//	--root <dir>  repository root, absolute (default: nearest directory at or
//	              above the working directory with .sdd/.sdd-config.json)
//
// Output, plain text, one line per node: ok, written, synced, approved (with
// the count), unchanged, staged, or the drift found. Exit 0 when done; 1 when a check
// fails or a sync or approval is refused; 2 on a usage error (go run reports
// every failure as 1).
//
// It implements **Metadata** of the root AGENTS.md; that section wins over it.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"tools/internal/graph"
	"tools/internal/meta"
	"tools/internal/repo"
)

const usage = "usage: go run -C .agents/tools ./metadata [--write | --sync | --approve | --staged] [--root <dir>] [node|dir...]"

type options struct {
	write, sync, approve, staged bool
	root                         string
	dirs                         []string
}

func main() {
	opts := parse(os.Args[1:])
	root, err := repo.Root(opts.root)
	check(err)
	g, err := graph.Load(root)
	check(err)

	var targets []*graph.Node
	var uncovered []string
	switch {
	case opts.staged:
		targets, uncovered, err = g.Staged()
	case len(opts.dirs) == 0:
		targets = g.Nodes
	default:
		targets, err = resolve(g, root, opts.dirs, opts.sync || opts.approve)
	}
	check(err)
	for _, line := range uncovered {
		fmt.Fprintf(os.Stderr, "metadata: %s\n", line)
	}
	targets = g.SourcesFirst(targets)
	if len(targets) == 0 && !opts.staged {
		fmt.Printf("no node under %s/ or %s/\n", g.Paths.Specs, g.Paths.Src)
		return
	}

	now := time.Now()
	failed := len(uncovered) > 0
	var stage []string
	for _, n := range targets {
		switch {
		case opts.sync:
			m, err := g.Sync(n, now)
			if err != nil {
				fmt.Fprintf(os.Stderr, "metadata: %v\n", err)
				failed = true
				continue
			}
			written, err := graph.Save(n, m)
			check(err)
			report(written, "synced", n.MetaRel)
		case opts.approve:
			m, err := g.Approve(n, now)
			if err != nil {
				fmt.Fprintf(os.Stderr, "metadata: %v\n", err)
				failed = true
				continue
			}
			_, err = graph.Save(n, m)
			check(err)
			fmt.Printf("approved %s (%d)\n", n.MetaRel, *m.Audited)
		case opts.write || opts.staged:
			m, err := graph.Refresh(n, now)
			if err != nil {
				fmt.Fprintf(os.Stderr, "metadata: %v\n", err)
				failed = true
				continue
			}
			written, err := graph.Save(n, m)
			check(err)
			if opts.staged {
				fmt.Println("staged " + n.MetaRel)
				stage = append(stage, n.MetaRel)
			} else {
				report(written, "written", n.MetaRel)
			}
		default:
			if d := g.NodeDrift(n); len(d) > 0 {
				for _, line := range d {
					fmt.Println(line)
				}
				failed = true
			} else {
				fmt.Println("ok " + n.MetaRel)
			}
		}
	}
	if len(stage) > 0 {
		args := append([]string{"-C", root, "add", "--"}, stage...)
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			fail(fmt.Sprintf("git add: %s", strings.TrimSpace(string(out))), 1)
		}
	}
	if failed {
		os.Exit(1)
	}
}

func parse(args []string) options {
	var o options
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--write":
			o.write = true
		case a == "--sync":
			o.sync = true
		case a == "--approve":
			o.approve = true
		case a == "--staged":
			o.staged = true
		case a == "--root" && i+1 < len(args):
			i++
			o.root = args[i]
		case strings.HasPrefix(a, "-"):
			fail("unknown argument: "+a+"\n"+usage, 2)
		default:
			o.dirs = append(o.dirs, a)
		}
	}
	modes := 0
	for _, on := range []bool{o.write, o.sync, o.approve, o.staged} {
		if on {
			modes++
		}
	}
	if modes > 1 {
		fail("--write, --sync, --approve, and --staged exclude each other\n"+usage, 2)
	}
	if o.staged && len(o.dirs) > 0 {
		fail("--staged takes no target\n"+usage, 2)
	}
	if o.sync && len(o.dirs) == 0 {
		fail("--sync takes the nodes to sync; it never syncs every node\n"+usage, 2)
	}
	if o.approve && len(o.dirs) == 0 {
		fail("--approve takes the code nodes a judge approved; it never approves every node\n"+usage, 2)
	}
	return o
}

// resolve maps root-relative (or absolute) targets to nodes, each once: a
// node path or its metadata file names one node; a directory names its
// directory node and every node at or under it — refused when single is
// set and the directory names more than one.
func resolve(g *graph.Graph, root string, targets []string, single bool) ([]*graph.Node, error) {
	var out []*graph.Node
	seen := map[*graph.Node]bool{}
	add := func(nodes ...*graph.Node) {
		for _, n := range nodes {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	for _, d := range targets {
		rel := d
		if filepath.IsAbs(d) {
			rel = repo.Rel(root, d)
		}
		rel = repo.Clean(rel)
		base := path.Base(rel)
		var nodes []*graph.Node
		switch {
		case base == meta.FileName:
			if n := g.Node(path.Dir(rel)); n != nil && n.IsDir {
				nodes = []*graph.Node{n}
			}
		case strings.HasSuffix(base, meta.Suffix):
			if n := g.Node(strings.TrimSuffix(rel, meta.Suffix)); n != nil && !n.IsDir {
				nodes = []*graph.Node{n}
			}
		default:
			if n := g.Node(rel); n != nil && !n.IsDir {
				nodes = []*graph.Node{n}
			} else {
				nodes = g.NodesUnder(rel)
			}
		}
		if len(nodes) == 0 {
			return nil, fmt.Errorf("`%s` is no node: a node is a file with <file>%s beside it, or a directory under %s/ whose %s links its files; a directory names every node at or under it", rel, meta.Suffix, g.Paths.Src, meta.FileName)
		}
		if single && len(nodes) > 1 {
			return nil, fmt.Errorf("`%s` names %d nodes; --sync and --approve take one node per target: a file path, or %s/%s for the directory node", rel, len(nodes), rel, meta.FileName)
		}
		add(nodes...)
	}
	return out, nil
}

func report(written bool, verb, file string) {
	if written {
		fmt.Println(verb + " " + file)
	} else {
		fmt.Println("unchanged " + file)
	}
}

func check(err error) {
	if err != nil {
		fail(err.Error(), 1)
	}
}

func fail(msg string, code int) {
	fmt.Fprintf(os.Stderr, "metadata: %s\n", msg)
	os.Exit(code)
}
