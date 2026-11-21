// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

// Recursive-descent JSON parsing and serialization into a shared buffer.

package jcs

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

var jsonLiterals = []string{"true", "false", "null"}

func (p *parser) parseEntry() error {
	c, err := p.scan()
	if err != nil {
		return err
	}
	p.pos--
	switch c {
	case '{', '"', '[':
		return p.parseElement()
	default:
		// scan already skipped leading whitespace. Preserve whitespace inside
		// the root token and trim only trailing JSON whitespace.
		raw := strings.TrimRight(string(p.data[p.pos:]), " \t\r\n")
		p.pos = len(p.data)
		return p.appendLiteral(raw)
	}
}

func (p *parser) parseElement() error {
	c, err := p.scan()
	if err != nil {
		return err
	}
	switch c {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		_, err := p.appendString()
		return err
	default:
		return p.parseSimpleType()
	}
}

// appendString appends a canonical string and returns its decoded contents
// for use as an object key. Unescaped strings can be copied from the input.
func (p *parser) appendString() ([]byte, error) {
	start := p.pos
	for i := start; i < len(p.data); i++ {
		c := p.data[i]
		if c == '"' {
			raw := p.data[start:i]
			if !utf8.Valid(raw) {
				return nil, errors.New("Invalid UTF-8 in string literal")
			}
			p.pos = i + 1
			p.out = append(p.out, p.data[start-1:p.pos]...)
			return raw, nil
		}
		if c == '\\' || c < ' ' {
			break
		}
	}
	raw, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}
	p.out = appendEscaped(p.out, raw)
	return []byte(raw), nil
}

func (p *parser) parseProperty() (property, error) {
	if err := p.scanFor('"'); err != nil {
		return property{}, err
	}
	start := len(p.out)
	name, err := p.appendString()
	if err != nil {
		return property{}, err
	}
	if err := p.scanFor(':'); err != nil {
		return property{}, err
	}
	p.out = append(p.out, ':')
	if err := p.parseElement(); err != nil {
		return property{}, err
	}
	return property{name: name, start: start, end: len(p.out)}, nil
}

func (p *parser) parseObject() error {
	start := len(p.out)
	p.out = append(p.out, '{')
	c, err := p.peek()
	if err != nil {
		return err
	}
	if c == '}' {
		p.pos++
		p.out = append(p.out, '}')
		return nil
	}
	first, err := p.parseProperty()
	if err != nil {
		return err
	}
	c, err = p.scan()
	if err != nil {
		return err
	}
	if c == '}' {
		p.out = append(p.out, '}')
		return nil
	}
	// Nested objects use the same scratch slice and restore its length
	// before returning. A single-member object needs no scratch space.
	base := len(p.props)
	p.props = append(p.props, first)
	for c == ',' {
		p.out = append(p.out, ',')
		prop, err := p.parseProperty()
		if err != nil {
			return err
		}
		p.props = append(p.props, prop)
		c, err = p.scan()
		if err != nil {
			return err
		}
	}
	if c != '}' {
		return fmt.Errorf("Expected , but got %s", string(c))
	}
	p.out = append(p.out, '}')
	props := p.props[base:]
	// Strictly increasing keys are both ordered and unique.
	ordered := true
	for i := 1; i < len(props); i++ {
		if compareNames(props[i-1].name, props[i].name) >= 0 {
			ordered = false
			break
		}
	}
	if !ordered {
		slices.SortFunc(props, func(a, b property) int { return compareNames(a.name, b.name) })
		members := make([]span, len(props))
		for i, prop := range props {
			if i > 0 && compareNames(props[i-1].name, prop.name) == 0 {
				return fmt.Errorf("Duplicate key: %s", prop.name)
			}
			members[i] = span{prop.start, prop.end}
		}
		p.reorders = append(p.reorders, reorder{start, len(p.out), members})
	}
	p.props = p.props[:base]
	return nil
}

