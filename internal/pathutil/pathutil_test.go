package pathutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInside(t *testing.T) {
	sep := string(filepath.Separator)
	cases := []struct {
		name      string
		root      string
		candidate string
		want      bool
	}{
		{"exact match", "/repo", "/repo", true},
		{"direct child", "/repo", "/repo/a", true},
		{"nested grandchild", "/repo", "/repo/a/b/c", true},
		{"trailing slash on root", "/repo/", "/repo/a", true},
		{"trailing slash on candidate", "/repo", "/repo/a/", true},
		{"dotdot escapes one level", "/repo", "/repo/../etc", false},
		{"dotdot then back in", "/repo", "/repo/../repo/a", true},
		{"absolute outside", "/repo", "/etc/passwd", false},
		{"prefix without separator (repository)", "/repo", "/repository/x", false},
		{"prefix without separator (repo-evil)", "/repo", "/repo-evil/x", false},
		{"sibling shared prefix", "/var/r", "/var/rx", false},
		{"empty candidate", "/repo", "", false},
		{"empty root", "", "/repo/x", false},
		{"relative root fails closed", "repo", "repo/x", false},
		{"relative candidate fails closed", "/repo", "repo/x", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				t.Skip("posix path layout")
			}
			got := Inside(tc.root, tc.candidate)
			if got != tc.want {
				t.Errorf("Inside(%q, %q) = %v, want %v (sep=%q)", tc.root, tc.candidate, got, tc.want, sep)
			}
		})
	}
}

func TestResolve_existingFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(target)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Resolve(%q) = %q, want %q", target, got, want)
	}
}

func TestResolve_existingDir(t *testing.T) {
	dir := t.TempDir()
	got, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Resolve(%q) = %q, want %q", dir, got, want)
	}
}

func TestResolve_missingPath_lexicalFallback(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist", "child")
	got, err := Resolve(missing)
	if err != nil {
		t.Fatalf("Resolve missing: want no error (lexical fallback), got %v", err)
	}
	want := filepath.Clean(missing)
	if got != want {
		t.Errorf("Resolve(missing) = %q, want %q", got, want)
	}
}

func TestResolve_symlinkInside(t *testing.T) {
	dir := t.TempDir()
	dirReal, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dirReal, "a")
	if werr := os.WriteFile(target, []byte("x"), 0o600); werr != nil {
		t.Fatal(werr)
	}
	link := filepath.Join(dirReal, "link")
	if serr := os.Symlink(target, link); serr != nil {
		t.Fatal(serr)
	}
	got, err := Resolve(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Errorf("Resolve(symlink) = %q, want %q", got, target)
	}
}

func TestResolve_symlinkOutside(t *testing.T) {
	outside := t.TempDir()
	inside := t.TempDir()
	insideReal, err := filepath.EvalSymlinks(inside)
	if err != nil {
		t.Fatal(err)
	}
	outsideReal, err := filepath.EvalSymlinks(outside)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(outsideReal, "elsewhere")
	if werr := os.WriteFile(target, []byte("x"), 0o600); werr != nil {
		t.Fatal(werr)
	}
	link := filepath.Join(insideReal, "link")
	if serr := os.Symlink(target, link); serr != nil {
		t.Fatal(serr)
	}
	got, err := Resolve(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Errorf("Resolve(symlink-outside) = %q, want %q", got, target)
	}
	if Inside(insideReal, got) {
		t.Errorf("Inside(%q, %q) = true, want false (symlink escape)", insideReal, got)
	}
}

func TestResolve_brokenSymlink(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "broken")
	if err := os.Symlink(filepath.Join(dir, "nope"), link); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(link); err == nil {
		t.Error("Resolve(broken-symlink): want error, got nil")
	}
}

func TestResolve_symlinkLoop(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(a); err == nil {
		t.Error("Resolve(symlink-loop): want error, got nil")
	}
}

func TestResolve_lstatPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix permissions")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	dir := t.TempDir()
	dirReal, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(dirReal, "noaccess")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(parent, "file")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	if _, err := Resolve(target); err == nil {
		t.Error("Resolve(no-access): want error, got nil")
	}
}

func TestResolve_relativeInputRejected(t *testing.T) {
	for _, in := range []string{"a", "./a", "../a", ""} {
		t.Run(in, func(t *testing.T) {
			_, err := Resolve(in)
			if err == nil {
				t.Fatalf("Resolve(%q): want error, got nil", in)
			}
			if !errors.Is(err, ErrNotAbsolute) {
				t.Errorf("Resolve(%q): want ErrNotAbsolute, got %v", in, err)
			}
		})
	}
}
