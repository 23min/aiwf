package main

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// M-069 AC-1 — Envelope conforms to documented schema for every
// --format=json verb.
//
// `internal/render/render.go` documents the JSON envelope contract:
// every --format=json invocation emits a single object with the
// load-bearing keys `tool` (always "aiwf"), `version` (non-empty
// string), `status` (closed set "ok" / "findings" / "error"),
// `findings` (array, never null/missing), and the optional
// verb-specific `result` and `metadata`. Downstream CI tooling keys
// off `findings` the same way across every verb and switches on the
// verb name to interpret `result`. A regression where a verb omits
// `findings`, returns a `status` outside the closed set, or quietly
// adds a top-level key is a silent breaking change for those
// consumers — and no current test catches it across the full set of
// JSON-emitting verbs.
//
// This test exercises every documented `--format=json` verb through
// the same dispatcher production uses (`run([]string{...})`), parses
// stdout, and asserts the envelope's *shape*: locked-in fields
// compared with go-cmp.Diff, type assertions for the run-varying
// values (version, metadata contents), and a per-verb assertion that
// `result` is either present-as-object/array or deliberately absent
// (the documented "check / contract verify → findings is the result"
// pattern).
//
// The verb table is the source of truth: a new --format=json verb
// that lands without an entry here is the regression we want this
// test to surface on the next CI run.

// envelopeRequiredKeys is the documented closed set of required
// top-level keys per internal/render/render.go's package godoc.
// Anything outside required ∪ optional is drift.
var (
	envelopeRequiredKeys  = []string{"findings", "status", "tool", "version"}
	envelopeOptionalKeys  = []string{"metadata", "result"}
	envelopeAllowedStatus = map[string]bool{"ok": true, "findings": true, "error": true}
)

// resultKind captures whether a verb is expected to populate the
// envelope's `result` field, and if so, whether the value is an
// object or array. Verbs whose findings *are* the result (check,
// contract verify) document that explicitly in render.go and ship
// `result` absent.
type resultKind int

const (
	resultAbsent resultKind = iota // result key is missing or null
	resultObject                   // result is a JSON object (map)
	resultArray                    // result is a JSON array
)

type envelopeVerbCase struct {
	name           string
	setup          func(t *testing.T, root string)
	args           []string // <root> placeholder is substituted at call time
	wantResultKind resultKind
}

// TestEnvelopeSchemaConformance_AllJSONVerbs (M-069 AC-1) drives every
// verb that supports --format=json through the dispatcher seam,
// captures the envelope, and asserts conformance to the documented
// schema. The assertions are structural (key set + closed-set values
// + types) rather than substring-based; a verb that drifts will
// surface here even if its specific tests still pass.
func TestEnvelopeSchemaConformance_AllJSONVerbs(t *testing.T) {
	standardSetup := func(t *testing.T, root string) {
		t.Helper()
		if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
			t.Fatalf("init: %d", rc)
		}
		if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
			t.Fatalf("add epic: %d", rc)
		}
		if rc := run([]string{"add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "First", "--actor", "human/test", "--root", root}); rc != exitOK {
			t.Fatalf("add milestone: %d", rc)
		}
		if rc := run([]string{"add", "ac", "M-001", "--title", "AC sample", "--actor", "human/test", "--root", root}); rc != exitOK {
			t.Fatalf("add ac: %d", rc)
		}
	}
	initOnly := func(t *testing.T, root string) {
		t.Helper()
		if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
			t.Fatalf("init: %d", rc)
		}
	}
	noSetup := func(t *testing.T, root string) { t.Helper() }

	cases := []envelopeVerbCase{
		// `check` — findings is the result; no `result` key per render.go godoc.
		{
			name:           "check",
			setup:          standardSetup,
			args:           []string{"check", "--root", "<root>", "--format=json"},
			wantResultKind: resultAbsent,
		},
		// `show <id>` and composite show — verb-specific result is required.
		{
			name:           "show entity",
			setup:          standardSetup,
			args:           []string{"show", "--root", "<root>", "--format=json", "E-01"},
			wantResultKind: resultObject,
		},
		{
			name:           "show composite AC",
			setup:          standardSetup,
			args:           []string{"show", "--root", "<root>", "--format=json", "M-001/AC-1"},
			wantResultKind: resultObject,
		},
		// `history <id>` — events list lives under result.events but result is an object.
		{
			name:           "history",
			setup:          standardSetup,
			args:           []string{"history", "--root", "<root>", "--format=json", "E-01"},
			wantResultKind: resultObject,
		},
		// `status` — project snapshot under result.
		{
			name:           "status",
			setup:          standardSetup,
			args:           []string{"status", "--root", "<root>", "--format=json"},
			wantResultKind: resultObject,
		},
		// `list` (no-args) — per-kind counts under result (object).
		{
			name:           "list no-args",
			setup:          standardSetup,
			args:           []string{"list", "--root", "<root>", "--format=json"},
			wantResultKind: resultObject,
		},
		// `list` filtered — array of summary objects under result.
		{
			name:           "list filtered",
			setup:          standardSetup,
			args:           []string{"list", "--root", "<root>", "--kind", "milestone", "--format=json"},
			wantResultKind: resultArray,
		},
		// `schema [kind]` — closed-set schema dump under result.
		{
			name:           "schema all",
			setup:          noSetup,
			args:           []string{"schema", "--format", "json"},
			wantResultKind: resultObject,
		},
		{
			name:           "schema epic",
			setup:          noSetup,
			args:           []string{"schema", "--format", "json", "epic"},
			wantResultKind: resultObject,
		},
		// `template [kind]` — body-section template under result.
		{
			name:           "template all",
			setup:          noSetup,
			args:           []string{"template", "--format", "json"},
			wantResultKind: resultObject,
		},
		{
			name:           "template milestone",
			setup:          noSetup,
			args:           []string{"template", "--format", "json", "milestone"},
			wantResultKind: resultObject,
		},
		// `contract verify` — same family as check; findings is the result.
		{
			name:           "contract verify (no bindings)",
			setup:          initOnly,
			args:           []string{"contract", "verify", "--root", "<root>", "--format", "json"},
			wantResultKind: resultAbsent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := setupCLITestRepo(t)
			tc.setup(t, root)

			args := make([]string, len(tc.args))
			for i, a := range tc.args {
				if a == "<root>" {
					args[i] = root
				} else {
					args[i] = a
				}
			}

			captured := captureStdout(t, func() {
				if rc := run(args); rc != exitOK && rc != exitFindings {
					t.Fatalf("run %v = %d (want ok or findings)", args, rc)
				}
			})

			assertEnvelopeConforms(t, captured, tc.wantResultKind)
		})
	}
}

