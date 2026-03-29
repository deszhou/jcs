// Copyright 2026 deszhou.
//
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"os"
	"path/filepath"
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
