// Copyright 2026 deszhou.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// SPDX-License-Identifier: Apache-2.0

// Package jcs implements JSON Canonicalization Scheme (JCS) per RFC 8785.
//
// Use [Transform] to convert UTF-8 JSON bytes into a canonical form suitable
// for stable hashing or signing. Number formatting follows ECMAScript rules;
// object keys are sorted by UTF-16 code unit order.
//
// Reference: https://www.rfc-editor.org/rfc/rfc8785
package jcs
