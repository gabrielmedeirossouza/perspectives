// Package meta is the metadata of a node: its shape, its byte-stable
// encoding, the node and rule hashes, and the rule-level delta between two
// rule maps. The law is **Metadata** in the root AGENTS.md.
package meta

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tools/internal/md"
)

const (
	Version   = 2
	Algorithm = "sha256"
	Suffix    = ".metadata.json" // beside a file: <file>.metadata.json
	FileName  = ".metadata.json" // of a directory node: <dir>/.metadata.json
	SpecFile  = "spec.md"
	LawFile   = "AGENTS.md"
)

// Source is one upstream node with the hashes it had when this node was last synced.
type Source struct {
	Path  string            `json:"path"`
	Hash  string            `json:"hash,omitempty"`
	Rules map[string]string `json:"rules,omitempty"`
}

// Metadata is the content of a node's metadata file.
type Metadata struct {
	Version   int                 `json:"version"`
	Algorithm string              `json:"algorithm"`
	Hash      string              `json:"hash"`
	Modified  string              `json:"modified"`
	Audited   *int                `json:"audited,omitempty"` // code nodes only: approvals at the current hash
	Rules     map[string]string   `json:"rules,omitempty"`
	Sources   []Source            `json:"sources,omitempty"`
	Files     map[string][]string `json:"files,omitempty"` // directory nodes only: file → the sources it is linked to
}

// MetaRel is the root-relative metadata file of a node at rel: a directory
// node's .metadata.json, or <rel>.metadata.json beside a file.
func MetaRel(rel string, dir bool) string {
	if dir {
		return rel + "/" + FileName
	}
	return rel + Suffix
}

// Read parses the metadata file at path. A file without a version is a
// skeleton a skill wrote by hand; any version other than Version, and any
// field this shape does not know, is refused.
func Read(path string) (*Metadata, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Metadata
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}
	if m.Version != 0 && m.Version != Version {
		return nil, fmt.Errorf("version %d is not supported; this tool writes version %d", m.Version, Version)
	}
	return &m, nil
}

// Encode renders m byte-stably: fixed field order, sorted map keys, two-space
// indent, no HTML escaping, one trailing newline. A file linked to no source
// is rendered as an empty list, never null.
func Encode(m *Metadata) ([]byte, error) {
	c := *m
	if len(m.Files) > 0 {
		c.Files = make(map[string][]string, len(m.Files))
		for k, v := range m.Files {
			if v == nil {
				v = []string{}
			}
			c.Files[k] = v
		}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Sum is the lowercase hex sha256 of b.
func Sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// HashFiles is the node hash of files (paths relative to dir, forward
// slashes): the sha256 of the sorted entries "<path>\x00<sha256 of bytes>\n".
func HashFiles(dir string, files []string) (string, error) {
	sorted := append([]string(nil), files...)
	sort.Strings(sorted)
	var acc bytes.Buffer
	for _, rel := range sorted {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&acc, "%s\x00%s\n", rel, Sum(b))
	}
	return Sum(acc.Bytes()), nil
}

// RuleHashes maps every ### title of a spec file to the sha256 of its body
// and returns the titles in file order. Titles are unique per file,
// case-insensitively; a duplicate is an error. No rule → nil maps.
func RuleHashes(text string) (map[string]string, []string, error) {
	hashes := map[string]string{}
	var order []string
	seen := map[string]string{}
	for _, r := range md.Rules(text) {
		key := strings.ToLower(r.Title)
		if prev, dup := seen[key]; dup {
			return nil, nil, fmt.Errorf("duplicate rule title %q (also %q)", r.Title, prev)
		}
		seen[key] = r.Title
		hashes[r.Title] = Sum([]byte(r.Body))
		order = append(order, r.Title)
	}
	if len(hashes) == 0 {
		return nil, nil, nil
	}
	return hashes, order, nil
}

// Stamp formats t as the value of "modified".
func Stamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// SameRules reports whether two rule maps hold the same titles and hashes;
// nil and empty are equal.
func SameRules(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// Delta is the rule-level difference between the rules recorded at the last
// sync and the current ones.
type Delta struct {
	Added   []string
	Changed []string
	Removed []string
	Renamed [][2]string // {previous title, current title}: same body, new title
}

// Diff compares then (recorded) against now (current). order lists now's
// titles in file order; when nil, titles come out sorted.
func Diff(then, now map[string]string, order []string) Delta {
	var d Delta
	if order == nil {
		for t := range now {
			order = append(order, t)
		}
		sort.Strings(order)
	}
	gone := map[string][]string{} // body hash → recorded titles no longer present
	for t, h := range then {
		if _, ok := now[t]; !ok {
			gone[h] = append(gone[h], t)
		}
	}
	for h := range gone {
		sort.Strings(gone[h])
	}
	for _, t := range order {
		h := now[t]
		if old, ok := then[t]; ok {
			if old != h {
				d.Changed = append(d.Changed, t)
			}
			continue
		}
		if olds := gone[h]; len(olds) > 0 {
			d.Renamed = append(d.Renamed, [2]string{olds[0], t})
			gone[h] = olds[1:]
			continue
		}
		d.Added = append(d.Added, t)
	}
	for _, titles := range gone {
		d.Removed = append(d.Removed, titles...)
	}
	sort.Strings(d.Removed)
	return d
}

// Empty reports whether no rule was added, changed, removed, or renamed.
func (d Delta) Empty() bool {
	return len(d.Added)+len(d.Changed)+len(d.Removed)+len(d.Renamed) == 0
}

// String renders `added "A"; changed "B"; removed "C"; renamed "D" → "E"`,
// or "no rule changed".
func (d Delta) String() string {
	var parts []string
	part := func(label string, titles []string) {
		if len(titles) == 0 {
			return
		}
		q := make([]string, len(titles))
		for i, t := range titles {
			q[i] = fmt.Sprintf("%q", t)
		}
		parts = append(parts, label+" "+strings.Join(q, ", "))
	}
	part("added", d.Added)
	part("changed", d.Changed)
	part("removed", d.Removed)
	for _, r := range d.Renamed {
		parts = append(parts, fmt.Sprintf("renamed %q → %q", r[0], r[1]))
	}
	if len(parts) == 0 {
		return "no rule changed"
	}
	return strings.Join(parts, "; ")
}
