package stresstest

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/check"
)

// readpath_agreement_test.go — M-0300/AC-1 and AC-3. The comparison core
// is driven directly with constructed observations, which is the only
// way to reach its failing direction: both defects that motivated the
// property (G-0558, G-0556) are repaired on main, so no repository the
// harness can build makes the real surfaces contradict each other.

// readPathClaimAt is one surface's classification of one subject, flat
// enough that a test reads as what a surface stated rather than as map
// literal punctuation.
type readPathClaimAt struct {
	Subject readPathSubject
	Claim   readPathClaim
}

func claimAt(entityID, code, subcode, severity string) readPathClaimAt {
	return readPathClaimAt{
		Subject: readPathSubject{EntityID: entityID, Code: code},
		Claim:   readPathClaim{Subcode: subcode, Severity: severity},
	}
}

// substantive builds an itemized observation from a surface name and the
// claims that surface stated.
func substantive(surface string, claims ...readPathClaimAt) readPathObservation {
	obs := readPathObservation{
		Surface:  surface,
		Claims:   make(map[readPathSubject]readPathClaimSet, len(claims)),
		Itemized: true,
	}
	for _, c := range claims {
		if obs.Claims[c.Subject] == nil {
			obs.Claims[c.Subject] = readPathClaimSet{}
		}
		obs.Claims[c.Subject][c.Claim] = true
		if c.Claim.Severity == severityError {
			obs.Blocking++
		}
	}
	return obs
}

func TestClassifyReadPathAgreement_ReportsTwoSurfacesClassifyingOneSubjectDifferently(t *testing.T) {
	t.Parallel()

	gate := substantive("aiwf check",
		claimAt("G-0001", check.CodeRefsResolve, "cross-branch-local-only", severityError))
	other := substantive("aiwf check --fast",
		claimAt("G-0001", check.CodeRefsResolve, "cross-branch-pending", "warning"))

	got := classifyReadPathAgreement("step 1", gate, []readPathObservation{other})

	if len(got) != 1 {
		t.Fatalf("classifyReadPathAgreement() returned %d violations, want 1: %+v", len(got), got)
	}
	msg := got[0].Message
	for _, want := range []string{
		"step 1", "aiwf check", "aiwf check --fast", "G-0001", check.CodeRefsResolve,
		"cross-branch-local-only", "cross-branch-pending",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("violation message %q does not name %q", msg, want)
		}
	}
}

