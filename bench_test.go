// Copyright 2026 deszhou.
//
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadTestFile(name string) []byte {
	data, err := os.ReadFile(filepath.Join("testdata/input", name))
	if err != nil {
		panic(err)
	}
	return data
}

var (
	inputStructures = loadTestFile("structures.json")
	inputArrays     = loadTestFile("arrays.json")
	inputUnicode    = loadTestFile("unicode.json")
	inputValues     = loadTestFile("values.json")
	inputWeird      = loadTestFile("weird.json")
)

func BenchmarkTransformStructures(b *testing.B) {
	for b.Loop() {
		Transform(inputStructures)
	}
}

func BenchmarkTransformArrays(b *testing.B) {
	for b.Loop() {
		Transform(inputArrays)
	}
}

func BenchmarkTransformUnicode(b *testing.B) {
	for b.Loop() {
		Transform(inputUnicode)
	}
}

func BenchmarkTransformValues(b *testing.B) {
	for b.Loop() {
		Transform(inputValues)
	}
}

func BenchmarkTransformWeird(b *testing.B) {
	for b.Loop() {
		Transform(inputWeird)
	}
}

func BenchmarkTransformDeeplyNested(b *testing.B) {
	const depth = 1024
	leaf := `"` + strings.Repeat("x", 4096) + `"`
	cases := []struct {
		name, open, close string
	}{
		{"array", "[", "]"},
		{"unordered_object", `{"z":`, `,"a":0}`},
	}
	for _, tc := range cases {
		input := []byte(strings.Repeat(tc.open, depth) + leaf + strings.Repeat(tc.close, depth))
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := Transform(input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
