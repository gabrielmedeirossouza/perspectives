// metadata: does each node's metadata match its content? Flags and output: .agents/tools/AGENTS.md.
package main

import (
	"fmt"
	"os"
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

	checking := !opts.write && !opts.sync && !opts.approve && !opts.staged
	var targets []*graph.Node
	var uncovered []string
	switch {
	case opts.staged:
		targets, uncovered, err = g.Staged()
	case len(opts.dirs) == 0:
		targets = g.Nodes
	default:
		targets, err = resolve(g, root, opts.dirs, opts.sync || opts.approve, checking)
	}
	check(err)
	for _, line := range uncovered {
		fmt.Fprintf(os.Stderr, "metadata: %s\n", line)
	}
	targets = g.SourcesFirst(targets)
	var loose []string
	if checking {
		loose = g.Loose(scope(root, opts.dirs))
	}
	if len(targets) == 0 && len(loose) == 0 && !opts.staged {
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
	for _, line := range loose {
		fmt.Println(line)
		failed = true
	}
	if len(stage) > 0 {
		_, err := repo.Git(root, append([]string{"add", "--"}, stage...)...)
		check(err)
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

// resolve maps targets to nodes (graph.Resolve), each once — refused when
// single is set and a target names more than one.
func resolve(g *graph.Graph, root string, targets []string, single, checking bool) ([]*graph.Node, error) {
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
		rel := repo.Target(root, d)
		nodes := g.Resolve(rel)
		if len(nodes) == 0 && checking && len(g.Loose([]string{rel})) > 0 {
			continue // the check lists its files in no node
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

// scope maps the targets to the root-relative paths whose loose files a check
// reports; none means every one.
func scope(root string, targets []string) []string {
	var out []string
	for _, d := range targets {
		out = append(out, repo.Target(root, d))
	}
	return out
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
