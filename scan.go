// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

// Lexer: UTF-8 JSON byte stream position and low-level scanning.

package jcs

import (
	"errors"
	"fmt"
	"strconv"
	"unicode/utf16"
)

// property is one object entry with a precomputed UTF-16 sort key (RFC 8785 §3.2.3).
type property struct {
	name    string
	sortKey []uint16
	value   string
}

// parser holds mutable state for one canonicalization pass.
type parser struct {
	data []byte
	pos  int
}

func isWhiteSpace(c byte) bool {
	return c == 0x20 || c == 0x0a || c == 0x0d || c == 0x09
}

// toSortKey maps a property name to UTF-16 code units for lexicographic compare.
func toSortKey(s string) []uint16 {
	for i := range len(s) {
		if s[i] >= 0x80 {
			return utf16.Encode([]rune(s))
		}
	}
	key := make([]uint16, len(s))
	for i := range len(s) {
		key[i] = uint16(s[i])
	}
	return key
}

func (p *parser) nextChar() (byte, error) {
	if p.pos < len(p.data) {
		c := p.data[p.pos]
		if c > 0x7f {
			return 0, errors.New("Unexpected non-ASCII character")
		}
		p.pos++
		return c, nil
	}
	return 0, errors.New("Unexpected EOF reached")
}

func (p *parser) scan() (byte, error) {
	for {
		c, err := p.nextChar()
		if err != nil {
			return 0, err
		}
		if !isWhiteSpace(c) {
			return c, nil
		}
	}
}

func (p *parser) scanFor(expected byte) error {
	c, err := p.scan()
	if err != nil {
		return err
	}
	if c != expected {
		return fmt.Errorf("Expected %s but got %s", string(expected), string(c))
	}
	return nil
}

func (p *parser) peek() (byte, error) {
	c, err := p.scan()
	if err != nil {
		return 0, err
	}
	p.pos--
	return c, nil
}

func (p *parser) getUEscape() (rune, error) {
	start := p.pos
	for range 4 {
		if _, err := p.nextChar(); err != nil {
			return 0, err
		}
	}
	u16, err := strconv.ParseUint(string(p.data[start:p.pos]), 16, 64)
	if err != nil {
		return 0, err
	}
	return rune(u16), nil
}
