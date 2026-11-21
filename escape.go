// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

package jcs

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

// appendEscaped writes a decoded string as a canonical JSON string.
func appendEscaped(out []byte, raw string) []byte {
	out = append(out, '"')
	for i := range len(raw) {
		c := raw[i]
		switch {
		case !needsEscaping[c]:
			out = append(out, c)
		case escapedChar[c] != 0:
			out = append(out, '\\', escapedChar[c])
		default:
			out = append(out, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
		}
	}
	return append(out, '"')
}
