# The gap program — sequencing goml/Go+ toward Idris2 and Lean4

Status: **approved program, execution begins at T1a** (2026-08-30).
This document records the dependency-ordered milestone program that
fills the gaps between goml (plus the shared Go+ core) and the feature
sets of Lean4 and Idris2. It sequences the dependent-typing roadmap's
Stages C–H (`GOALS.md`, "Dependent-typing roadmap") together with the
goml-surface gaps that roadmap does not cover. It is normative for
ordering; each milestone still cuts its own grammar delta and design
notes when it lands.

## The three buckets

- **A — core gaps.** Stages C (dependent matching), D (totality),
  E (automation), F (kernel), G (interop hardening), H (metaprogramming),
  plus codata and `do_dag`. No goml syntax can conjure these.
- **B — surface lag.** Core features goml cannot spell: named `prop`
  declarations (`type P[i nat] prop { … }` shipped v0.156.0; goml has no
  body form), `open` (lexed, rejected), interface declarations.
- **C — front-end gaps.** Unannotated lambdas, list literals, record
  update, `where` clauses, mutual `let rec`, user infix operators,
  monad-generic `let*`, and the interactive tier (holes you can act on,
  `goml fmt`, a real goml source map).

Where goml already meets or beats the comparators, nothing is scheduled:
multiplicity polymorphism (Idris2 lacks it, Lean4 has no linearity),
named instances with property-tested laws, idiomatic committed-Go
output, `select`/channel embedding.

## Decisions taken at program approval

1. **goml gains an interface-declaration form**, reversing the
   mixed-package-only deferral in `spec/goml-design.md` §10. The mixed
   package remains legal; it stops being the only route.
2. **Tuples are deferred to the Stage F tranche.** Sigma/dependent pairs
   arrive with the kernel; non-dependent tuples fall out of the same
   design. Tuple erasure is designed once, not twice.

## Ordering

Critical path to "dependently typed": **C2 → D1 → D2/D3 → F1–F4**.
Everything else is value-ordered. Stage-internal order is strict;
stages interleave: D1 may start after C2; do_dag needs only D1; codata
and E follow D3; G1 may run alongside F; H follows F4.

```
T0 → T1a → T1b → T1c → T1.5
  → C1 → C2 → C3 → C4
        → D1 → DG1 → DG2
          D1 → D2 → D3
            D3 → E1 → E2
            D3 → CO1 → CO2
              E1 → F1 → F2 → F3 → F4     (G1 alongside)
                F3 → G1 → G2 → G3
                F4 → H1 → H2 → H3
```

## The milestones

### T1 — goml surface drift and cheap wins (front end only)

- **T1a**: named `prop` declarations
  (`type InRange (i : Nat) (n : Nat) := prop { And (Le 0 i) (Lt i n) }`
  lowering to the v0.156.0 `.gp` form); interface declarations; list
  literals, `make`, and conversions; record update
  (`{ r with Port = p }`).
- **T1b**: `where` clauses (hoisted package-private lets); mutual
  `let rec … and …`; unannotated lambdas via expected-type threading
  (generalizing the existing `funStringTyped` single-site precedent).
- **T1c**: `open` (export sets loaded at transpile time; file-local
  declarations win; ambiguity is an error; patterns included); user
  infix operators as a goml-only fixity table that always emits
  call-form `.gp`, preserving the differential gate. Full
  notation/mixfix stays in H1.

### T1.5 — monad-generic railway (core) — SHIPPED 2026-08-30, rescoped

Landed as a **rail table, not a structural test**: `std/option.Option`
joined `std/result.Result` as a recognized rail
(`internal/resolve/seg.go isOption`, `railway.go optionLift`), with its
own single-track stage table — Bind for `T → Option[U]`, comma-ok
adapt through `option.Of` for `T → (value, ok)`, Map for `T → U`, no
Tee (hard error), dot segments raw, constructor-literal heads
(`option.Some(16)`, variant struct types) on the rail. Structural
recognition ("anything with Bind/Map/Of") was rejected deliberately:
it grants railway semantics by coincidence of method names; a
per-type opt-in (a registry marker, or the post-F `Monad` class) is
the sound generalization and stays deferred. Also deferred, each
wanting a semantics decision of its own: Kleisli `>=>` over Option and
postfix `?` in Option-returning functions.

### T2 — Stage C: dependent matching

- **C1a — SHIPPED 2026-08-30 (grammar v0.19.0)**: explicit `impossible`
  arms in both surfaces (`case Nil(): impossible` / `| Nil =>
  impossible`). Checked against the same pruning the checker already
  infers (index clash under hypotheses, GADT incompatibility), then
  dropped — generated Go is byte-identical to the omitted-arm form. In
  `.gp` the spelling costs no parser change (a bare `impossible`
  statement is never valid Go); in goml it is a reserved word.
  `std/vec` First/Rest carry the arms. Wildcard/binder/multi-pattern
  impossible arms are guided errors.
- **C1b** dot patterns — deferred behind a design spike: in Go+,
  indices are type-level and constructor fields are runtime data, so
  what a "forced position" *is* needs pinning before a sigil is chosen
  (`.(expr)` vs `=(expr)`). The spike must name the forcing judgment
  (unification-determined field values) and its checker
  (`DecideEqTexts` under arm substitution).
- **C2** guards + literal patterns, both surfaces simultaneously (the
  reserved productions flip on). Coverage policy: nat literals count as
  exhaustive when the decider proves interval exhaustion under
  hypotheses; guarded arms contribute nothing to exhaustiveness in C2.
  Interactive-hole **case split** attaches here.
