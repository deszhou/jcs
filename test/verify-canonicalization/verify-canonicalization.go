// Copyright 2026 deszhou.
// Portions copyright 2006-2019 WebPKI.org (cyberphone/json-canonicalization).
//
// SPDX-License-Identifier: Apache-2.0

// verify-canonicalization checks every file in testdata/input against the
// corresponding expected output in testdata/output and prints the UTF-8 hex
// representation of each result. Run from the repository root:
//
//	go run ./test/verify-canonicalization
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deszhou/jcs"
)

var testdata = flag.String("testdata", "testdata", "path to the testdata directory")

var failures int

func mustRead(dir, name string) []byte {
	data, err := os.ReadFile(filepath.Join(*testdata, dir, name))
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s/%s: %v\n", dir, name, err)
		os.Exit(1)
	}
	return data
}

func verify(fileName string) {
	actual, err := jcs.Transform(mustRead("input", fileName))
	if err != nil {
		fmt.Fprintf(os.Stderr, "transform %s: %v\n", fileName, err)
		failures++
		return
	}
	recycled, err := jcs.Transform(actual)
	if err != nil {
		fmt.Fprintf(os.Stderr, "re-transform %s: %v\n", fileName, err)
		failures++
		return
	}
	expected := mustRead("output", fileName)

	// Print UTF-8 hex dump (32 bytes per line).
	fmt.Printf("\nFile: %s\n", fileName)
	for i, b := range actual {
		if i > 0 && i%32 == 0 {
			fmt.Println()
		} else if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%02x", b)
	}
	fmt.Println()

	if !bytes.Equal(actual, expected) || !bytes.Equal(actual, recycled) {
		failures++
		fmt.Println("THE TEST ABOVE FAILED!")
	}
}

func main() {
	flag.Parse()

	entries, err := os.ReadDir(filepath.Join(*testdata, "input"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input dir: %v\n", err)
		os.Exit(1)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			verify(entry.Name())
		}
	}

	if failures == 0 {
		fmt.Println("\nAll tests succeeded!")
	} else {
		fmt.Fprintf(os.Stderr, "\n****** ERRORS: %d *******\n", failures)
		os.Exit(1)
	}
}
