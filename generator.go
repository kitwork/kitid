package id

import (
	"fmt"
	"time"
)

// Generator encapsulates the logic and state for ID generation.
type Generator struct {
	epoch      int64
	charset    []byte
	err        error
	repeat     bool
	increasing bool
}

// New returns a new Generator.
// If no arguments are provided, it uses the default epoch (2025-01-01).
// If a time argument is provided, it uses that as the epoch.
func New(t ...time.Time) *Generator {
	if len(t) == 0 {
		return &Generator{epoch: since.UnixMilli()}
	}
	if len(t) > 1 {
		return &Generator{err: fmt.Errorf("id: too many epochs provided")}
	}
	ts := t[0].UnixMilli()
	if ts < 0 {
		return &Generator{err: fmt.Errorf("id: epoch must be non-negative")}
	}
	if ts > time.Now().UnixMilli() {
		return &Generator{err: fmt.Errorf("id: epoch must be in the past")}
	}
	return &Generator{epoch: ts}
}

// Charset sets the charset for the generator based on length.
func (g *Generator) Charset(n int) *Generator {
	switch n {
	case 10:
		g.charset = charset10
	case 26:
		g.charset = charset26
	case 36:
		g.charset = charset36
	case 58:
		g.charset = charset58
	case 62:
		g.charset = charset62
	default:
		g.err = fmt.Errorf("id: unsupported charset length %d", n)
	}
	return g
}

// Repeat configures the generator to allow repeated characters.
func (g *Generator) Repeat(repeat bool) *Generator {
	g.repeat = repeat
	return g
}

// Increasing configures the generator to produce time-based sortable IDs.
func (g *Generator) Increasing(increasing bool) *Generator {
	g.increasing = increasing
	return g
}

// Custom sets a custom charset.
func (g *Generator) Custom(chars string) *Generator {
	if len(chars) < 2 {
		g.err = fmt.Errorf("id: charset length must be at least 2")
		return g
	}
	g.charset = []byte(chars)
	return g
}