- **C3** `with` abstraction/views (an extra scrutinee column whose
  unification substitution flows across columns) + decider-checked guard
  coverage.
- **C4** dependent motives for expression matches (annotation-required;
  inference only for the same-head case).

### T3 — Stage D: totality widening, and do_dag

- **D1** `total` over enum-typed params/results (the nat-only surface
  gate widens to core-representable types). Names the **purity
  predicate** (elaborable + `CheckTotal`) with a marker — do_dag's
  precondition.
- **DG1** `do_dag … end`: bindings-only DAG, purity required,
  deterministic topological emission (the checked assertion).
- **DG2** concurrent scheduling (per-block opt-in; structured
  join; sequential-vs-concurrent differential gate under the race
  detector).
- **D2** author-supplied measures (`decreases`; strict decrease decided
  under path hypotheses; one path-fact walker shared with the
  subtraction obligations).
- **D3** mutual recursion (SCCs over the existing call-graph walk;
  shared measure per SCC; SCCs may not span packages).

### T4 — Stage E: automation

- **E1** law-driven rewriting: rules extracted from class laws of shape
  `Eq(lhs, rhs)` whose sides elaborate into the core fragment; oriented
  left-to-right under var-subset and size conditions; fuel-bounded.
  **Trust policy**: obligations discharged via property-tested (unproved)
  laws are *validated*, not proved — the boundary guard stays and the
  diagnostic names the assumed laws; `//goplus:law trusted` removes the
  guard auditably; kernel-proved laws (F3+) upgrade automatically.
- **E2** `auto` proof parameters: depth-bounded backward chaining over
  hypotheses, named-prop unfolding, conjunction, the decider, and E1 —
  generalizing instance search's determinism discipline. Interactive-hole
  **expression/witness search** attaches here (same engine).

### T5 — codata

- **CO1** `codata` declarations (observer-style dual of enum), syntactic
  guardedness (`CheckProductive`), erasure to memoized thunks.
- **CO2** guardedness through match/if; inductive-consumes-codata rules;
  do_dag purity extended to productive forcing.

### T6 — Stage F: the kernel

**Decision: a new `internal/kernel` package.** Small, dependency-free,
auditable — checkable without trusting the elaborator. `internal/core`
is not rebuilt; it becomes the untrusted automation layer that produces
kernel terms.

- **F1** kernel terms (de Bruijn; Pi/Sigma/lambda/pairs/Sort),
  predicative `Type 0..n` with cumulativity, bidirectional checker, NbE,
  definitional equality; a textual kernel format and `goplus kernel
  check` so the milestone gates without surface wiring.
- **F2** inductive families, strict positivity, eliminators/recursors,
  intensional `Id`/`refl`/J.
- **F3** the elaboration bridge (outside the kernel): enums/GADTs →
  families, total funcs → recursors/Acc-recursion, props → kernel types;
  **decider facts enter as labeled axioms** (an auditable ledger);
  `//goplus:kernel` per-package opt-in. Gate: `std/vec` kernel-checks
  with its axiom ledger asserted.
- **F4** surface Pi/Sigma under the kernel pragma; dependent pairs (the
  deferred tuples); minimal `proof` expressions; C4 motives re-founded.
  The threshold milestone: the "dependently typed" claim becomes
  earnable. Gate: `std/vec` `reverse` with a J-proved length equation.

### T7 — Stage G: interop and hardening

- **G1** LSP goals/hover/completion over C/D/E; the **two-hop source
  map** (goml → `.gp` → Go) emitted from lowering and consumed by the
  goml LSP; `goml fmt`; a dedicated editor grammar.
- **G2** ABI/erasure/marker spec frozen (versioned marker format,
  proof-irrelevant hashing, what-plain-Go-sees per exported API).
- **G3** uniform resource limits, incremental checking, parser and
  kernel fuzzing, `goplus audit` over every labeled trust point.

### T8 — Stage H: metaprogramming

- **H1** syntax quotation over the extension AST + hygienic macros
  (total Go+ functions over reified syntax) expanded in a new pass-1
  phase; full notation/mixfix.
- **H2** elaborator reflection: goals, hypotheses, unification, search,
  and simp exposed to macros that produce kernel terms; the kernel
  re-checks everything (tactics without trusting tactics).
- **H3** interactive goal actions end-to-end: case split + search +
  apply + refine as LSP code actions over G1's map, identical from goml.

## Open decisions, and when they must close

1. `.gp` dot-pattern sigil — before C1's grammar delta.
2. `with` as a dedicated clause vs multi-column sugar — C3 (dedicated
   clause recommended).
3. E1 default trust tier — guard-on default recommended; confirm at E1.
4. Certificate-producing decider (empties F3's axiom ledger) — decide
   before G2 freezes the spec.
5. A Prop sort with proof irrelevance — decide (even if "never") by G2.
6. do_dag concurrency spelling and panic semantics — DG2.

## The gate, every milestone

`go generate ./...` clean; `go tool goplus gen -check .` clean;
`go test ./...` including the strict Godog suite, the goml differential
gates, fork equivalence, and law tests, race+vet. New features add
positive and negative scenarios (exact diagnostic text) and a goml twin;
named forcing consumers (`std/vec`, `std/smt`, `std/algebra`) are
reauthored where the milestone says so. Version lines advance per repo
convention; nothing is tagged or pushed without an explicit go-ahead.
