// Package md reads the markdown shapes of perspectives/AGENTS.md: headings of
// level 1–3 outside code fences, and the ### rules of a spec file.
package md

import (
	"regexp"
	"strings"
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
