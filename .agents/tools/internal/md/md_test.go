package md

import (
	"reflect"
	"testing"
)

func TestSectionsSkipFencesAndDeepHeadings(t *testing.T) {
	text := "# A\n\nintro\n\n```md\n## not a heading\n```\n\n## B\n\n### C\n\nbody\n\n#### D\n\nmore\n~~~\n### hidden\n~~~\n### E\n"
	var titles []string
	for _, s := range Sections(text) {
		titles = append(titles, s.Title)
	}
	if want := []string{"A", "B", "C", "E"}; !reflect.DeepEqual(titles, want) {
		t.Fatalf("titles = %v, want %v", titles, want)
	}
	c := Sections(text)[2]
	if c.Level != 3 || !contains(c.Lines, "#### D") || !contains(c.Lines, "### hidden") {
		t.Fatalf("section C = %+v", c)
	}
}

func TestFenceClosesOnlyWithSameMarker(t *testing.T) {
	text := "```\n~~~\n## inside\n```\n## outside\n"
	s := Sections(text)
	if len(s) != 1 || s[0].Title != "outside" {
		t.Fatalf("sections = %+v", s)
	}
}

func TestRulesTrimBlankLines(t *testing.T) {
	rules := Rules("## S\n\n### R1\n\nbody one\n\n\n### R2\nbody two\n\n#### Sub\n\nkept\n\n## T\n\n### R3\r\nwindows\r\n")
	want := []Rule{
		{"R1", "body one"},
		{"R2", "body two\n\n#### Sub\n\nkept"},
		{"R3", "windows"},
	}
	if !reflect.DeepEqual(rules, want) {
		t.Fatalf("rules = %#v, want %#v", rules, want)
	}
}

func contains(lines []string, s string) bool {
	for _, l := range lines {
		if l == s {
			return true
		}
	}
	return false
}

func TestFenceIndentationFollowsCommonMark(t *testing.T) {
	text := "### A\n\n   ```\n   ### inside three-space fence\n   ```\n### B\n\n    ```\n### C after an indented code block, not a fence\n"
	var titles []string
	for _, s := range Sections(text) {
		titles = append(titles, s.Title)
	}
	if !reflect.DeepEqual(titles, []string{"A", "B", "C after an indented code block, not a fence"}) {
		t.Fatalf("titles = %v", titles)
	}
}

func TestLinksSkipFencesAndCodeSpans(t *testing.T) {
	text := "Serves [A](../a/spec.md#rule-a) and [B](b.md \"title\").\r\n" +
		"`[not](code.md)` ![img](pic.png) ``[x](`y`.md)`` [C](<c d.md#h>) [D](d.md 'single') [E](e.md (paren))\n" +
		"```\n[fenced](f.md)\n```\n" +
		"[web](https://example.com) [self](#own) \\[esc\\](escaped.md) a ` lone [F](f.md)\n" +
		"[ref]: ../r/spec.md#ref-rule\n" +
		"[^1]: Balances are shown.\n[Option]: describes x\n\\[not a link](no.md) [ok \\[x\\]](yes.md)\n" +
		"[a](adj-a.md)[b](adj-b.md) [the [Single use] rule](nested.md) [![img](in.png)](out.md) \\\\[e](after-backslash.md)\n"
	want := []string{"../a/spec.md#rule-a", "b.md", "pic.png", "c d.md#h", "d.md", "e.md", "https://example.com", "#own", "f.md", "../r/spec.md#ref-rule", "yes.md", "adj-a.md", "adj-b.md", "nested.md", "in.png", "out.md", "after-backslash.md"}
	if got := Links(text); !reflect.DeepEqual(got, want) {
		t.Fatalf("Links = %q, want %q", got, want)
	}
}

func TestAnchorsOfEveryHeadingForm(t *testing.T) {
	text := "# Title\n\n## Foo ##\n\n  ### Indented\n\n#### Deep\n\nSetext heading\n--------------\n\n---\n\n```\n## fenced\n```\n## Foo\n### Foo\n#### Notes\n#### Notes 1\n#### Notes\n"
	want := []string{"title", "foo", "indented", "deep", "setext-heading", "foo-1", "foo-2", "notes", "notes-1", "notes-2"}
	if got := Anchors(text); !reflect.DeepEqual(got, want) {
		t.Fatalf("Anchors = %q, want %q", got, want)
	}
}

func TestSlugFollowsAnchors(t *testing.T) {
	for title, want := range map[string]string{
		"Invalid credentials with generic message": "invalid-credentials-with-generic-message",
		"Multi-tab session — product rules":        "multi-tab-session--product-rules",
		"Sessão expirada":                          "sessão-expirada",
		"SQLite through node:sqlite":               "sqlite-through-nodesqlite",
		"Node.js 26 and `TypeScript` 5":            "nodejs-26-and-typescript-5",
		"snake_case codes":                         "snake_case-codes",
	} {
		if got := Slug(title); got != want {
			t.Errorf("Slug(%q) = %q, want %q", title, got, want)
		}
	}
}