// Compare decoded names by UTF-16 order without allocating sort keys.
func compareNames(a, b []byte) int {
	for len(a) != 0 && len(b) != 0 {
		if min(a[0], b[0]) < utf8.RuneSelf {
			if a[0] != b[0] {
				return int(a[0]) - int(b[0])
			}
			a, b = a[1:], b[1:]
			continue
		}
		ar, as := utf8.DecodeRune(a)
		br, bs := utf8.DecodeRune(b)
		if ar != br {
			return utf16Order(ar) - utf16Order(br)
		}
		a, b = a[as:], b[bs:]
	}
	return len(a) - len(b)
}

// All supplementary code points start with a UTF-16 high surrogate, so
// they sort between the BMP below the surrogates and the BMP above them.
func utf16Order(r rune) int {
	if 0xe000 <= r && r < 0x10000 {
		return int(r) + utf8.MaxRune + 1
	}
	return int(r)
}

func (p *parser) parseArray() error {
	p.out = append(p.out, '[')
	c, err := p.peek()
	if err != nil {
		return err
	}
	if c != ']' {
		for {
			if err := p.parseElement(); err != nil {
				return err
			}
			c, err = p.peek()
			if err != nil {
				return err
			}
			if c == ']' {
				break
			}
			if err = p.scanFor(','); err != nil {
				return err
			}
			p.out = append(p.out, ',')
		}
	}
	p.pos++
	p.out = append(p.out, ']')
	return nil
}

func (p *parser) parseQuotedString() (string, error) {
	var out strings.Builder

	for {
		if p.pos >= len(p.data) {
			return "", errors.New("Unexpected EOF reached")
		}
		c := p.data[p.pos]
		p.pos++

		switch {
		case c == '"':
			// RFC 8785 §3.2.4 defines canonical output as valid UTF-8. Raw
			// bytes copied below may form an invalid sequence (e.g. 0xff);
			// reject so such input cannot pass through unchanged.
			s := out.String()
			if !utf8.ValidString(s) {
				return "", errors.New("Invalid UTF-8 in string literal")
			}
			return s, nil

		case c < ' ':
			return "", errors.New("Unterminated string literal")

		case c == '\\':
			esc, err := p.nextChar()
			if err != nil {
				return "", err
			}
			switch esc {
			case 'u':
				first, err := p.getUEscape()
				if err != nil {
					return "", err
				}
				if utf16.IsSurrogate(first) {
					hi, err := p.nextChar()
					if err != nil {
						return "", err
					}
					lo, err := p.nextChar()
					if err != nil {
						return "", err
					}
					if hi != '\\' || lo != 'u' {
						return "", errors.New("Missing surrogate")
					}
					second, err := p.getUEscape()
					if err != nil {
						return "", err
					}
					// DecodeRune returns U+FFFD when first/second are not a
					// valid high+low surrogate pair (e.g. a reversed pair or a
					// lone low surrogate). RFC 8785 §3.2.2.2 requires a fatal
					// error, so reject rather than emit the replacement rune.
					combined := utf16.DecodeRune(first, second)
					if combined == unicode.ReplacementChar {
						return "", errors.New("Invalid surrogate pair")
					}
					out.WriteRune(combined)
				} else {
					out.WriteRune(first)
				}
			case '/':
				out.WriteByte('/')
			default:
				if !isValidEscape[esc] {
					return "", fmt.Errorf("Unexpected escape: \\%s", string(esc))
				}
				out.WriteByte(binaryForEscape[esc])
			}

		default:
			out.WriteByte(c)
		}
	}
}

func (p *parser) parseSimpleType() error {
	var token strings.Builder
	p.pos--

	for {
		c, err := p.scan()
		if err != nil {
			return err
		}
		if c == ',' || c == ']' || c == '}' {
			p.pos--
			break
		}
		token.WriteByte(c)
	}

	if token.Len() == 0 {
		return errors.New("Missing argument")
	}
	return p.appendLiteral(token.String())
}

func (p *parser) appendLiteral(value string) error {
	if slices.Contains(jsonLiterals, value) {
		p.out = append(p.out, value...)
		return nil
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	p.out, err = appendNumber(p.out, f)
	return err
}
