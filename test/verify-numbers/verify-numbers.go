// Copyright 2026 deszhou.
// Portions copyright 2006-2019 WebPKI.org (cyberphone/json-canonicalization).
//
// SPDX-License-Identifier: Apache-2.0

// verify-numbers tests NumberToJSON against a set of discrete IEEE-754 values
// and, when available, against the 100-million-value test suite published at
// https://github.com/cyberphone/json-canonicalization.
//
// The large test file is optional. Pass its path via -testfile or the
// ES6_TESTFILE environment variable:
//
//	go run ./test/verify-numbers -testfile /path/to/es6testfile100m.txt
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/deszhou/jcs"
)

var testFile = flag.String("testfile", os.Getenv("ES6_TESTFILE"),
	"optional path to es6testfile100m.txt (overrides ES6_TESTFILE env var)")

const invalidNumber = "null"

var conversionErrors int

func verify(ieeeHex, expected string) {
	for len(ieeeHex) < 16 {
		ieeeHex = "0" + ieeeHex
	}
	ieeeU64, err := strconv.ParseUint(ieeeHex, 16, 64)
	if err != nil {
		panic(err)
	}
	got, err := jcs.NumberToJSON(math.Float64frombits(ieeeU64))

	if expected == invalidNumber {
		if err == nil {
			panic(fmt.Sprintf("hex %s: expected error, got %q", ieeeHex, got))
		}
		return
	}
	if err != nil {
		panic(err)
	}
	if got != expected {
		conversionErrors++
		fmt.Printf("\nhex:      %s\ngot:      %s\nexpected: %s\n", ieeeHex, got, expected)
		return
	}
	// Round-trip sanity: re-parsing the output must give back the same bits.
	parsed, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		panic(fmt.Sprintf("parse %q: %v", expected, err))
	}
	if parsed != math.Float64frombits(ieeeU64) {
		panic(fmt.Sprintf("round-trip mismatch for %s", ieeeHex))
	}
}

// discrete is the fixed set of hand-picked test vectors.
var discrete = []struct{ hex, want string }{
	{"4340000000000001", "9007199254740994"},
	{"4340000000000002", "9007199254740996"},
	{"444b1ae4d6e2ef50", "1e+21"},
	{"3eb0c6f7a0b5ed8d", "0.000001"},
	{"3eb0c6f7a0b5ed8c", "9.999999999999997e-7"},
	{"8000000000000000", "0"},           // negative zero → "0"
	{"7fffffffffffffff", invalidNumber}, // NaN
	{"7ff0000000000000", invalidNumber}, // +Infinity
	{"fff0000000000000", invalidNumber}, // -Infinity
}

func main() {
	flag.Parse()

	for _, tc := range discrete {
		verify(tc.hex, tc.want)
	}
	fmt.Printf("Discrete tests: %d passed\n", len(discrete))

	if *testFile == "" {
		fmt.Println("No large test file specified; skipping. Use -testfile or ES6_TESTFILE.")
		if conversionErrors > 0 {
			fmt.Fprintf(os.Stderr, "****** ERRORS: %d *******\n", conversionErrors)
			os.Exit(1)
		}
		return
	}

	f, err := os.Open(*testFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s: %v\n", *testFile, err)
		os.Exit(1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount%1_000_000 == 0 {
			fmt.Printf("line: %d\n", lineCount)
		}
		line := scanner.Text()
		comma := strings.IndexByte(line, ',')
		if comma <= 0 {
			panic(fmt.Sprintf("line %d: missing comma", lineCount))
		}
		verify(line[:comma], line[comma+1:])
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
		os.Exit(1)
	}

	if conversionErrors == 0 {
		fmt.Printf("Successful operation. Lines read: %d\n", lineCount)
	} else {
		fmt.Fprintf(os.Stderr, "****** ERRORS: %d *******\n", conversionErrors)
		os.Exit(1)
	}
}
