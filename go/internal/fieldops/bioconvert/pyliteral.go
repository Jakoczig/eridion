// Package bioconvert turns the pinned EDMC-BioScan / ExploData data modules
// into ERIDION's native rule file. The upstream files are parsed as Python
// literal data (dicts, lists, tuples, strings, numbers, booleans); no Python
// is executed, imported or required.
package bioconvert

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Tuple marks a Python tuple literal, which upstream gives different meaning
// from a list in star predicates.
type Tuple []any

// Dict preserves the source key order of a Python dict literal.
type Dict struct {
	Keys   []string
	Values map[string]any
}

// Get returns the value for key.
func (d *Dict) Get(key string) (any, bool) {
	v, ok := d.Values[key]
	return v, ok
}

type parser struct {
	src   string
	pos   int
	depth int
}

const maxDepth = 64

// Assignment parses the literal assigned to name at module level
// (`name = <literal>` or `name: <annotation> = <literal>`; annotations in
// these modules never contain '=').
func Assignment(src, name string) (any, error) {
	for _, line := range lineStarts(src) {
		rest := src[line:]
		if !strings.HasPrefix(rest, name) || len(rest) == len(name) || (rest[len(name)] != ' ' && rest[len(name)] != ':' && rest[len(name)] != '=') {
			continue
		}
		eq := strings.IndexByte(rest, '=')
		if eq < 0 {
			break
		}
		p := &parser{src: src, pos: line + eq + 1, depth: 0}
		v, err := p.value()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		// Only a comment may follow the literal on its line: a trailing
		// expression (`{...} | extra`) would silently change the data.
		tail := src[p.pos:]
		if nl := strings.IndexByte(tail, '\n'); nl >= 0 {
			tail = tail[:nl]
		}
		if t := strings.TrimSpace(tail); t != "" && !strings.HasPrefix(t, "#") {
			return nil, fmt.Errorf("%s: unexpected %q after the literal", name, t)
		}
		// A later module-level rebinding or mutation of the name is refused.
		for _, later := range lineStarts(src) {
			if later <= line || later >= len(src) {
				continue
			}
			l := src[later:]
			if strings.HasPrefix(l, name) && len(l) > len(name) && strings.ContainsRune(" :=.[|+", rune(l[len(name)])) {
				return nil, fmt.Errorf("%s: rebound or modified later in the module", name)
			}
		}
		return v, nil
	}
	return nil, fmt.Errorf("assignment %q not found", name)
}

func lineStarts(src string) []int {
	out := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			out = append(out, i+1)
		}
	}
	return out
}

func (p *parser) skip() {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			p.pos++
		case c == '#':
			for p.pos < len(p.src) && p.src[p.pos] != '\n' {
				p.pos++
			}
		case c == '\\' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '\n':
			p.pos += 2
		default:
			return
		}
	}
}

func (p *parser) fail(msg string) error {
	line := strings.Count(p.src[:min(p.pos, len(p.src))], "\n") + 1
	return fmt.Errorf("line %d: %s", line, msg)
}

func (p *parser) value() (any, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > maxDepth {
		return nil, p.fail("nesting too deep")
	}
	p.skip()
	if p.pos >= len(p.src) {
		return nil, p.fail("unexpected end")
	}
	switch c := p.src[p.pos]; {
	case c == '{':
		return p.dict()
	case c == '[':
		p.pos++
		items, err := p.items(']')
		return items, err
	case c == '(':
		p.pos++
		items, err := p.items(')')
		return Tuple(items), err
	case c == '\'' || c == '"':
		return p.str()
	case c == '-' || c == '+' || c == '.' || (c >= '0' && c <= '9'):
		return p.number()
	default:
		for _, w := range []struct {
			word string
			v    any
		}{{"True", true}, {"False", false}, {"None", nil}} {
			if strings.HasPrefix(p.src[p.pos:], w.word) {
				p.pos += len(w.word)
				return w.v, nil
			}
		}
		return nil, p.fail(fmt.Sprintf("unsupported token %q (only literal data is accepted)", p.src[p.pos:min(p.pos+20, len(p.src))]))
	}
}

func (p *parser) items(end byte) ([]any, error) {
	out := []any{}
	for {
		p.skip()
		if p.pos < len(p.src) && p.src[p.pos] == end {
			p.pos++
			return out, nil
		}
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		p.skip()
		if p.pos < len(p.src) && p.src[p.pos] == ',' {
			p.pos++
			continue
		}
		if p.pos < len(p.src) && p.src[p.pos] == end {
			p.pos++
			return out, nil
		}
		return nil, p.fail("expected ',' or closing bracket")
	}
}

func (p *parser) dict() (*Dict, error) {
	p.pos++
	d := &Dict{Values: map[string]any{}}
	for {
		p.skip()
		if p.pos < len(p.src) && p.src[p.pos] == '}' {
			p.pos++
			return d, nil
		}
		k, err := p.value()
		if err != nil {
			return nil, err
		}
		key, ok := k.(string)
		if !ok {
			return nil, p.fail("non-string dict key")
		}
		p.skip()
		if p.pos >= len(p.src) || p.src[p.pos] != ':' {
			return nil, p.fail("expected ':'")
		}
		p.pos++
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		if _, dup := d.Values[key]; dup {
			return nil, p.fail("duplicate key " + key)
		}
		d.Keys = append(d.Keys, key)
		d.Values[key] = v
		p.skip()
		if p.pos < len(p.src) && p.src[p.pos] == ',' {
			p.pos++
			continue
		}
		if p.pos < len(p.src) && p.src[p.pos] == '}' {
			p.pos++
			return d, nil
		}
		return nil, p.fail("expected ',' or '}'")
	}
}

func (p *parser) str() (string, error) {
	quote := p.src[p.pos]
	p.pos++
	var b strings.Builder
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == quote:
			p.pos++
			// Adjacent literals concatenate, as in Python.
			save := p.pos
			p.skip()
			if p.pos < len(p.src) && (p.src[p.pos] == '\'' || p.src[p.pos] == '"') {
				next, err := p.str()
				return b.String() + next, err
			}
			p.pos = save
			return b.String(), nil
		case c == '\\' && p.pos+1 < len(p.src):
			n := p.src[p.pos+1]
			switch n {
			case '\\', '\'', '"':
				b.WriteByte(n)
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				return "", p.fail("unsupported escape")
			}
			p.pos += 2
		case c == '\n':
			return "", p.fail("unterminated string")
		default:
			b.WriteByte(c)
			p.pos++
		}
	}
	return "", errors.New("unterminated string")
}

func (p *parser) number() (any, error) {
	start := p.pos
	for p.pos < len(p.src) && strings.IndexByte("+-0123456789._eE", p.src[p.pos]) >= 0 {
		p.pos++
	}
	text := strings.ReplaceAll(p.src[start:p.pos], "_", "")
	if !strings.ContainsAny(text, ".eE") {
		if n, err := strconv.ParseInt(text, 10, 64); err == nil {
			return n, nil
		}
	}
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, p.fail("bad number " + text)
	}
	return f, nil
}
