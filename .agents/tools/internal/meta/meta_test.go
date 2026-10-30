package meta

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEncodeIsByteStableAndUnescaped(t *testing.T) {
	m := &Metadata{Version: 1, Algorithm: "sha256", Hash: "h", Modified: "2026-01-01T00:00:00Z",
		Rules:   map[string]string{"Z & <last>": "1", "A first": "2"},
		Sources: []Source{{Path: "specs/x", Hash: "s", Rules: map[string]string{"b": "1", "a": "2"}}}}
	a, err := Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Encode(m)
	if !bytes.Equal(a, b) {
		t.Fatal("two encodings differ")
	}
	s := string(a)
	if !strings.HasSuffix(s, "}\n") || strings.Contains(s, `\u003c`) || !strings.Contains(s, "Z & <last>") {
		t.Fatalf("encoding = %s", s)
	}
	if strings.Index(s, `"A first"`) > strings.Index(s, `"Z & <last>"`) || strings.Index(s, `"a"`) > strings.Index(s, `"b"`) {
		t.Fatalf("map keys not sorted: %s", s)
	}
	if strings.Index(s, `"version"`) > strings.Index(s, `"hash"`) || strings.Index(s, `"rules"`) > strings.Index(s, `"sources"`) {
		t.Fatalf("field order: %s", s)
	}
}

func TestEncodeOmitsEmptyRulesAndSources(t *testing.T) {
	a, _ := Encode(&Metadata{Version: 1, Algorithm: "sha256", Hash: "h", Modified: "m"})
	if strings.Contains(string(a), "rules") || strings.Contains(string(a), "sources") {
		t.Fatalf("encoding = %s", a)
	}
}

func TestRuleHashesOrderAndDuplicates(t *testing.T) {
	hashes, order, err := RuleHashes("# F\n\n## S\n\n### B rule\n\nx\n\n### A rule\n\ny\n")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"B rule", "A rule"}) || len(hashes) != 2 || hashes["B rule"] != Sum([]byte("x")) {
		t.Fatalf("hashes = %v order = %v", hashes, order)
	}
	if _, _, err := RuleHashes("### Same\n\na\n\n### same\n\nb\n"); err == nil {
		t.Fatal("duplicate title accepted")
	}
	if hashes, order, err := RuleHashes("# F\n\nno rules\n"); hashes != nil || order != nil || err != nil {
		t.Fatalf("empty spec: %v %v %v", hashes, order, err)
	}
}

func TestDiff(t *testing.T) {
	then := map[string]string{"Kept": "k", "Changed": "c1", "Gone": "g", "Old name": "same"}
	now := map[string]string{"Kept": "k", "Changed": "c2", "New": "n", "New name": "same"}
	d := Diff(then, now, []string{"New name", "Kept", "New", "Changed"})
	want := Delta{Added: []string{"New"}, Changed: []string{"Changed"}, Removed: []string{"Gone"}, Renamed: [][2]string{{"Old name", "New name"}}}
	if !reflect.DeepEqual(d, want) {
		t.Fatalf("delta = %+v, want %+v", d, want)
	}
	if s := d.String(); s != `added "New"; changed "Changed"; removed "Gone"; renamed "Old name" → "New name"` {
		t.Fatalf("string = %s", s)
	}
	if !Diff(now, now, nil).Empty() || Diff(nil, nil, nil).String() != "no rule changed" {
		t.Fatal("identical maps differ")
	}
	if d := Diff(nil, map[string]string{"B": "1", "A": "2"}, nil); !reflect.DeepEqual(d.Added, []string{"A", "B"}) {
		t.Fatalf("nil order not sorted: %v", d.Added)
	}
}

func TestHashFilesDefinition(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{"spec.md": "# A\n", "sub/b.txt": "b"} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	got, err := HashFiles(dir, []string{"sub/b.txt", "spec.md"})
	if err != nil {
		t.Fatal(err)
	}
	want := Sum([]byte("spec.md\x00" + Sum([]byte("# A\n")) + "\n" + "sub/b.txt\x00" + Sum([]byte("b")) + "\n"))
	if got != want {
		t.Fatalf("hash = %s, want %s", got, want)
	}
	if empty, _ := HashFiles(dir, nil); empty != Sum(nil) {
		t.Fatalf("empty node hash = %s", empty)
	}
}

func TestReadRefusesUnknownFieldsAndVersions(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, FileName), []byte(`{"version": 1, "source": []}`), 0o644)
	if _, err := Read(dir); err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("unknown field accepted: %v", err)
	}
	os.WriteFile(filepath.Join(dir, FileName), []byte(`{"version": 2}`), 0o644)
	if _, err := Read(dir); err == nil || !strings.Contains(err.Error(), "version 2") {
		t.Fatalf("version 2 accepted: %v", err)
	}
	os.WriteFile(filepath.Join(dir, FileName), []byte(`{"sources": [{"path": "x"}]}`), 0o644)
	if m, err := Read(dir); err != nil || len(m.Sources) != 1 {
		t.Fatalf("skeleton refused: %v %+v", err, m)
	}
}

func TestEncodeKeepsAuditedZeroBetweenModifiedAndRules(t *testing.T) {
	zero := 0
	a, _ := Encode(&Metadata{Version: 1, Algorithm: "sha256", Hash: "h", Modified: "m", Audited: &zero, Rules: map[string]string{"r": "1"}})
	s := string(a)
	if !strings.Contains(s, `"audited": 0`) || strings.Index(s, `"modified"`) > strings.Index(s, `"audited"`) || strings.Index(s, `"audited"`) > strings.Index(s, `"rules"`) {
		t.Fatalf("encoding = %s", s)
	}
	b, _ := Encode(&Metadata{Version: 1, Algorithm: "sha256", Hash: "h", Modified: "m"})
	if strings.Contains(string(b), "audited") {
		t.Fatalf("audited rendered on a node without it: %s", b)
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, FileName), []byte(`{"version": 1, "hash": "h", "audited": 2}`), 0o644)
	if m, err := Read(dir); err != nil || m.Audited == nil || *m.Audited != 2 {
		t.Fatalf("audited not read: %v %+v", err, m)
	}
}
