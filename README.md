

# jcs — JSON Canonicalization (RFC 8785)

[![Go Report Card](https://goreportcard.com/badge/github.com/deszhou/jcs)](https://goreportcard.com/report/github.com/deszhou/jcs)
[![godoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://pkg.go.dev/github.com/deszhou/jcs)
[![GitHub license](https://img.shields.io/github/license/deszhou/jcs.svg?style=flat)](https://github.com/deszhou/jcs/blob/master/LICENSE)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/deszhou/jcs.svg?style=flat)](https://github.com/deszhou/jcs)
[![Stable](https://img.shields.io/badge/stability-v1.0%20stable-brightgreen.svg?style=flat)](https://github.com/deszhou/jcs/releases)

Cryptographic hashing and signing need a **stable byte representation** of JSON. The same logical value can be serialized many ways — key order, spacing, number formatting — making naive JSON unsuitable as a signing payload.

**JCS** ([RFC 8785](https://www.rfc-editor.org/rfc/rfc8785)) solves this by defining a canonical form:

- Serializes primitives like **ECMAScript** `JSON.stringify`
- Restricts to the **I-JSON** subset ([RFC 7493](https://www.rfc-editor.org/rfc/rfc7493))
- Sorts object keys by **UTF-16 code unit** order, recursively
- Preserves array element order

This library is a **v1.0 stable** implementation in Go. The public API (`Transform`, `NumberToJSON`) will not have breaking changes within the v1 major version.

---

## Install

Requires **Go 1.24+**.

```bash
go get github.com/deszhou/jcs
```

---

## Usage

### Transform — canonicalize a JSON document

```go
import "github.com/deszhou/jcs"

canonical, err := jcs.Transform(jsonBytes)
if err != nil {
    log.Fatal(err)
}
// canonical is a UTF-8 JCS byte slice, safe to hash or sign directly
```

`Transform` accepts any valid JSON root value: object, array, or scalar (`true` / `null` / number / string). Leading and trailing whitespace around the root value is allowed.

**Typical use — sign a canonical payload:**

```go
canonical, err := jcs.Transform(jsonBytes)
if err != nil {
    return err
}
digest := sha256.Sum256(canonical)
signature := sign(privateKey, digest[:])
```

### NumberToJSON — ES6-style float64 formatting

```go
formatted := jcs.NumberToJSON(1e30)   // "1e+30"
formatted  = jcs.NumberToJSON(4.5)    // "4.5"
formatted  = jcs.NumberToJSON(0.002)  // "0.002"
```

Use this when you need to format a standalone float64 in JCS-compatible form — for example, when constructing a canonical payload manually rather than round-tripping through `json.Marshal`.

---

## Errors

`Transform` returns an error for any input that violates RFC 8785 or I-JSON constraints:

| Condition                         | Example               |
| --------------------------------- | --------------------- |
| Invalid JSON                      | `{key: value}`        |
| Duplicate object keys             | `{"a":1,"a":2}`       |
| Number out of safe-integer range  | integers beyond ±2⁵³  |
| Lone UTF-16 surrogate in a string | `"\uD800"` (unpaired) |

---

## Example

**Input:**

```json
{
  "numbers": [333333333.33333329, 1E30, 4.50, 2e-3, 0.000000000000000000000000001],
  "string": "\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/",
  "literals": [null, true, false]
}
```

**Canonical output** (single line, keys sorted, numbers normalized):

```
{"literals":[null,true,false],"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],"string":"€$\u000f\nA'B\"\\\\\"/"}
```

Key differences from the input: `"literals"` sorts before `"numbers"` and `"string"`; numbers are normalized to ES6 form; control characters use the shortest valid escape sequence.

---

## Performance

The same inputs ran against cyberphone at `19d51d7fe467`, gowebpki at
`v1.0.2`, and this library. Results are medians of three runs on an Apple M5
(darwin/arm64) with Go 1.26.2 and `GOMAXPROCS=1`. All three implementations
produced byte-identical canonical output for the benchmark inputs.

### Time per operation (ns/op)

| Scenario | cyberphone | gowebpki | **this library** |
| --- | ---: | ---: | ---: |
| Structures (nested object) | 1,921 | 2,197 | **1,016** |
| Arrays (mixed array) | 569.4 | 753.3 | **381.8** |
| Unicode (non-ASCII values) | 372.8 | 321.3 | **80.6** |
| Values (numbers and escapes) | 1,793 | 2,517 | **997.2** |
| Weird (special characters in keys) | 2,442 | 2,555 | **1,194** |
| 1,024 nested arrays | 514,356 | 74,594 | **14,871** |
| 1,024 nested objects with unordered keys | 1,238,440 | 374,331 | **191,510** |

### Allocations per operation (allocs/op)

| Scenario | cyberphone | gowebpki | **this library** |
| --- | ---: | ---: | ---: |
| Structures (nested object) | 95 | 84 | **17** |
| Arrays (mixed array) | 27 | 25 | **10** |
| Unicode (non-ASCII values) | 16 | 13 | **3** |
| Values (numbers and escapes) | 53 | 51 | **22** |
| Weird (special characters in keys) | 95 | 87 | **25** |
| 1,024 nested arrays | 2,074 | 1,050 | **1** |
| 1,024 nested objects with unordered keys | 15,386 | 14,362 | **2,063** |

Both deep cases use a 4 KiB string at the leaf. The object case reorders keys
at every level. Arrays and scalar values append directly to a shared buffer.
Objects with out-of-order keys record byte ranges and emit them in UTF-16
key order, avoiding a copy of the serialized subtree at each level.

Run this library's benchmarks from the repository root.

```sh
go test -run '^$' -bench '^BenchmarkTransform' -benchmem -benchtime=120ms -count=3 -cpu=1
```

---

## Contributing

Bug reports, test cases, and pull requests are welcome.

1. Open an issue to discuss what you'd like to change before a large PR
2. Run `go test ./...` and `go vet ./...` before submitting
3. For benchmark changes, include before/after numbers in the PR description

---

## Attribution

Derived from **[cyberphone/json-canonicalization](https://github.com/cyberphone/json-canonicalization)** (Anders Rundgren), the original multi-language JCS reference implementation. Licensed Apache-2.0 — see [LICENSE](LICENSE).

---

## See also

- [RFC 8785](https://www.rfc-editor.org/rfc/rfc8785) — the specification
- [RFC 7493](https://www.rfc-editor.org/rfc/rfc7493) — I-JSON subset
- [JWS-JCS](https://github.com/cyberphone/jws-jcs) — combining JCS with JWS (RFC 7515)
- [Browser JCS demo](https://cyberphone.github.io/doc/security/browser-json-canonicalization.html)
