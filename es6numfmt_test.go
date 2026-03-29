// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

const testFile = "/home/test/es6testfile100m.txt"

const invalidNumber = "null"

func TestNumberToJSON(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ieeeHex  string
		expected string
		wantErr  bool
	}{
		{"4340000000000001", "9007199254740994", false},
		{"4340000000000002", "9007199254740996", false},
		{"444b1ae4d6e2ef50", "1e+21", false},
		{"3eb0c6f7a0b5ed8d", "0.000001", false},
		{"3eb0c6f7a0b5ed8c", "9.999999999999997e-7", false},
		{"8000000000000000", "0", false},
		{"7fffffffffffffff", invalidNumber, true},
		{"7ff0000000000000", invalidNumber, true},
		{"fff0000000000000", invalidNumber, true},
	}

	if _, err := os.Stat(testFile); err == nil {
		file, err := os.Open(testFile)
		if err != nil {
			t.Fatalf("open large test file: %v", err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		lineCount := 0
		for scanner.Scan() {
			lineCount++
			line := scanner.Text()
			parts := strings.Split(line, ",")
			if len(parts) != 2 {
				t.Fatalf("line %d: want hex,expected", lineCount)
			}
			hex, expected := parts[0], parts[1]
			t.Run(expected, func(t *testing.T) {
				runNumberCase(t, hex, expected, false)
			})
		}
		if err := scanner.Err(); err != nil {
			t.Fatalf("scan: %v", err)
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat test file: %v", err)
	}

	for _, tc := range cases {
		name := tc.ieeeHex
		if !tc.wantErr {
			name = tc.expected + "_" + tc.ieeeHex
		}
		t.Run(name, func(t *testing.T) {
			runNumberCase(t, tc.ieeeHex, tc.expected, tc.wantErr)
		})
	}
}

func runNumberCase(t *testing.T, ieeeHex, expected string, wantErr bool) {
	t.Helper()
	hex := ieeeHex
	for len(hex) < 16 {
		hex = "0" + hex
	}
	u, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		t.Fatalf("parse hex %q: %v", ieeeHex, err)
	}
	f := math.Float64frombits(u)
	got, err := NumberToJSON(f)
	if wantErr {
		if err == nil {
			t.Fatalf("hex %s: want error, got %q", ieeeHex, got)
		}
		return
	}
	if err != nil {
		t.Fatalf("hex %s: %v", ieeeHex, err)
	}
	if got != expected {
		t.Fatalf("hex %s: got %q want %q", ieeeHex, got, expected)
	}
}
