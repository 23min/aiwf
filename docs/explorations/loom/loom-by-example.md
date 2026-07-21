---
title: "loom `.lm` by example (claims-only surface)"
status: illustrative companion to the integration proposal (rev. 2)
note: >
  This shows the *claims-only* `.lm` surface — no `does`, no codegen. `.lm` is a
  spec/interface file (like `.mli` / `.h`); the implementation lives in the
  verifier's language (Dafny here) or the host language. Dafny shown is
  illustrative — it demonstrates the lowering, not guaranteed to compile verbatim.
---

# loom `.lm` by example

## 1. A complete `.lm` file (ledger)

```loom
/// Ledger: money moves between accounts without being created or destroyed.
/// Rationale (the "why these claims") lives in aiwf — epic E-0042, ADR-0007.
/// This file references nothing aiwf-specific, so it is usable standalone.
@loom-version 0.1

module ledger {
  knows {
    /// A non-negative amount. The refinement IS the no-negative-balance
    /// invariant — expressed in the type, not as a separate claim.
    Money     :: {x: int | x >= 0}
    AccountId :: {s: string | s.length > 0 and s.length <= 64}

    Account :: { id: AccountId, balance: Money, open: bool }

    pred solvent(a: Account) = a.balance > 0
  }

  relates {
    /// Open an account with a starting balance.
    open_account(id: AccountId, initial: Money) -> Account
      ensures { result.id = id, result.balance = initial, result.open = true }

    /// Move `amount` from one open account to another.
    transfer(from: Account, to: Account, amount: {x: int | x > 0}) -> (Account, Account)
      requires { from.open, to.open, from.balance >= amount }
      ensures  {
        let (f, t) = result;
        f.balance = from.balance - amount,
        t.balance = to.balance + amount,
        f.id = from.id, t.id = to.id,
        f.open = from.open, t.open = to.open,
      }
  }

  shows {
    opens:
      open_account("alice", 100) -> {id: "alice", balance: 100, open: true}

    moves_thirty:
      transfer({id:"alice",balance:100,open:true}, {id:"bob",balance:0,open:true}, 30)
      -> ({id:"alice",balance:70,open:true}, {id:"bob",balance:30,open:true})

    moves_exact_balance:
      transfer({id:"alice",balance:50,open:true}, {id:"bob",balance:0,open:true}, 50)
      -> ({id:"alice",balance:0,open:true}, {id:"bob",balance:50,open:true})
  }

  proves {
    /// Conservation: a transfer never creates or destroys money.
    conservation:
      for-all from: Account, to: Account, amount: {x:int|x>0},
        from.open and to.open and from.balance >= amount =>
          let (f, t) = transfer(from, to, amount);
          from.balance + to.balance = f.balance + t.balance

    /// Opening with balance B yields balance B.
    open_sets_balance:
      for-all id: AccountId, initial: Money,
        open_account(id, initial).balance = initial
  }
}
```

Things to notice:

- **`no_overdrafts` is not a `proves`.** It is absorbed into `Money :: {x | x >= 0}`: you simply cannot construct an `Account` with a negative balance, so the verifier *must* discharge `from.balance - amount >= 0` to typecheck `transfer`'s result. The type does the work a separate claim would.
- **Cross-register coverage** is what `loom check` enforces: every type in `knows` is used, and every operation in `relates` has at least one `shows` example *and* at least one `proves` property. A bare operation with no example and no property is structurally incomplete — that rule is containment, not bookkeeping.
- **No `does`.** The implementation is elsewhere (below).

## 2. How it lowers (illustrative Dafny)

```dafny
// ===== lowered from ledger.lm =====
type Money     = x: int | x >= 0
type AccountId = s: string | 0 < |s| <= 64

datatype Account = Account(id: AccountId, balance: Money, open: bool)

predicate solvent(a: Account) { a.balance > 0 }

// relates.open_account  (function: usable in specs)
function OpenAccount(id: AccountId, initial: Money): Account {
  Account(id, initial, true)
}

// relates.transfer  +  proves.conservation
//   A `proves` that is a property of ONE operation's result lowers to an
//   `ensures` on that operation — no quantifier needed, since a method's
//   pre/post is already universally quantified over its inputs.
method Transfer(from: Account, to: Account, amount: int)
  returns (f: Account, t: Account)
  requires amount > 0
  requires from.open && to.open
  requires from.balance >= amount
  ensures  f.balance == from.balance - amount        // relates.ensures
  ensures  t.balance == to.balance + amount
  ensures  f.id == from.id && t.id == to.id
  ensures  f.open == from.open && t.open == to.open
  ensures  from.balance + to.balance == f.balance + t.balance   // proves.conservation
{
  f := from.(balance := from.balance - amount);  // must prove result is a valid Money (>= 0)
  t := to.(balance := to.balance + amount);       //   <- this discharges no_overdrafts, in the type
}

// proves.open_sets_balance  (property of a function result — here a lemma)
lemma OpenSetsBalance(id: AccountId, initial: Money)
  ensures OpenAccount(id, initial).balance == initial { }
```

