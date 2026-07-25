# GoForge Wave 1 — feature backlog (new / nice-to-have)

_Drafted 2026-07-24. Companion to
[`GOFORGE_CANDIDATES_WAVE1.md`](GOFORGE_CANDIDATES_WAVE1.md), which owns parity
and release status. **This file owns forward-looking work**: Go+-native
additions beyond upstream parity ("useful additions" the definition-of-done
asks for), and the parity slices intentionally deferred for later._

Each item notes the **Go+ idiom** it leans on and a rough **effort** (S/M/L).
Nothing here blocks a first parity release; these are the "later" list.

## Legend

- **Parity-deferred** — a compatibility slice deliberately out of the current
  cut; eventual parity, not a new feature.
- **Additive** — a new Go+ API with no upstream equivalent; makes invalid states
  unrepresentable, effects explicit, ownership clear, or partiality exhaustive.
- **Perf** — an optimization past the point of diminishing returns; ledger the
  target and move on if the win is not there.

---

## Viper (`goforge.dev/viper`)

Parity is broad (192/192 declarations); the semantic core is `std/config`.

- **Additive — provenance as exhaustive states (M).** Model source precedence
  (default → file → env → flag → override → remote) as an exhaustive Go+ sum so
  "where did this value come from" is a matched value, not a string. Idiom:
  exhaustive enums + retained witnesses.
- **Additive — typed keys / schema projection codegen (M).** Generate typed
  accessors from a declared schema so `cfg.Port` replaces `GetInt("port")`;
  missing required keys become a compile-time or load-time proof. Idiom:
  indexed values + presence witnesses.
- **Additive — reload as an explicit effect stream (M).** Expose watch/reload as
  an ordered event capability rather than a callback, so immutable reads and
  reload effects never interleave silently. Idiom: capabilities + owned effects.
- **Additive — `std/config` capability-scoped source loading (L).** The open
  stdlib obligation; best landed with a *second* consumer (see Wave 2 `direnv`).
- **Parity-deferred — differential concurrency + filesystem/provider integration
  suites (M).** Race-covered reload under concurrent readers; real afero/remote
  provider integration matrices.
- **Perf (diminishing returns).** Immutable-snapshot lookup is already ~8.6×.
  Extend paired benches to discovery/decode/reload/provider only where a real
  hot path exists; otherwise ledger and stop.

## Validator (`goforge.dev/validator`)

Parity is **complete** (106/106 declarations, 210/210 behaviors). Backlog is
almost entirely additive + a bounded set of failure-path perf items.

- **Additive — proof-carrying `Validated[T,P]` threaded through app code (M).**
  Let a validated value carry its predicate witness so downstream code needn't
  re-check; erase to plain `T` at Go boundaries. Idiom: predicate-indexed types.
- **Additive — derive validators from types via codegen (L).** Generate a rule
  set from a struct definition, removing stringly-typed tags on hot paths while
  keeping the tag API for compatibility. Idiom: refinements + generation.
- **Additive — localized message registry laws (S).** `Failure.Descriptor()`
  exists; add lawful, translation-neutral code/param descriptors with a tested
  round-trip to locale catalogs.
- **Perf (bounded — real target misses).** Four failure paths miss 2×/50%:
  `VarWithValue` failure (1.34×), selective struct failure (2.02× / partial,
  1.60× / except), filtered failure (0.93×), cached `Var`/alias failure
  (1.22×/1.16×). Common cause: the `ValidationErrors` compatibility boundary
  materializes eagerly. Backlog: design compact, proof-bearing failures that
  materialize lazily without weakening the exact-error contract. Idiom: owned
  failure trees + a non-materializing boolean runner (already prototyped).

## GJSON (`goforge.dev/gjson`)

Parity is broad (45/45 declarations, 1,301/1,301 pinned path pairs); malformed-
path differential parity was hardened 2026-07-24 (see Wave 1 gjson ledger).

- **Parity-deferred — permissive compatibility lookup semantics (M).** Keep the
  permissive compat evaluator byte-exact while the strict validated document is
  the new API. The **pipe-into-container** malformed class is document-engine-
  dependent and intentionally out of scope (excluded by
  `dynamicMalformedContainerLiteral` in `laws_fuzz_test.go`).
- **Additive — compiled-path handle as a first-class API (S).** Surface the
  immutable compiled path/`Registry` as the recommended API (parse once, query
  many); today it is additive-but-secondary.
- **Additive — schema-indexed typed documents (M).** Promote validated indexed
  documents so a lookup's result type is known from the schema. Idiom: indexed
  types + exhaustive lookup sums.
