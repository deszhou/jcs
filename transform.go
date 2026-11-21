// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"errors"
	"slices"
)

// Transform converts raw JSON into its RFC 8785 canonical UTF-8 form.
// The input must be valid JSON; non-ASCII bytes are only allowed inside strings.
func Transform(jsonData []byte) ([]byte, error) {
	if jsonData == nil {
		return nil, errors.New("No JSON data provided")
	}

	// Bound the initial allocation since an input can be mostly whitespace.
	const maxInitialCapacity = 64 << 10
	p := &parser{data: jsonData, out: make([]byte, 0, min(len(jsonData), maxInitialCapacity))}
	if err := p.parseEntry(); err != nil {
		return nil, err
	}

	for p.pos < len(p.data) {
		if !isWhiteSpace(p.data[p.pos]) {
			return nil, errors.New("Improperly terminated JSON object")
		}
		p.pos++
	}
	if len(p.reorders) == 0 {
		return p.out, nil
	}
	slices.SortFunc(p.reorders, func(a, b reorder) int { return a.start - b.start })
	return p.writeRange(make([]byte, 0, len(p.out)), span{0, len(p.out)}), nil
}

// A span describes a half-open byte range in parser.out.
type span struct{ start, end int }

// A reorder describes an object and its member spans in canonical key order.
// Member spans omit the commas between members.
type reorder struct {
	start, end int
	members    []span
}

// writeRange copies runs of bytes verbatim until it reaches an object that
// needs reordering. Its members may contain further reorders of their own.
func (p *parser) writeRange(dst []byte, region span) []byte {
	i, _ := slices.BinarySearchFunc(p.reorders, region.start, compareReorderStart)
	for i < len(p.reorders) && p.reorders[i].start < region.end {
		r := p.reorders[i]
		dst = append(dst, p.out[region.start:r.start+1]...)
		for j, member := range r.members {
			if j != 0 {
				dst = append(dst, ',')
			}
			dst = p.writeRange(dst, member)
		}
		dst = append(dst, '}')
		region.start = r.end
		// Skip nested objects already handled while writing these members.
		skip, _ := slices.BinarySearchFunc(p.reorders[i+1:], r.end, compareReorderStart)
		i += 1 + skip
	}
	return append(dst, p.out[region.start:region.end]...)
}

func compareReorderStart(r reorder, start int) int {
	return r.start - start
}
