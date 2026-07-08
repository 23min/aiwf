package logger

import (
	"crypto/rand"
	"encoding/hex"
)

// NewRunID returns a fresh, per-invocation correlation id: 16 hex
// characters from 8 cryptographically random bytes. Not an RFC-4122
// UUID — nothing here needs that format, only enough entropy that two
// concurrent aiwf invocations never collide (ADR-0017 Decision #7),
// so a stdlib-only crypto/rand read is enough without a new dependency.
func NewRunID() string {
	var b [8]byte
	// crypto/rand.Read on an *os.File-backed source only errors when
	// the OS entropy source itself is broken — not reproducible in a
	// test environment, and this package's per-invocation id is best-
	// effort correlation, not a security boundary, so a read failure
	// falls back to the zero-value id rather than propagating an error
	// that would otherwise have to thread through every caller.
	_, _ = rand.Read(b[:]) //coverage:ignore crypto/rand.Read fails only when the OS entropy source is broken
	return hex.EncodeToString(b[:])
}
