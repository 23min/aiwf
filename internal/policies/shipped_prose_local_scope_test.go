package policies

import "testing"

// TestDetectProseAssertions_LocalDeclarationsAreScoped pins that a
// function-local path declaration stands for its path inside its own
// function and nowhere else: a same-named identifier in another function is
// not a shipped path, while a local const joined to the read, or a local var
// initialized from one, still carries it.
func TestDetectProseAssertions_LocalDeclarationsAreScoped(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, body string
		want       int
	}{
		{
			name: "a local const does not lend its path to another function",
			body: `func elsewhere() { const rel = "internal/skills/embedded-rituals/x/SKILL.md"; _ = rel }

func load(rel string) string {
	data, _ := os.ReadFile(rel)
	return string(data)
}

func TestOther(t *testing.T) {
	if !strings.Contains(load("docs/other.md"), "a phrase") {
		t.Error("missing")
	}
}`,
		},
		{
			name: "a local const names its path inside its function",
			body: `func TestPin(t *testing.T) {
	const rel = "internal/skills/embedded-rituals/x/SKILL.md"
	data, _ := os.ReadFile(rel)
	if !strings.Contains(string(data), "a phrase") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			name: "a local var initialized from a read holds the document",
			body: `func TestPin(t *testing.T) {
	var body = readSkill(t, ritualPath)
	if !strings.Contains(body, "a phrase") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			name: "every name a multi-value read binds holds the document",
			body: `func readPair(rel string) (int, string) {
	data, _ := os.ReadFile(rel)
	return len(data), string(data)
}

func TestPin(t *testing.T) {
	var n, body = readPair(ritualPath)
	_ = n
	if !strings.Contains(body, "a phrase") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
		{
			name: "the name a multi-value declaration binds to a shipped path names it",
			body: `func pick(rel string) (string, bool) { return rel, true }

func TestPin(t *testing.T) {
	var p, ok = pick(ritualPath)
	_ = ok
	data, _ := os.ReadFile(p)
	if !strings.Contains(string(data), "a phrase") {
		t.Error("missing")
	}
}`,
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fset, files, paths := parseSyntheticPackage(t, map[string]string{
				"header.go": fixtureHeader,
				"a_test.go": "package pkg\n\n" + tt.body + "\n",
			})
			if got := detectProseAssertions(fset, files, paths); len(got) != tt.want {
				t.Errorf("got %d findings %+v, want %d", len(got), got, tt.want)
			}
		})
	}
}