**The lowering rules, in one table:**

| `.lm` | lowers to |
|---|---|
| `knows` refinement type | Dafny subset type (`type T = x: B \| P`) |
| `knows` record / sum | `datatype` |
| `knows` `pred` | `predicate` |
| `relates` op + `requires`/`ensures` | `method`/`function` signature + `requires`/`ensures` |
| `proves` about **one** op's result | an `ensures` on that op |
| `proves` relating **several** ops | a `lemma` (the ops must be `function`s to appear in it) |
| `shows` example | a checked example (assertion / test) |
| cross-register coverage | structural check in `loom check` (not lowered) |

## 3. Containment in action — catching a gamed claim

The whole point of the surface is that the *spec* is also LLM-authored, so it can be weakened. Suppose the model writes this instead of the real `conservation`:

```loom
proves {
  /// looks like conservation; isn't.
  conservation:
    for-all from: Account, to: Account, amount: {x:int|x>0},
      from.open and to.open and from.balance >= amount and amount > from.balance =>
        let (f, t) = transfer(from, to, amount);
        from.balance + to.balance = f.balance + t.balance
}
```

`from.balance >= amount and amount > from.balance` is a contradiction, so the antecedent is **never satisfiable** — the implication is vacuously true, and the verifier discharges it instantly. It *passes*. Two lines of defence catch it:

1. **The surface makes it conspicuous.** A linter (or a careful reader) sees `>=` and `>` on the same two terms — a guard that excludes everything. Containment-shaping is exactly this: vacuous shapes are awkward and visible rather than hidden.

2. **The vacuity check catches it mechanically** — and this is the differentiator. Mutate the *conclusion* and re-run the verifier:

   - *Healthy* `conservation`: flip `=` to `≠` (or swap `f.balance`/`t.balance`) and the verifier now **rejects** the mutant. Mutants die → high kill rate → the claim genuinely constrains `transfer`.
   - *Gamed* `conservation`: mutate the conclusion any way you like and it **still verifies**, because a false antecedent makes every conclusion vacuously true. Mutants survive → kill rate ≈ 0 → flagged.

   The finding (lowered-level mutation, lifted back to the `.lm` line):

   ```
   ledger.lm:34: warning weak-claim: proves.conservation survives 6/6 conclusion mutations
     — hint: antecedent may be unsatisfiable (`from.balance >= amount` and `amount > from.balance`); widen the guard
   ```

   That finding goes to wrap-time triage. The gate stays mechanical; its *meaning* is checked and routed to a human. (Related but distinct: an antecedent that is narrow-but-*satisfiable* — say `and from.balance = amount` — isn't vacuous, so mutation still kills it; that under-coverage is what `shows` examples and coverage analysis are for, not the vacuity check.)

## 4. A second example (multi-operation property → a lemma)

```loom
/// Bounded, non-negative counter.
@loom-version 0.1

module counter {
  knows {
    Count :: {n: int | n >= 0 and n <= 100}
  }
  relates {
    zero() -> Count            ensures { result = 0 }
    inc(c: Count) -> Count     requires { c < 100 }  ensures { result = c + 1 }
    dec(c: Count) -> Count     requires { c > 0 }    ensures { result = c - 1 }
  }
  shows {
    starts_zero: zero() -> 0
    inc_five:    inc(5) -> 6
    dec_five:    dec(5) -> 4
  }
  proves {
    /// inc then dec is the identity — relates two operations.
    inc_then_dec_id:
      for-all c: Count when c < 100,
        dec(inc(c)) = c
  }
}
```

```dafny
// ===== lowered from counter.lm =====
type Count = n: int | 0 <= n <= 100

function Zero(): Count { 0 }
function Inc(c: Count): Count requires c < 100 { c + 1 }
function Dec(c: Count): Count requires c > 0  { c - 1 }

// proves.inc_then_dec_id  (relates two ops -> a lemma over functions)
lemma IncThenDecId(c: Count)
  requires c < 100
  ensures Dec(Inc(c)) == c { }
```

`inc`/`dec` lower to `function`s precisely so they can appear inside the lemma; `when c < 100` becomes the lemma's `requires`. (Compare with `conservation` in §2, a single-operation property that became an `ensures` and needed no lemma.)

## 5. What this example is and isn't

- It **is** the claims-only surface: types, signatures + contracts, examples, properties, and the coverage rule. Small. Portable. References nothing aiwf-specific.
- The **prose** ("why these claims, what we're guaranteeing") lives in the doc-comments for local notes and in aiwf's epics/ADRs for the real rationale — and aiwf references *these* claims, never the other way around.
- The **implementation** is the Dafny (or host code) — visible, in a language the LLM is fluent in, checked against the umbrella by the verifier.
- It is **not** an implementation language (no `does`), **not** a Python generator, and the Dafny lowering is **not** something the author writes twice — it is what `.lm` lowers *to*, and the vacuity check runs at that lowered level.