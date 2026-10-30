// metadata: does the .metadata.json of each node reflect its content? It also
// writes it, records which version of each source a node validated against,
// and records the approvals of /audit on a code node.
//
// Usage: go run -C .agents/tools ./metadata [--write | --sync | --approve | --staged] [--root <dir>] [dir...]
//
//	dir...        root-relative node directories (default: every node; --sync
//	              and --approve require them)
//	(no flag)     check: list every drift of the targeted nodes; fail if any
//	--write       recompute hash and rules; move modified only when the hash
//	              changed; on a code node, return audited to 0 when the hash
//	              changed; create the file for a directory of the specs root
//	              that holds spec.md; never touch sources
//	--sync        --write, then record the current hash and rules of every
//	              source, returning audited to 0 when a source is recorded for
//	              the first time; refuses a dangling or invalid source, one
//	              whose own metadata has drift, a code node with no technical
//	              spec or whose sources do not match the perspective nodes its
//	              technical specs serve, and a code node that waits on /design
//	--approve     --write, then add one approval to audited: a judge confirmed
//	              the sources of the code node; refuses a node that is no code
//	              node, has drift, changed since the judge read it, or is an
//	              /implement unit (stale, dangling, missing a served
//	              perspective)
//	--staged      --write on every node holding a staged file, then git add
//	              the metadata of every such node; the pre-commit hook runs
//	              this. It hashes the working tree, so a partially staged file
//	              is hashed as it is on disk
//	--root <dir>  repository root, absolute (default: nearest directory at or
//	              above the working directory with .sdd/.sdd-config.json)
//
// Output, plain text, one line per node: ok, written, synced, approved (with
// the count), unchanged, or the drift found. Exit 0 when done; 1 when a check
// fails or a sync or approval is refused; 2 on a usage error (go run reports
// every failure as 1).
//
// It implements **Metadata** of the root AGENTS.md; that section wins over it.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tools/internal/graph"
	"tools/internal/meta"
	"tools/internal/repo"
)

const usage = "usage: go run -C .agents/tools ./metadata [--write | --sync | --approve | --staged] [--root <dir>] [dir...]"

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
	switch {
	case opts.staged:
		targets, err = g.Staged()
	case len(opts.dirs) == 0:
		targets = g.Nodes
	default:
		targets, err = resolve(g, root, opts.dirs)
	}
	check(err)
	targets = g.SourcesFirst(targets)
	if len(targets) == 0 && !opts.staged {
		fmt.Printf("no node under %s/ or %s/\n", g.Paths.Specs, g.Paths.Src)
		return
	}

	now := time.Now()
	failed := false
	var stage []string
	for _, n := range targets {
		file := n.Rel + "/" + meta.FileName
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
			report(written, "synced", file)
		case opts.approve:
			m, err := g.Approve(n, now)
			if err != nil {
				fmt.Fprintf(os.Stderr, "metadata: %v\n", err)
				failed = true
				continue
			}
			_, err = graph.Save(n, m)
			check(err)
			fmt.Printf("approved %s (%d)\n", file, *m.Audited)
		case opts.write || opts.staged:
			m, err := graph.Refresh(n, now)
			if err != nil {
				fmt.Fprintf(os.Stderr, "metadata: %v\n", err)
				failed = true
				continue
			}
			written, err := graph.Save(n, m)
			check(err)
			report(written, "written", file)
			if opts.staged {
				stage = append(stage, file)
			}
		default:
			if d := g.NodeDrift(n); len(d) > 0 {
				for _, line := range d {
					fmt.Println(line)
				}
				failed = true
			} else {
				fmt.Println("ok " + file)
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
		fail("--staged takes no dir\n"+usage, 2)
	}
	if o.sync && len(o.dirs) == 0 {
		fail("--sync takes the node directories to sync; it never syncs every node\n"+usage, 2)
	}
	if o.approve && len(o.dirs) == 0 {
		fail("--approve takes the code node directories a judge approved; it never approves every node\n"+usage, 2)
	}
	return o
}

// resolve maps root-relative (or absolute) directories to nodes, each once.
func resolve(g *graph.Graph, root string, dirs []string) ([]*graph.Node, error) {
	var out []*graph.Node
	seen := map[*graph.Node]bool{}
	for _, d := range dirs {
		rel := d
		if filepath.IsAbs(d) {
			rel = repo.Rel(root, d)
		}
		rel = repo.Clean(rel)
		n := g.Node(rel)
		if n == nil {
			return nil, fmt.Errorf("`%s` is no node: a directory under %s/ holds %s; a directory under %s/ holds %s with its sources", rel, g.Paths.Specs, meta.SpecFile, g.Paths.Src, meta.FileName)
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
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
