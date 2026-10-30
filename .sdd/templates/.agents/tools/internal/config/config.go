// Package config reads .sdd/.sdd-config.json: the framework version a project
// is on, the remote it came from, and the layout the tools read — where the
// product specs, the technical specs, and the code live. The file marks the
// repository root. A missing file means the default layout.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// File is the root-relative path of the config.
const File = ".sdd/.sdd-config.json"

// Paths is the layout: root-relative directories, forward slashes, no
// trailing slash.
type Paths struct {
	Specs     string `json:"specs"`     // product specs, one directory per perspective
	Src       string `json:"src"`       // code; every code node sits under it
	Technical string `json:"technical"` // technical specs; under Src or beside it
}

// Perspective is one perspective of the project and the template kind it was
// materialized from; From is empty for a kind the project added.
type Perspective struct {
	Name string `json:"name"`
	From string `json:"from,omitempty"`
}

// Config is the content of .sdd/.sdd-config.json.
type Config struct {
	Version      int           `json:"version"`
	Remote       string        `json:"remote,omitempty"`
	Product      string        `json:"product,omitempty"`
	Paths        Paths         `json:"paths"`
	Perspectives []Perspective `json:"perspectives,omitempty"`
	Skills       []string      `json:"skills,omitempty"`
}

// Default is the layout the framework ships.
func Default() Config {
	return Config{Paths: Paths{Specs: "specs", Src: "src", Technical: "src/specs"}}
}

// Load reads root/.sdd/.sdd-config.json. A missing file yields Default; an
// unknown field, a wrong type, or an invalid layout is an error. A path left
// empty takes its default.
func Load(root string) (Config, error) {
	c := Default()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(File)))
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("%s: invalid JSON: %v", File, err)
	}
	if err := c.Paths.validate(); err != nil {
		return c, fmt.Errorf("%s: %v", File, err)
	}
	return c, nil
}

// validate cleans each path and checks the layout: three root-relative
// directories; specs and src apart; technical apart from specs, and under
// src or apart from it.
func (p *Paths) validate() error {
	d := Default().Paths
	for _, f := range []struct {
		name string
		p    *string
		def  string
	}{{"specs", &p.Specs, d.Specs}, {"src", &p.Src, d.Src}, {"technical", &p.Technical, d.Technical}} {
		if *f.p == "" {
			*f.p = f.def
		}
		c := strings.TrimPrefix(path.Clean(strings.ReplaceAll(*f.p, "\\", "/")), "./")
		if c == "." || c == "" || strings.HasPrefix(c, "/") || c == ".." || strings.HasPrefix(c, "../") {
			return fmt.Errorf("paths.%s %q is not a directory inside the repository", f.name, *f.p)
		}
		*f.p = c
	}
	switch {
	case Under(p.Specs, p.Src) || Under(p.Src, p.Specs):
		return fmt.Errorf("paths.specs %q and paths.src %q overlap", p.Specs, p.Src)
	case Under(p.Technical, p.Specs) || Under(p.Specs, p.Technical):
		return fmt.Errorf("paths.technical %q and paths.specs %q overlap", p.Technical, p.Specs)
	case p.Technical == p.Src || Under(p.Src, p.Technical):
		return fmt.Errorf("paths.technical %q holds paths.src %q; technical specs sit under the code root or beside it", p.Technical, p.Src)
	}
	return nil
}

// Under reports whether root-relative p is dir or sits under it.
func Under(p, dir string) bool {
	return p == dir || strings.HasPrefix(p, dir+"/")
}
