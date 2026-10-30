// Package md reads the markdown shapes of specs/AGENTS.md: headings of
// level 1–3 outside code fences, the ### rules of a spec file, the targets of
// its links, and the anchor slugs of its headings.
package md

import (
	"regexp"
	"strings"
	"unicode"
)

// Section is a heading of level 1–3 with the lines up to the next one.
type Section struct {
	Level int
	Title string
	Lines []string
}

// Rule is a ### heading with its body: the lines after the title up to the
// next heading of level 1–3, leading and trailing blank lines removed.
type Rule struct {
	Title string
	Body  string
}

var (
	fenceRe   = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
	headingRe = regexp.MustCompile(`^(#{1,3})\s+(.+?)\s*$`)
	atxRe     = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))?(?:[ \t]+#+)?[ \t]*$`)
	setextRe  = regexp.MustCompile(`^ {0,3}(?:=+|-+)[ \t]*$`)
	inlineRe  = regexp.MustCompile(`(?:^|[^\\])\]\(\s*(<[^>]*>|[^)\s]+)(?:\s+(?:"[^"]*"|'[^']*'|\([^)]*\)))?\s*\)`)
	refDefRe  = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:\s*(<[^>]*>|\S+)`)
)

// Sections returns the headings of level 1–3 outside code fences, in order. A
// fence opens with three or more backticks or tildes indented at most three
// spaces (CommonMark) and closes with the same marker.
func Sections(text string) []Section {
	var out []Section
	var fence byte
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if m := fenceRe.FindStringSubmatch(line); m != nil {
			switch c := m[1][0]; {
			case fence == 0:
				fence = c
			case fence == c:
				fence = 0
			}
		} else if fence == 0 {
			if m := headingRe.FindStringSubmatch(line); m != nil {
				out = append(out, Section{Level: len(m[1]), Title: m[2]})
				continue
			}
		}
		if len(out) > 0 {
			out[len(out)-1].Lines = append(out[len(out)-1].Lines, line)
		}
	}
	return out
}

// Rules returns the ### rules of a spec file in file order.
func Rules(text string) []Rule {
	var out []Rule
	for _, s := range Sections(text) {
		if s.Level != 3 {
			continue
		}
		lines := s.Lines
		for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
			lines = lines[1:]
		}
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		out = append(out, Rule{Title: s.Title, Body: strings.Join(lines, "\n")})
	}
	return out
}

// outsideFences returns the lines of text outside code fences, the fence
// lines dropped.
func outsideFences(text string) []string {
	var out []string
	var fence byte
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if m := fenceRe.FindStringSubmatch(line); m != nil {
			switch c := m[1][0]; {
			case fence == 0:
				fence = c
			case fence == c:
				fence = 0
			}
			out = append(out, "")
			continue
		}
		if fence == 0 {
			out = append(out, line)
		}
	}
	return out
}

// Links returns the targets of the links outside code fences and code spans,
// in order: inline links and images, `[text](target)` with an optional title
// and `<target>` brackets removed, and reference definitions, `[ref]: target`.
// An escaped `\]` opens no link.
func Links(text string) []string {
	var out []string
	unbracket := func(t string) string { return strings.TrimSuffix(strings.TrimPrefix(t, "<"), ">") }
	for _, line := range outsideFences(text) {
		line = stripCode(line)
		if m := refDefRe.FindStringSubmatch(line); m != nil {
			out = append(out, unbracket(m[1]))
			continue
		}
		for _, m := range inlineRe.FindAllStringSubmatch(line, -1) {
			out = append(out, unbracket(m[1]))
		}
	}
	return out
}

// stripCode removes the code spans of a line: a run of backticks up to the
// next run of the same length. A run with no match is literal text.
func stripCode(line string) string {
	var b strings.Builder
	for i := 0; i < len(line); {
		if line[i] != '`' {
			b.WriteByte(line[i])
			i++
			continue
		}
		n := 0
		for i+n < len(line) && line[i+n] == '`' {
			n++
		}
		end := -1
		for j := i + n; j < len(line); {
			if line[j] != '`' {
				j++
				continue
			}
			k := 0
			for j+k < len(line) && line[j+k] == '`' {
				k++
			}
			if k == n {
				end = j + k
				break
			}
			j += k
		}
		if end < 0 {
			b.WriteString(line[i : i+n])
			i += n
			continue
		}
		i = end
	}
	return b.String()
}

// Anchors returns the slugs of every heading outside code fences — ATX of any
// level, indented up to three spaces, closing hashes dropped, and setext — in
// order.
func Anchors(text string) []string {
	var out []string
	prev := ""
	for _, line := range outsideFences(text) {
		switch m := atxRe.FindStringSubmatch(line); {
		case m != nil:
			out = append(out, Slug(m[2]))
			line = ""
		case setextRe.MatchString(line) && strings.TrimSpace(prev) != "":
			out = append(out, Slug(strings.TrimSpace(prev)))
			line = ""
		}
		prev = line
	}
	return out
}

// Slug is the #anchor of a title, as Anchors of specs/AGENTS.md defines it:
// lowercase; every rune that is not a letter, a number, `_`, a space, or `-`
// stripped, accented letters kept; each space becomes `-`.
func Slug(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '_' || r == '-' || unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}
