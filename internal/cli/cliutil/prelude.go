package cliutil

import "context"

// ResolvePrelude resolves the repo root then the actor for a mutating verb,
// emitting the shared "aiwf <verb>: <err>" usage error to stderr and
// returning ExitUsage on either failure. label is the verb's error prefix
// (e.g. "aiwf promote", "aiwf set-area"). On success it returns the resolved
// root and actor with ok == true; on failure rootDir and actorStr are empty
// and the caller returns code without using them:
//
//	rootDir, actorStr, code, ok := cliutil.ResolvePrelude("aiwf promote", root, actor)
//	if !ok {
//		return code
//	}
//
// This single-sources the ResolveRoot → ResolveActor prelude and its
// identical usage-error arm that the mutating verbs otherwise hand-roll.
// Verbs whose failure path emits a structured outcome envelope (honoring
// --format=json) use ResolvePreludeEnvelope instead.
func ResolvePrelude(label, root, actor string) (rootDir, actorStr string, code int, ok bool) {
	rootDir, err := ResolveRoot(root)
	if err != nil { //coverage:ignore ResolveRoot only fails on a broken cwd (filepath.Abs / os.Getwd) or missing aiwf.yaml + a non-existent --root path; not deterministically reproducible.
		Errorf("%s: %v\n", label, err)
		return "", "", ExitUsage, false
	}
	actorStr, err = ResolveActor(actor, rootDir)
	if err != nil {
		Errorf("%s: %v\n", label, err)
		return "", "", ExitUsage, false
	}
	return rootDir, actorStr, ExitOK, true
}

// ResolvePreludeEnvelope is ResolvePrelude for verbs whose failure path emits
// a structured outcome envelope via FinishVerbOutcome (honoring --format=json)
// rather than a plain stderr line. It preserves the raw root on a ResolveRoot
// failure (rootDir is not yet resolved) and the resolved rootDir on a
// ResolveActor failure, matching the hand-rolled arms it replaces.
func ResolvePreludeEnvelope(ctx context.Context, label, root, actor string, out OutputFormat) (rootDir, actorStr string, code int, ok bool) {
	rootDir, err := ResolveRoot(root)
	if err != nil { //coverage:ignore ResolveRoot only fails on a broken cwd or missing aiwf.yaml + a non-existent --root path; not deterministically reproducible.
		code, _ = FinishVerbOutcome(ctx, root, label, nil, err, out)
		return "", "", code, false
	}
	actorStr, err = ResolveActor(actor, rootDir)
	if err != nil {
		code, _ = FinishVerbOutcome(ctx, rootDir, label, nil, err, out)
		return "", "", code, false
	}
	return rootDir, actorStr, ExitOK, true
}