// assertEnvelopeConforms parses raw as the documented envelope and
// checks the schema in three layers:
//
//  1. Top-level key set: required keys present, every key in
//     required ∪ optional. Diff via go-cmp so a drift (extra key,
//     missing required) prints a precise ±-summary.
//  2. Pinned fields (tool, status closed-set membership) compared
//     with go-cmp.Diff. Run-varying values (version, metadata
//     contents) are checked via type assertions, mirroring the
//     IgnoreFields idiom for fields whose specific values aren't
//     part of the contract.
//  3. Per-verb result-kind: result is either deliberately absent
//     (check / contract verify pattern) or a JSON object / array.
func assertEnvelopeConforms(t *testing.T, raw []byte, wantResultKind resultKind) {
	t.Helper()

	var env map[string]any
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("envelope is not valid JSON: %v\nraw: %s", err, raw)
	}

	// Layer 1: top-level key set.
	gotKeys := make([]string, 0, len(env))
	for k := range env {
		gotKeys = append(gotKeys, k)
	}
	sort.Strings(gotKeys)

	wantKeys := append([]string{}, envelopeRequiredKeys...)
	for _, k := range envelopeOptionalKeys {
		if _, has := env[k]; has {
			wantKeys = append(wantKeys, k)
		}
	}
	sort.Strings(wantKeys)

	if diff := cmp.Diff(wantKeys, gotKeys); diff != "" {
		t.Errorf("envelope top-level key set mismatch (-want +got):\n%s\nraw: %s", diff, raw)
	}

	// Layer 2: pinned fields. tool is locked at "aiwf"; status must
	// be in the closed set. The status value itself isn't pinned per
	// case (a verb run against a tree with warnings legitimately
	// returns "findings"); we only assert membership.
	type pinned struct {
		Tool              string
		StatusInClosedSet bool
	}
	gotTool, _ := env["tool"].(string)
	gotStatus, _ := env["status"].(string)
	wantPinned := pinned{Tool: "aiwf", StatusInClosedSet: true}
	gotPinned := pinned{Tool: gotTool, StatusInClosedSet: envelopeAllowedStatus[gotStatus]}
	if diff := cmp.Diff(wantPinned, gotPinned); diff != "" {
		t.Errorf("envelope pinned fields mismatch (-want +got):\n%s\nstatus value: %q\nraw: %s", diff, gotStatus, raw)
	}

	// Run-varying field type checks.
	if v, ok := env["version"].(string); !ok || v == "" {
		t.Errorf("envelope.version = %v (type %T), want non-empty string\nraw: %s", env["version"], env["version"], raw)
	}
	if _, ok := env["findings"].([]any); !ok {
		t.Errorf("envelope.findings = %v (type %T), want []any (JSON array, possibly empty — never null/missing)\nraw: %s", env["findings"], env["findings"], raw)
	}
	if md, has := env["metadata"]; has && md != nil {
		if _, isObj := md.(map[string]any); !isObj {
			t.Errorf("envelope.metadata = %v (type %T), want object when present\nraw: %s", md, md, raw)
		}
	}

	// Layer 3: per-verb result-kind.
	got := env["result"]
	switch wantResultKind {
	case resultAbsent:
		if got != nil {
			t.Errorf("envelope.result is present (type %T) but verb contract says findings is the result; expected absent or null\nraw: %s", got, raw)
		}
	case resultObject:
		if _, ok := got.(map[string]any); !ok {
			t.Errorf("envelope.result = %v (type %T), want JSON object\nraw: %s", got, got, raw)
		}
	case resultArray:
		if _, ok := got.([]any); !ok {
			t.Errorf("envelope.result = %v (type %T), want JSON array\nraw: %s", got, got, raw)
		}
	}
}
