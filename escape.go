// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

package jcs

import "strings"

// JSON standard escape pairs: ASCII representation ↔ binary value.
var (
	asciiEscapes  = []byte{'\\', '"', 'b', 'f', 'n', 'r', 't'}
	binaryEscapes = []byte{'\\', '"', '\b', '\f', '\n', '\r', '\t'}
)

const hexDigits = "0123456789abcdef"

// Pre-computed 256-entry lookup tables for O(1) escape processing.
var (
	// escapedChar[b] is the ASCII escape character for binary byte b.
	// Zero means the byte is not a standard named escape (use \uXXXX or write as-is).
	escapedChar [256]byte
	// needsEscaping[b] is true when byte b must be escaped in JSON output.
	needsEscaping [256]bool
	// binaryForEscape[c] maps an ASCII escape character back to its binary value.
	binaryForEscape [256]byte
	// isValidEscape[c] is true when c is a valid single-character JSON escape.
	isValidEscape [256]bool
)

func init() {
	for i, bin := range binaryEscapes {
		asc := asciiEscapes[i]
		escapedChar[bin] = asc
		needsEscaping[bin] = true
		binaryForEscape[asc] = bin
		isValidEscape[asc] = true
	}
	// All remaining ASCII control characters need \u00XX escaping.
	for c := range byte(0x20) {
		needsEscaping[c] = true
	}
}

// decorateString wraps rawUTF8 in JSON double-quotes, escaping characters as
// required by RFC 8785. It is a pure function that requires no parser state.
func decorateString(rawUTF8 string) string {
	var b strings.Builder
	b.Grow(len(rawUTF8) + 2)
	b.WriteByte('"')
	for i := range len(rawUTF8) {
		c := rawUTF8[i]
		if !needsEscaping[c] {
			b.WriteByte(c)
			continue
		}
		if esc := escapedChar[c]; esc != 0 {
			b.WriteByte('\\')
			b.WriteByte(esc)
		} else {
			// ASCII control character: emit \u00XX without allocating via fmt.
			b.WriteByte('\\')
			b.WriteByte('u')
			b.WriteByte('0')
			b.WriteByte('0')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0xf])
		}
	}
	b.WriteByte('"')
	return b.String()
}