func TestClassifyReadPathAgreement_ReportsASeverityDisagreementOnAnIdenticalSubcode(t *testing.T) {
	t.Parallel()

	gate := substantive("aiwf check",
		claimAt("M-0002", check.CodeACsTDDAudit, "met-without-done", severityError))
	other := substantive("aiwf check --fast",
		claimAt("M-0002", check.CodeACsTDDAudit, "met-without-done", "warning"))

	got := classifyReadPathAgreement("step 2", gate, []readPathObservation{other})

	if len(got) != 1 {
		t.Fatalf("classifyReadPathAgreement() returned %d violations, want 1: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, severityError) || !strings.Contains(got[0].Message, "warning") {
		t.Errorf("violation message %q does not carry both severities", got[0].Message)
	}
}

func TestClassifyReadPathAgreement_ASubjectOnlyOneSurfaceClaimsIsNotAContradiction(t *testing.T) {
	t.Parallel()

	// --shape-only runs one rule, so it says nothing at all about
	// refs-resolve. Silence is not a competing claim; treating it as one
	// would fire on every run against a correct tree.
	gate := substantive("aiwf check",
		claimAt("G-0001", check.CodeRefsResolve, "unresolved", severityError))
	other := substantive("aiwf check --shape-only")

	if got := classifyReadPathAgreement("step 3", gate, []readPathObservation{other}); got != nil {
		t.Errorf("classifyReadPathAgreement() = %+v, want no violations", got)
	}
}

func TestClassifyReadPathAgreement_AContainedClaimSetIsNotAContradiction(t *testing.T) {
	t.Parallel()

	// One entity, two offending prose tokens. Both surfaces agree the
	// malformed one is malformed; the cheaper surface declined the other,
	// so its claim set is contained in the gate's. Containment is the
	// absence rule at per-classification granularity — the gate saw more,
	// it did not see differently.
	gate := substantive("aiwf check",
		claimAt("M-0002", check.CodeBodyProseID, "malformed-shape", severityError),
		claimAt("M-0002", check.CodeBodyProseID, "cross-branch-pending", "warning"))
	other := substantive("aiwf check --fast",
		claimAt("M-0002", check.CodeBodyProseID, "malformed-shape", severityError))

	if got := classifyReadPathAgreement("step 4", gate, []readPathObservation{other}); got != nil {
		t.Errorf("classifyReadPathAgreement() = %+v, want no violations", got)
	}
}

func TestClassifyReadPathAgreement_ReportsOverlappingClaimSetsThatNeitherContains(t *testing.T) {
	t.Parallel()

	// Same two-token entity, but now the surfaces classify the second
	// token incompatibly. Neither set contains the other, so the shared
	// malformed-shape claim does not excuse the divergence.
	gate := substantive("aiwf check",
		claimAt("M-0002", check.CodeBodyProseID, "malformed-shape", severityError),
		claimAt("M-0002", check.CodeBodyProseID, "cross-branch-pending", "warning"))
	other := substantive("aiwf check --fast",
		claimAt("M-0002", check.CodeBodyProseID, "malformed-shape", severityError),
		claimAt("M-0002", check.CodeBodyProseID, "unresolved", severityError))

	if got := classifyReadPathAgreement("step 5", gate, []readPathObservation{other}); len(got) != 1 {
		t.Errorf("classifyReadPathAgreement() returned %d violations, want 1: %+v", len(got), got)
	}
}

func TestClassifyReadPathAgreement_AgreesWhenEverySharedSubjectMatches(t *testing.T) {
	t.Parallel()

	gate := substantive("aiwf check",
		claimAt("G-0001", check.CodeRefsResolve, "unresolved", severityError),
		claimAt("M-0002", check.CodeEntityBodyEmpty, "", "warning"))
	other := substantive("aiwf check --fast",
		claimAt("G-0001", check.CodeRefsResolve, "unresolved", severityError))

	if got := classifyReadPathAgreement("step 6", gate, []readPathObservation{other}); got != nil {
		t.Errorf("classifyReadPathAgreement() = %+v, want no violations", got)
	}
}

func TestClassifyReadPathAgreement_ComparesEveryPairNotOnlyAgainstTheGate(t *testing.T) {
	t.Parallel()

	// The two cheaper surfaces contradict each other while each stays
	// silent on what the gate claims. Comparing only against the gate
	// would miss it.
	gate := substantive("aiwf check")
	fast := substantive("aiwf check --fast",
		claimAt("G-0001", check.CodeBodyProseID, "unresolved", severityError))
	shape := substantive("aiwf check --shape-only",
		claimAt("G-0001", check.CodeBodyProseID, "malformed-shape", severityError))

	got := classifyReadPathAgreement("step 7", gate, []readPathObservation{fast, shape})

	if len(got) != 1 {
		t.Fatalf("classifyReadPathAgreement() returned %d violations, want 1: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "--fast") || !strings.Contains(got[0].Message, "--shape-only") {
		t.Errorf("violation message %q does not name both cheaper surfaces", got[0].Message)
	}
}

func TestClassifyReadPathAgreement_ReportsANonItemizedSurfaceBlockingMoreThanTheGate(t *testing.T) {
	t.Parallel()

	// `aiwf status` states an error count and no per-finding subcode or
	// severity, so it is compared at the granularity it speaks in. Its
	// rule set is a subset of the gate's with declined judgments already
	// downgraded, so it can never legitimately block where the gate does
	// not.
	gate := substantive("aiwf check")
	status := readPathObservation{Surface: "aiwf status", Blocking: 2}

	got := classifyReadPathAgreement("step 8", gate, []readPathObservation{status})

	if len(got) != 1 {
		t.Fatalf("classifyReadPathAgreement() returned %d violations, want 1: %+v", len(got), got)
	}
	for _, want := range []string{"step 8", "aiwf status", "aiwf check", "2"} {
		if !strings.Contains(got[0].Message, want) {
			t.Errorf("violation message %q does not name %q", got[0].Message, want)
		}
	}
}

func TestClassifyReadPathAgreement_ANonItemizedSurfaceBlockingNoMoreThanTheGateAgrees(t *testing.T) {
	t.Parallel()

	gate := substantive("aiwf check",
		claimAt("G-0001", check.CodeRefsResolve, "unresolved", severityError),
		claimAt("G-0002", check.CodeRefsResolve, "unresolved", severityError))
	status := readPathObservation{Surface: "aiwf status", Blocking: 1}

	if got := classifyReadPathAgreement("step 9", gate, []readPathObservation{status}); got != nil {
		t.Errorf("classifyReadPathAgreement() = %+v, want no violations", got)
	}
}

func TestClassifyReadPathAgreement_OrdersViolationsDeterministically(t *testing.T) {
	t.Parallel()

	gate := substantive("aiwf check",
		claimAt("M-0002", check.CodeACsShape, "duplicate-id", severityError),
		claimAt("G-0001", check.CodeRefsResolve, "unresolved", severityError))
	other := substantive("aiwf check --fast",
		claimAt("M-0002", check.CodeACsShape, "duplicate-id", "warning"),
		claimAt("G-0001", check.CodeRefsResolve, "cross-branch-pending", "warning"))

	first := classifyReadPathAgreement("step 10", gate, []readPathObservation{other})
	if len(first) != 2 {
		t.Fatalf("classifyReadPathAgreement() returned %d violations, want 2: %+v", len(first), first)
	}
	if !strings.Contains(first[0].Message, "G-0001") {
		t.Errorf("violations are not subject-sorted; first is %q", first[0].Message)
	}

	for i := 0; i < 8; i++ {
		if diff := cmp.Diff(first, classifyReadPathAgreement("step 10", gate, []readPathObservation{other})); diff != "" {
			t.Fatalf("classifyReadPathAgreement() is not deterministic (-first +repeat):\n%s", diff)
		}
	}
}

func TestReadPathObservationFrom_DropsDeclinedJudgments(t *testing.T) {
	t.Parallel()

	// A surface reporting unresolved-unverified is saying it did not
	// build the tier that would settle the question. That is a declined
	// judgment, not a competing one, so it must not enter the claim set
	// at all (G-0558).
	obs := readPathObservationFrom("aiwf check --fast", []verbEnvelopeFinding{
		{Code: check.CodeRefsResolve, Subcode: check.SubcodeUnresolvedUnverified, Severity: "warning", EntityID: "G-0001"},
		{Code: check.CodeBodyProseID, Subcode: check.SubcodeUnresolvedUnverified, Severity: "warning", EntityID: "G-0002"},
	})

	if len(obs.Claims) != 0 {
		t.Errorf("readPathObservationFrom() kept %d declined judgments as claims: %+v", len(obs.Claims), obs.Claims)
	}
	if !obs.Itemized {
		t.Error("readPathObservationFrom() produced a non-itemized observation")
	}
	if obs.Blocking != 0 {
		t.Errorf("Blocking = %d, want 0", obs.Blocking)
	}
}

func TestReadPathObservationFrom_KeepsSubstantiveClaimsAndCountsBlockingOnes(t *testing.T) {
	t.Parallel()

	obs := readPathObservationFrom("aiwf check", []verbEnvelopeFinding{
		{Code: check.CodeRefsResolve, Subcode: "cross-branch-local-only", Severity: "error", EntityID: "G-0001"},
		{Code: check.CodeRefsResolve, Subcode: "unresolved", Severity: "error", EntityID: "G-0001"},
		{Code: check.CodeArchiveSweepPending, Severity: "warning"},
	})

	want := map[readPathSubject]readPathClaimSet{
		{EntityID: "G-0001", Code: check.CodeRefsResolve}: {
			{Subcode: "cross-branch-local-only", Severity: "error"}: true,
			{Subcode: "unresolved", Severity: "error"}:              true,
		},
		{EntityID: "", Code: check.CodeArchiveSweepPending}: {
			{Subcode: "", Severity: "warning"}: true,
		},
	}
	if diff := cmp.Diff(want, obs.Claims); diff != "" {
		t.Errorf("readPathObservationFrom() claims mismatch (-want +got):\n%s", diff)
	}
	if obs.Blocking != 2 {
		t.Errorf("Blocking = %d, want 2", obs.Blocking)
	}
}

func TestReadPathObservationFrom_CanonicalizesEntityIDsSoNarrowWidthsNameOneSubject(t *testing.T) {
	t.Parallel()

	// A narrow legacy width names the same entity (ADR-0008) — for a gap
	// the tolerated narrow form is three digits. Two surfaces spelling
	// one id differently must not read as two subjects; that would hide a
	// real disagreement behind a spelling.
	gate := readPathObservationFrom("aiwf check", []verbEnvelopeFinding{
		{Code: check.CodeRefsResolve, Subcode: "unresolved", Severity: "error", EntityID: "G-001"},
	})
	other := readPathObservationFrom("aiwf check --fast", []verbEnvelopeFinding{
		{Code: check.CodeRefsResolve, Subcode: "cross-branch-pending", Severity: "warning", EntityID: "G-0001"},
	})

	if got := classifyReadPathAgreement("step 11", gate, []readPathObservation{other}); len(got) != 1 {
		t.Errorf("classifyReadPathAgreement() returned %d violations, want 1 — narrow and canonical ids must name one subject: %+v", len(got), got)
	}
}

func TestClaimsContradict_SilenceIsContainedInEveryClaimSet(t *testing.T) {
	t.Parallel()

	claims := readPathClaimSet{{Subcode: "unresolved", Severity: severityError}: true}

	if claimsContradict(readPathClaimSet{}, claims) {
		t.Error("an empty claim set contradicts a stated one; silence is not a claim")
	}
	if claimsContradict(claims, readPathClaimSet{}) {
		t.Error("a stated claim set contradicts silence; silence is not a claim")
	}
	if claimsContradict(readPathClaimSet{}, readPathClaimSet{}) {
		t.Error("two silent surfaces contradict each other")
	}
}

func TestReadPathSubjectString_NamesATreeWideSubjectWithoutAnEntity(t *testing.T) {
	t.Parallel()

	if got := (readPathSubject{Code: check.CodeArchiveSweepPending}).String(); got != "(tree-wide)/archive-sweep-pending" {
		t.Errorf("readPathSubject.String() = %q, want the tree-wide form", got)
	}
}

func TestReadPathClaimString_OmitsAnAbsentSubcode(t *testing.T) {
	t.Parallel()

	if got := (readPathClaim{Severity: "warning"}).String(); got != "warning" {
		t.Errorf("readPathClaim.String() = %q, want %q", got, "warning")
	}
}
