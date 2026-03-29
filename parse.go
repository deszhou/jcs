// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

// Recursive-descent JSON parse → canonical string builders.

package jcs

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
)

var jsonLiterals = []string{"true", "false", "null"}

func (p *parser) parseEntry() (string, error) {
	c, err := p.scan()
	if err != nil {
		return "", err
	}
	p.pos--

	switch c {
	case '{', '"', '[':
		return p.parseElement()
	default:
		// Root scalar: scan() left p.pos at the first content byte; trim only trailing
		// JSON whitespace (RFC 8259). Do not use string(p.data) — that breaks inputs
		// like "  true" or "true\n".
		raw := strings.TrimRight(string(p.data[p.pos:]), " \t\r\n")
		value, err := parseLiteral(raw)
		if err != nil {
			return "", err
		}
		p.pos = len(p.data)
		return value, nil
	}
}

func (p *parser) parseElement() (string, error) {
	c, err := p.scan()
	if err != nil {
		return "", err
	}
	switch c {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		raw, err := p.parseQuotedString()
		if err != nil {
			return "", err
		}
		return decorateString(raw), nil
	default:
		return p.parseSimpleType()
	}
}

func (p *parser) parseObject() (string, error) {
	var props []property
	var next bool

	for {
		c, err := p.peek()
		if err != nil {
			return "", err
		}
		if c == '}' {
			p.pos++
			break
		}

		if next {
			if err = p.scanFor(','); err != nil {
				return "", err
			}
		}
		next = true

		if err = p.scanFor('"'); err != nil {
			return "", err
		}
		name, err := p.parseQuotedString()
		if err != nil {
			return "", err
		}
		sortKey := toSortKey(name)

		if err = p.scanFor(':'); err != nil {
			return "", err
		}
		value, err := p.parseElement()
		if err != nil {
			return "", err
		}
		props = append(props, property{name, sortKey, value})
	}

	slices.SortFunc(props, func(a, b property) int {
		return slices.Compare(a.sortKey, b.sortKey)
	})

	for i := 1; i < len(props); i++ {
		if slices.Equal(props[i].sortKey, props[i-1].sortKey) {
			return "", fmt.Errorf("Duplicate key: %s", props[i].name)
		}
	}

	var out strings.Builder
	out.WriteByte('{')
	for i, prop := range props {
		if i > 0 {
			out.WriteByte(',')
		}
		out.WriteString(decorateString(prop.name))
		out.WriteByte(':')
		out.WriteString(prop.value)
	}
	out.WriteByte('}')
	return out.String(), nil
}

func (p *parser) parseArray() (string, error) {
	var out strings.Builder
	var next bool

	out.WriteByte('[')
	for {
		c, err := p.peek()
		if err != nil {
			return "", err
		}
		if c == ']' {
			p.pos++
			break
		}

		if next {
			if err = p.scanFor(','); err != nil {
				return "", err
			}
			out.WriteByte(',')
		} else {
			next = true
		}

		element, err := p.parseElement()
		if err != nil {
			return "", err
		}
		out.WriteString(element)
	}
	out.WriteByte(']')
	return out.String(), nil
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
			return out.String(), nil

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
					out.WriteRune(utf16.DecodeRune(first, second))
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

func (p *parser) parseSimpleType() (string, error) {
	var token strings.Builder
	p.pos--

	for {
		c, err := p.scan()
		if err != nil {
			return "", err
		}
		if c == ',' || c == ']' || c == '}' {
			p.pos--
			break
		}
		token.WriteByte(c)
	}

	if token.Len() == 0 {
		return "", errors.New("Missing argument")
	}
	return parseLiteral(token.String())
}

func parseLiteral(value string) (string, error) {
	if slices.Contains(jsonLiterals, value) {
		return value, nil
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", err
	}
	return NumberToJSON(f)
}