- **Additive — owned/borrowed JSON-lines streaming (M).** Typed streaming with
  explicit ownership so borrowed slices can't outlive their buffer.
- **Additive — extend `std/pathquery` relation algebra (S).** `Relation`/
  `Relate` exist; add order/complement laws general enough for a second consumer
  (Wave 2 `yq`).
- **Perf (diminishing returns).** One-shot wildcard/projection/query baselines
  are documented (ns/alloc numbers in the ledger); they trail upstream on raw
  one-shot but win big with path caching. Optimize only the paths a real
  consumer hits; ledger the rest.

## Lo (`goforge.dev/lo`)

47/651 declarations are compatible; the remainder is **design-gated**, not
merely unwritten. Manifest status split: 205 iterator (`it`) declarations await
the `std/iter` design; 388 are "outside compatibility tier 1"; 6 need an
explicit aliasing API; 5 need a cancellation/ordering contract.

- **Additive / blocking — land `std/iter` (L, highest leverage).** A law-tested
  iterator algebra (fallible folds, stable grouping, ordered/unordered
  parallelism with cancellation, opt-in fusion) over Go's `iter.Seq`. This
  **unblocks 205 deferred declarations** and is the one genuinely-missing `std`
  core. Idiom: lawful lazy sequences + effect-typed parallelism.
- **Parity-deferred — 388 outside-tier-1 declarations (L).** Re-triage against
  `std/iter` once it exists; many collapse into iterator combinators.
- **Additive — total-by-default operations (M).** Make the non-panicking,
  Option/Result-returning form the default and keep the panicking upstream form
  as an explicit compat opt-in. Idiom: exhaustive partiality (`Search[T]`,
  `Option`, `Result`) — already started with `Locate`/`LocateLast`.
- **Additive — NonEmpty/sized-vector fusion (M).** Thread `std/nonempty` and
  `std/vec` evidence through pipelines so reductions on non-empty inputs need no
  empty case, and destination-buffer fusion removes intermediate allocation
  (`FilterMapInto` already ~2.65×). Idiom: non-empty evidence + sized indices.
- **Parity-deferred — aliasing API (6) and cancellation/ordering contract (5)
  (M).** Design the explicit ordered-parallel result and mutation-aliasing
  contracts before implementing.

## Chi (`goforge.dev/chi`)

Parity is broad (127/127 declarations); the semantic core is `std/http/route`.

- **Additive — typed route parameters (M).** Compile-time-known parameter names
  per pattern, so `URLParam` misttypes are caught at build. Idiom: pattern-
  indexed parameter sets.
- **Additive — owned request-body states (M).** Model the body as a linear
  resource (unread → read → closed) so double-reads/leaks are unrepresentable.
  Idiom: ownership typestates.
- **Additive — capability-composition laws for middleware (M).** Make middleware
  a typed capability so ordering/duplication constraints are checkable, and the
  global-before-route rule is structural, not conventional.
- **Additive — immutable route-set snapshots as the primary API (S).** Surface
  the deterministic compiled route set + typed `MatchOutcome` as the recommended
  API; keep the mutable builder for compatibility.
- **Parity-deferred — differential verification of precedence, ambiguous
  registration, middleware order, redirects, compression, throttling, auth
  helpers, profiler, and platform-sensitive handlers (M).**
- **Perf (diminishing returns).** Exact-route dispatch is ~5.2×; named-param is
  1.26× with 50% fewer allocs; the content-type compat path ties upstream and
  needs a typed/fused policy. Extend the paired gate to dynamic/regex/wildcard/
  middleware/mounts only where a real hot path exists.

---

## Cross-cutting backlog (all five)

- **Release + site pipeline (per project).** Nothing is tagged/published yet.
  Each needs: remove `replace`, use released deps, rerun all gates, publish
  `goforge.dev/<module>`, and deploy both the goforge.dev main page **and** the
  per-tool page (`/viper/`, `/validator/`, `/gjson/`, `/lo/`, `/chi/`). Requires
  an explicit release go-ahead.
- **Property-based law suites.** Each project claims semantic laws; make each a
  checked property (`testing/quick` / `pgregory.net/rapid`), cupel-seeded from an
  English spec where practical. Several `laws_fuzz_test.go` / `*_gp_laws_test.go`
  files exist; broaden coverage of the additive APIs above.
- **`std` promotion.** Every additive `std` extension needs a *second*
  independent consumer to promote; Wave 2 is designed to supply them
  (`std/config`←direnv, `std/pathquery`←yq, `std/workflow`←go-task,
  `std/parsec`←sqlc).
