---
id: D-0013
title: Surface Coded verb refusals as a status:error envelope object, exit 1
status: accepted
relates_to:
    - M-0143
    - E-0036
    - ADR-0012
---
## Context

E-0036's goal names the `--format=json` envelope as a consumer of the structured `Coded` codes — *"errors.As-able for the JSON envelope"* — and M-0143 keeps that promise. But the wiring is not where the milestone title suggests. Every `Coded` error (`FSMTransitionError`, `AuthorizeKindError`, the M-0139 cancel errors) originates in a **mutating verb** (`promote`, `cancel`, `authorize`). Those verbs have no `--format=json` flag (it lives only on the read verbs — check/show/status/list/history/schema/template/render/contract-verify) and route through the shared `cliutil.FinishVerb` / `DecorateAndFinish`, which on *any* error prints `label: <err>` to **stderr as plain text** and returns `ExitUsage` (2). There is no JSON path for a coded refusal to flow into, and no envelope `error` slot. Three coupled questions follow: which verbs get the flag, how the refusal is represented, and what exit code it carries.

## Resolution

**Flag scope — uniform (A2).** `--format`/`--pretty` are added to **every mutating verb** via a shared `cliutil.AddFormatFlags` registrar, threaded through each verb's `Run` into the single `FinishVerb`/`DecorateAndFinish` chokepoint. A `--format` flag behaves the same everywhere rather than working on `promote` but erroring on `add`. The centralized handler makes the per-verb cost a one-line registration plus passing the value through; the envelope-building lives in one place.

**Representation — structured error object (B-a).** The `render.Envelope` gains an additive `error` slot: `Error *EnvelopeError json:"error,omitempty"` where `EnvelopeError = {Code string json:"code,omitempty"; Message string json:"message"}`. A coded refusal emits `status: "error"` with `error: {code, message}`; `code` comes from `entity.Code(err)` (`errors.As`, never message-parsing), `message` from `err.Error()`. This mirrors the gRPC / JSON-RPC error-object convention, keeps `findings[]` pure (a verb refusal is not a path/line/severity validation finding), and is fully additive — existing consumers that read `tool`/`status`/`findings`/`result` are unaffected.

**Exit code — unify legality with check-time (C2).** A coded (legality) refusal returns `ExitFindings` (1), not `ExitUsage` (2). `FinishVerb` switches on `entity.Code(err)`: a `Coded` error → exit 1; a non-`Coded` verb error stays exit 2; internal failures (nil result, apply error) stay `ExitInternal` (3). This unifies the exit-code semantics of a legality violation across surfaces — the same `fsm-transition-illegal` already exits 1 when caught at check-time (`aiwf check`), so the verb-time refusal matching it is the coherent outcome. The exit code is format-independent (it changes in text mode too); the M-0125 negative driver requires only a non-zero exit, so it stays green.

**Success path.** With `--format=json`, a successful mutating verb emits `status: ok` (or `findings` when warnings rode along) with `result: {subject}` (the plan subject `FinishVerb` already prints in text mode) — uniform across all mutating verbs.

**Boundary.** The envelope is emitted for the verb's *outcome* on the `FinishVerb` path (success / warnings / coded refusal / verb error / apply error). Pre-dispatch flag-usage errors (mutually-exclusive flags, missing positional args — caught before the verb runs) remain plain-text `ExitUsage` (2); they are CLI-usage errors, not verb outcomes, and are out of scope for the envelope.

## Alternatives considered

1. **Representation (b): reuse `findings[]` (rejected).** `findings[]` already carries `code`, but a verb-time refusal is a single terminal error, not a cross-cutting validation finding with path/line/severity. Promoting it into `findings[]` muddies the documented "findings vs result" distinction.
2. **Representation (c): a bare top-level `code` (rejected).** No home for the message; the envelope has no top-level message field. Less structured than the error object.
3. **Flag scope (A1): only the coded-emitting verbs (rejected).** Smaller, but `aiwf promote --format=json` working while `aiwf add --format=json` errors "unknown flag" is a CLI inconsistency a user cannot predict. The shared chokepoint makes uniform nearly as cheap.
4. **Exit code (C1): keep `ExitUsage` (2) (rejected).** Non-breaking, but "usage" is semantically wrong — the command was well-formed; the *action* was illegal given state. C2 unifies with the check-time exit code for the same violation class.

## Consequences

- All mutating verbs gain `--format`/`--pretty` via `cliutil.AddFormatFlags`; a policy test pins the uniform rollout (a new mutating verb without the flag fails CI).
- `render.Envelope` gains an `error` slot (additive, `omitempty`); the `--format=json` schema stays backward-compatible.
- **Behavior change:** a verb-time legality refusal now exits `1` (was `2`), in both text and JSON modes. Documented in CHANGELOG. The negative driver (non-zero) and the check-time exit-1 semantics both remain consistent.
- The code is extracted via `entity.Code` (`errors.As`) — the same descriptor surface ADR-0012 / D-0011 established — so M-0139's cancel codes and any future `Coded` error surface in the envelope automatically.
