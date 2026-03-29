// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

package jcs

import "errors"

// Transform converts raw JSON into its RFC 8785 canonical UTF-8 form.
// The input must be valid JSON; non-ASCII bytes are only allowed inside strings.
func Transform(jsonData []byte) ([]byte, error) {
	if jsonData == nil {
		return nil, errors.New("No JSON data provided")
	}

	p := &parser{data: jsonData}
	out, err := p.parseEntry()
	if err != nil {
		return nil, err
	}

	for p.pos < len(p.data) {
		if !isWhiteSpace(p.data[p.pos]) {
			return nil, errors.New("Improperly terminated JSON object")
		}
		p.pos++
	}
	return []byte(out), nil
}
