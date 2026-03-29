// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func mustTransform(t *testing.T, in string) string {
	t.Helper()
	out, err := Transform([]byte(in))
	if err != nil {
		t.Fatalf("Transform(%q): unexpected error: %v", in, err)
	}
	return string(out)
}

func wantError(t *testing.T, in string) {
	t.Helper()
	_, err := Transform([]byte(in))
	if err == nil {
		t.Fatalf("Transform(%q): want error, got nil", in)
	}
}

// ── file-based integration tests ─────────────────────────────────────────────

// TestTransform runs each testdata pair and also verifies idempotence.
func TestTransform(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc     string
		filename string
	}{
		{"Null", "null.json"},
		{"True", "true.json"},
		{"False", "false.json"},
		{"Arrays", "arrays.json"},
		{"French", "french.json"},
		{"SimpleString", "simpleString.json"},
		{"Structures", "structures.json"},
		{"Unicode", "unicode.json"},
		{"Values", "values.json"},
		{"Weird", "weird.json"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			inPath := filepath.Join("testdata", "input", tc.filename)
			outPath := filepath.Join("testdata", "output", tc.filename)
			input, err := os.ReadFile(inPath)
			if err != nil {
				t.Fatalf("read input: %v", err)
			}
			want, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("read expected: %v", err)
			}
			got, err := Transform(input)
			if err != nil {
				t.Fatalf("Transform: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("mismatch for %s\ngot:  %q\nwant: %q", tc.filename, got, want)
			}
			// idempotence: a second pass must produce the same bytes
			again, err := Transform(got)
			if err != nil {
				t.Fatalf("second Transform: %v", err)
			}
			if !bytes.Equal(again, got) {
				t.Fatalf("second Transform changed output for %s", tc.filename)
			}
		})
	}
}

// ── nil input ────────────────────────────────────────────────────────────────

func TestTransform_nil(t *testing.T) {
	_, err := Transform(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

// ── error cases ──────────────────────────────────────────────────────────────

func TestTransform_errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc string
		in   string
	}{
		// structural
		{"empty input", ""},
		{"truncated object", `{`},
		{"truncated object after key", `{"a"`},
		{"truncated array", `[`},
		{"truncated string value", `"hello`},
		{"missing colon", `{"a" 1}`},
		{"missing value", `{"a":}`},
		{"trailing garbage after object", `{}x`},
		{"trailing garbage after array", `[]x`},
		{"trailing garbage after string", `"a"x`},

		// semantic
		{"duplicate key", `{"a":1,"a":2}`},

		// string escapes
		{"invalid escape sequence", `"\q"`},
		{"lone high surrogate", `"\ud83d"`},
		{"lone low surrogate", `"\ude02"`},

		// byte-level
		{"control char 0x01 in string", "\"\x01\""},
		{"control char 0x1f in string", "\"\x1f\""},
		{"non-ASCII byte outside string", "\x80"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			wantError(t, tc.in)
		})
	}
}

// ── root scalar whitespace ────────────────────────────────────────────────────

