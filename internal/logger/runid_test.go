package logger

import "testing"

func TestNewRunID_NonEmpty(t *testing.T) {
	t.Parallel()
	if id := NewRunID(); id == "" {
		t.Fatal("NewRunID() = \"\", want a non-empty id")
	}
}

func TestNewRunID_DistinctAcrossCalls(t *testing.T) {
	t.Parallel()
	seen := make(map[string]bool)
	for range 1000 {
		id := NewRunID()
		if seen[id] {
			t.Fatalf("NewRunID() produced a duplicate: %q", id)
		}
		seen[id] = true
	}
}

func TestNewRunID_HexEncoded(t *testing.T) {
	t.Parallel()
	id := NewRunID()
	for _, r := range id {
		isDigit := r >= '0' && r <= '9'
		isLowerHexLetter := r >= 'a' && r <= 'f'
		if !isDigit && !isLowerHexLetter {
			t.Fatalf("NewRunID() = %q, want lowercase hex only (found %q)", id, r)
		}
	}
}
