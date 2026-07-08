package main

import "testing"

func TestRun_HelpReturnsZero(t *testing.T) {
	t.Parallel()
	if code := run([]string{"--help"}); code != 0 {
		t.Fatalf("run([--help]) = %d, want 0", code)
	}
}

func TestRun_UnknownCommandReturnsOne(t *testing.T) {
	t.Parallel()
	if code := run([]string{"bogus-verb"}); code != 1 {
		t.Fatalf("run([bogus-verb]) = %d, want 1", code)
	}
}