func TestTransform_rootScalars(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want string }{
		{" true", "true"},
		{"\tfalse\n", "false"},
		{" null\r\n", "null"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := mustTransform(t, tc.in); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ── object key sorting (RFC 8785 §3.2.3 – UTF-16 code-unit order) ────────────

func TestTransform_objectSorting(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc string
		in   string
		want string
	}{
		{
			"two keys sorted",
			`{"b":2,"a":1}`,
			`{"a":1,"b":2}`,
		},
		{
			"uppercase before lowercase (U+0041 < U+0061)",
			`{"b":2,"A":1}`,
			`{"A":1,"b":2}`,
		},
		{
			"empty key sorts before any non-empty key",
			`{"b":2,"":0}`,
			`{"":0,"b":2}`,
		},
		{
			"numeric strings sorted lexicographically not numerically",
			`{"9":4,"2":3,"10":1,"1":0}`,
			`{"1":0,"10":1,"2":3,"9":4}`,
		},
		{
			"short key precedes longer key with same prefix",
			`{"ab":2,"a":1}`,
			`{"a":1,"ab":2}`,
		},
		{
			"nested object keys also sorted",
			`{"b":{"y":2,"x":1},"a":0}`,
			`{"a":0,"b":{"x":1,"y":2}}`,
		},
		{
			"object inside array keys sorted",
			`[{"z":3,"a":1}]`,
			`[{"a":1,"z":3}]`,
		},
		{
			"whitespace between tokens stripped",
			`{ "b" : 2 , "a" : 1 }`,
			`{"a":1,"b":2}`,
		},
		{
			"single key — no reordering needed",
			`{"only":true}`,
			`{"only":true}`,
		},
		{
			"empty object",
			`{}`,
			`{}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			if got := mustTransform(t, tc.in); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ── arrays ────────────────────────────────────────────────────────────────────

func TestTransform_arrays(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc string
		in   string
		want string
	}{
		{"empty array", `[]`, `[]`},
		{"single element", `[1]`, `[1]`},
		{"order preserved", `[3,1,2]`, `[3,1,2]`},
		{"nested arrays", `[[1,2],[3,4]]`, `[[1,2],[3,4]]`},
		{"whitespace stripped", `[ 1 , 2 , 3 ]`, `[1,2,3]`},
		{"mixed types preserve order", `[null,true,false,1,"x"]`, `[null,true,false,1,"x"]`},
		{
			"array of objects: element order preserved, keys inside sorted",
			`[{"b":2,"a":1},{"d":4,"c":3}]`,
			`[{"a":1,"b":2},{"c":3,"d":4}]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			if got := mustTransform(t, tc.in); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ── string escaping ───────────────────────────────────────────────────────────

func TestTransform_stringEscaping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc string
		in   string
		want string
	}{
		{"plain string unchanged", `"hello"`, `"hello"`},
		{"empty string", `""`, `""`},

		// RFC 8785 §3.2.2.2: solidus escape (\/) must be removed
		{"solidus escape removed", `"\/"`, `"/"`},

		// standard escapes round-trip
		{"backslash preserved", `"\\"`, `"\\"`},
		{"double-quote preserved", `"\""`, `"\""`},
		{"newline preserved", `"\n"`, `"\n"`},
		{"carriage return preserved", `"\r"`, `"\r"`},
		{"tab preserved", `"\t"`, `"\t"`},
		{"backspace preserved", `"\b"`, `"\b"`},
		{"form feed preserved", `"\f"`, `"\f"`},

		// \uXXXX normalization
		{"BMP escape normalized to char", `"\u0041"`, `"A"`},
		{"euro sign escape normalized", `"\u20ac"`, `"€"`},

		// control characters always get \u00XX in output
		{"control char U+0001 escaped in output", `"\u0001"`, `"\u0001"`},
		{"control char U+000f escaped in output", `"\u000f"`, `"\u000f"`},

		// surrogate pair → real codepoint
		{"surrogate pair decoded to emoji", `"\ud83d\ude02"`, `"😂"`},

		// key with escape in object
		{
			"escaped key in object",
			`{"\u006e":1}`,
			`{"n":1}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			if got := mustTransform(t, tc.in); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ── number formatting (inside a JSON structure) ───────────────────────────────

func TestTransform_numbers(t *testing.T) {
	t.Parallel()
	// Wrap each number as a JSON array element so the parser goes through
	// parseSimpleType (the normal path for embedded numbers).
	wrap := func(n string) string { return "[" + n + "]" }

	cases := []struct {
		desc string
		in   string
		want string
	}{
		// integers
		{"zero", "0", "0"},
		{"positive integer", "42", "42"},
		{"negative integer", "-1", "-1"},

		// sign of zero
		{"negative zero becomes zero", "-0", "0"},

		// trailing zeros / integer-valued floats
		{"trailing zero stripped", "1.50", "1.5"},
		{"float with integer value", "1.0", "1"},

		// format boundary: [1e-6, 1e21) uses fixed notation
		{"lower boundary 1e-6 uses fixed", "0.000001", "0.000001"},
		{"just below lower boundary uses e", "0.0000009", "9e-7"},
		{"large value below 1e21 uses fixed", "1e20", "100000000000000000000"},
		{"1e21 switches to e notation", "1e21", "1e+21"},
		{"large value uses e notation", "1e30", "1e+30"},

		// exponent formatting: leading zero stripped
		{"exponent leading zero stripped", "1e9", "1000000000"},
		{"negative exponent", "1e-7", "1e-7"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got := mustTransform(t, wrap(tc.in))
			want := "[" + tc.want + "]"
			if got != want {
				t.Fatalf("number %q: got %q, want %q", tc.in, got, want)
			}
		})
	}
}

// ── idempotence (broader than the file-based test) ───────────────────────────

func TestTransform_idempotent(t *testing.T) {
	t.Parallel()
	inputs := []string{
		`{"z":3,"a":1,"m":2}`,
		`[{"b":2,"a":1},{"d":4,"c":3}]`,
		`{"a":{"z":26,"a":1},"b":[3,1,2]}`,
		`{"numbers":[1e30,0.000001,-0],"str":"\u0041\/\n"}`,
	}
	for _, in := range inputs {
		t.Run(in[:min(len(in), 30)], func(t *testing.T) {
			t.Parallel()
			first := mustTransform(t, in)
			second := mustTransform(t, first)
			if first != second {
				t.Fatalf("not idempotent:\nfirst:  %q\nsecond: %q", first, second)
			}
		})
	}
}

// ── NumberToJSON additional edge cases ───────────────────────────────────────

func TestNumberToJSON_additional(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc    string
		ieeeHex string
		want    string
		wantErr bool
	}{
		// exact boundaries of the format switch
		{"1e-6 (lower bound, fixed)", "3eb0c6f7a0b5ed8d", "0.000001", false},
		{"just below 1e-6 (e format)", "3eb0c6f7a0b5ed8c", "9.999999999999997e-7", false},
		{"1e+21 (upper bound, e format)", "444b1ae4d6e2ef50", "1e+21", false},

		// representable integers
		{"max safe integer + 1", "4340000000000001", "9007199254740994", false},
		{"max safe integer + 2", "4340000000000002", "9007199254740996", false},

		// sign edge cases
		{"negative zero → 0", "8000000000000000", "0", false},
		{"negative one", "bff0000000000000", "-1", false},

		// special values must error
		{"NaN (quiet)", "7fffffffffffffff", "null", true},
		{"+Infinity", "7ff0000000000000", "null", true},
		{"-Infinity", "fff0000000000000", "null", true},
		{"NaN (signaling)", "7ff4000000000000", "null", true},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			runNumberCase(t, tc.ieeeHex, tc.want, tc.wantErr)
		})
	}
}

// ── key-in-error messages (spot check error text) ────────────────────────────

func TestTransform_errorMessages(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc        string
		in          string
		errContains string
	}{
		{"duplicate key message names the key", `{"foo":1,"foo":2}`, "foo"},
		{"invalid escape names the char", `"\q"`, `\q`},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			_, err := Transform([]byte(tc.in))
			if err == nil {
				t.Fatalf("want error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.errContains)
			}
		})
	}
}
