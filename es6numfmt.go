// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"bytes"
	"errors"
	"math"
	"strconv"
)

const invalidPattern uint64 = 0x7ff0000000000000

// NumberToJSON converts numbers in IEEE-754 double precision into the
// format specified for JSON in EcmaScript Version 6 and forward.
// The core application for this is canonicalization per RFC 8785.
func NumberToJSON(value float64) (string, error) {
	var buf [32]byte
	number, err := appendNumber(buf[:0], value)
	if err != nil {
		return "null", err
	}
	return string(number), nil
}

func appendNumber(dst []byte, value float64) ([]byte, error) {
	bits := math.Float64bits(value)
	if bits&invalidPattern == invalidPattern {
		return nil, errors.New("Invalid JSON number: " + strconv.FormatUint(bits, 16))
	}
	if value == 0 {
		return append(dst, '0'), nil
	}

	format := byte('e')
	absolute := math.Abs(value)
	if absolute >= 1e-6 && absolute < 1e+21 {
		format = 'f'
	}

	start := len(dst)
	dst = strconv.AppendFloat(dst, value, format, -1, 64)
	if format == 'e' {
		exponent := start + bytes.IndexByte(dst[start:], 'e')
		if dst[exponent+2] == '0' {
			// Go pads single-digit exponents. JSON uses e-7, for example.
			return append(dst[:exponent+2], dst[exponent+3:]...), nil
		}
	}
	return dst, nil
}
