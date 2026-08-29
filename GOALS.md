# Go+ rewrite goals

This is the authoritative index for `/goals/01` through `/goals/10`. Rankings
were refreshed on 2026-07-20. GitHub stars are recorded only as a reproducible
adoption proxy; compatibility imports and downstream consumers are stronger
evidence when available.

The ranking weights three questions in order: likely users helped, reusable
Go+ standard-library value, and whether Go+ can give the rewrite materially
better semantics than ordinary Go. Every source project is MIT-licensed. A
rewrite must retain the upstream copyright and license, identify its pinned
compatibility baseline, and distinguish copied/adapted code from independently
structured code.

## Candidate evidence snapshot

| Goal | Upstream | Adoption proxy | License |
|---|---|---:|---|
| 01 | [shopspring/decimal](https://github.com/shopspring/decimal) | 7.5k stars | MIT |
| 02 | [go-playground/validator](https://github.com/go-playground/validator) | 20.1k stars | MIT |
| 03 | [samber/lo](https://github.com/samber/lo) | 21.4k stars | MIT |
| 04 | [spf13/viper](https://github.com/spf13/viper) | 30.4k stars | MIT |
| 05 | [go-chi/chi](https://github.com/go-chi/chi) | 22.6k stars | MIT |
| 06 | [expr-lang/expr](https://github.com/expr-lang/expr) | 8.0k stars | MIT |
| 07 | [tidwall/gjson](https://github.com/tidwall/gjson) | 15.6k stars | MIT |
| 08 | [robfig/cron](https://github.com/robfig/cron) | 14.2k stars | MIT |
| 09 | [go-resty/resty](https://github.com/go-resty/resty) | 11.8k stars | MIT |
| 10 | [alecthomas/participle](https://github.com/alecthomas/participle) | 3.9k stars | MIT |

Counts are rounded snapshots, not ranking scores. The order deliberately moves
semantic forcing cases above some larger repositories.

## Execution policy

Across the rewrite program, **feature parity and compatibility are the hard
priorities**. A family is considered complete for release only after it has:

- a pinned-upstream-compatible declaration inventory,
- compatible and sound behavior on the differential corpus,
- and generation and interoperability gates at baseline compatibility.

Only after a family is parity-green do we prioritize NFR work on
throughput and heap-allocation reduction. We keep that work focused on the
highest-yield workloads and stop it once additional NFR tuning only yields
diminishing returns or risks semantic/coverage regressions. Exact throughput and
allocation parity is acceptable as a stop condition for an individual family
when further optimization is low-yield, while compiler/std-ecosystem improvements
that materially improve semantics, guarantees, or problem expression quality remain
in scope.

In practical terms, once a family is parity-green we should first use compiler and
stdlib constructs to improve encoding quality, semantic clarity, and user-facing
problem formulation. If those improvements do not unlock material runtime wins, or
if further micro-optimizing yields only incremental/noisy gains, exact throughput
and allocation parity is an acceptable release state for that family.

The practical sequence remains:
`feature parity and compatibility → NFR pass (throughput + allocations) → stable
termination at material parity or parity-safe exact parity`.

For each family, this means the sequence is:
feature parity first → NFR pass → release when either both goals are green or
parity is green and NFR has reached a materially stable point.

## Goals

### `/goals/01-decimal` - `shopspring/decimal` -> `std/decimal` (complete)

- **Why first:** Exact money and measurement arithmetic has broad application,
  and global/default rounding policies are a good target for explicit sums.
- **Go+ result:** Immutable decimal values, explicit rounding modes and division
  outcomes, refinement-checked precision/scale, and `Fixed[p]` arithmetic.
- **Dependent pressure:** Index arithmetic (`p+q`), cross-package index recovery,
  and retained runtime witnesses at erased Go boundaries.
- **Gate:** Compatibility/differential corpus, 100% statement coverage, race and
  fuzz testing, generated-source check, and performance/allocation parity.
- **Status:** implemented in `std/decimal`.

### `/goals/02-validator` - `go-playground/validator` -> `std/validate` (complete)

- **Why second:** Roughly 20k GitHub stars, widespread web/API use, and the
  default validation engine used by Gin make successful migration unusually
  valuable. Its tag-and-reflection API currently forgets validation after the
  call returns.
- **Destination:** A small Go+-authored `std/validate` core with a compatibility
  adapter in a GoForge module. Do not put translations, every baked-in format,
  or reflection cache policy into `std` initially.
- **Semantic rewrite:** `Rule[T]`, exhaustive structured failure trees, rule
  composition with laws, typed field paths, and `Validated[T, P]` witnesses.
  Existing struct tags remain an adapter, not the semantic center.
- **Dependent pressure:** Predicate-indexed witnesses, conjunction of predicates,
  proof-preserving `Map`, and safe erasure/runtime revalidation for ordinary Go.
- **Gate:** Pin validator v10; publish an explicit supported-tag matrix; match its
  success/failure corpus and namespace behavior; integrate with `std/config`;
  prove composition laws; benchmark cached success and failure paths; compile
  cross-package accept/reject examples for `Validated[T, P]`.
- **Status:** implemented in `std/validate`, with the reflection/tag migration
  adapter in sibling module `goforge.dev/validator`. Paired core workloads are
  3.6x–7.8x faster than v10.30.3; allocating failure paths use 1 vs 3 and 2 vs
  7 allocations, while both success paths remain allocation-free.

### `/goals/03-lo` - `samber/lo` -> GoForge collection algebra (complete)

- **Why third:** Roughly 21k stars and a very broad generic-programming audience.
  A literal helper-for-helper clone would bloat `std`, but a rewrite can expose
  the algebra hidden behind the popular API.
- **Destination:** A GoForge compatibility package. Promote only independently
  reused primitives to `std/option`, `std/result`, `std/algebra`, or a narrowly
  scoped `std/nonempty` package.
- **Semantic rewrite:** Total variants for partial helpers, explicit `Option` and
  `Result`, lawful folds/traversals, stable versus unordered operations, and no
  panic-based indexing in the semantic API.
- **Dependent pressure:** `NonEmpty[T]`, length-indexed arrays/vectors, shape-
  preserving map, equal-length zip, and bounds-proven indexing.
- **Gate:** API inventory and parity manifest, differential/property corpus,
  order and aliasing guarantees, zero-surprise allocation benchmarks, and at
  least two consumers before any new standard package is promoted.
- **Status:** implemented as sibling module `goforge.dev/lo`, with 47 compatible
  declarations and an explicit 651-declaration upstream inventory. The fused
  `FilterMapInto` semantic API is at least 2.51x faster than upstream
  `FilterMap` in the recorded five-run gate and uses 0 instead of 1 allocation;
  compatibility `Map` retains upstream allocation parity. `std/nonempty` and
  the dependent `Vec[T,n]`/`Fin[n]` extensions have two independent consumers.

### `/goals/04-viper` - `spf13/viper` -> typed GoForge config over `std/config` (complete)

- **Why fourth:** Roughly 30k stars and enormous downstream reach. It is ranked
  below smaller projects because cloning its mutable global/reflection-heavy API
  would preserve the semantics Go+ should improve.
- **Destination:** A GoForge compatibility facade built on `std/config`; only
  source precedence, provenance, decoding contracts, and schema primitives
  belong in the standard library.
- **Semantic rewrite:** Immutable snapshots, exhaustive source provenance,
  deterministic precedence, typed keys, explicit missing/decode errors, and a
  separate effectful reload stream.
- **Dependent pressure:** `Config[S]` schema indices, `Key[S, T]`, proof that
  required keys exist after validation, and typed subset projection.
- **Gate:** Pin a Viper release and precedence behavior; compatibility tests for
  defaults/files/env/flags/aliases; deterministic merge laws; config integration
  tests; race tests for reload; migration examples without package globals.
- **Status:** implemented as sibling module `goforge.dev/viper`, pinned to
  v1.21.0 (`394040caccbdf5821fa6839386a35f0fb1b1ee9e`). Its reproducible
  192-declaration inventory marks 63 high-use declarations compatible and 129
  explicitly deferred. `std/config` now supplies provenance-retaining
  `Snapshot[s]`, `Key[T,s]`, required-key evidence, and `Subset[s,sub]` typed
  projection; a race-tested `ReloadStream` publishes ordered immutable success
  or failure events outside the read path. The immutable snapshot read is
  8.62x faster than upstream `GetString` in the recorded five-run gate and uses
  0 rather than 3 allocations.

### `/goals/05-chi` - `go-chi/chi` -> typed GoForge router over `std/http` (complete)

- **Why fifth:** Roughly 22.5k stars, no external dependencies, and direct
  `net/http` compatibility give it a large audience and a clean rewrite seam.
- **Destination:** GoForge first. Donate route-pattern parsing and typed parameter
  primitives to `std/http` only after a second router/server consumer appears.
- **Semantic rewrite:** An immutable route tree, exhaustive match outcomes,
  explicit middleware capabilities, conflict detection, and generated OpenAPI-
  usable route metadata without runtime tree introspection.
- **Dependent pressure:** singleton route patterns, parameter environments,
  handlers whose argument record is derived from the pattern, and route-set
  indices preventing duplicate/conflicting registration.
- **Gate:** Chi routing/middleware corpus, `net/http` interoperability, ambiguity
  diagnostics, fuzzed pattern parsing, benchmark parity, and compile-time tests
  for missing/extra route parameters.
- **Status:** implemented as sibling module `goforge.dev/chi`, pinned to v5.3.1
  (`8b258c7bb28f97a5f2a856ff7ef962578fec9215`). Its reproducible
  178-declaration inventory marks 53 root declarations compatible, 26 other
  declarations deferred, and 99 middleware declarations deferred to a
  capability-typed tier. `std/http/route` now supplies `Pattern[p]`,
  `Request[p]`, `ParamKey[T,p]`, sealed `Handler[p]`, indexed route sets, and
  capability-indexed middleware, with retained erased-boundary witnesses;
  `goforge.dev/chi` and `std/http.RouteHandler` are its two production
  consumers. The immutable exact-route snapshot is 5.63x faster than upstream
  Chi in the recorded five-run gate and uses 0 rather than 2 allocations.

### `/goals/06-expr` - `expr-lang/expr` -> typed GoForge expression engine

- **Why sixth:** Roughly 8k stars, but the strongest direct forcing case for
  GADTs and existential types. Dynamic expression evaluation benefits greatly
  from making result types and effects explicit.
- **Destination:** GoForge package; reusable typed-AST/elaboration machinery is
  compiler infrastructure, not a general-purpose standard package.
- **Semantic rewrite:** `Expr[T]` GADT, exhaustive typed bytecode instructions,
  explicit compile versus runtime failures, controlled effects, and no `any`
  in the checked evaluation path.
- **Dependent pressure:** existential `SomeExpr` returned by parsing, equality
  witnesses for type refinement, typed environments, and length-indexed stack
  effects for bytecode verification.
- **Gate:** Language/conformance matrix, differential parser/evaluator corpus,
  rejection corpus, bytecode stack-safety proof tests, fuzzing, and competitive
  compile/evaluate benchmarks.
- **Status:** implemented as sibling module `goforge.dev/expr`, pinned to
  v1.17.8 (`21f4f0575591d7097e576edd7983daf23c1e4afe`). Its reproducible
  inventory records all 617 exported upstream declarations and its language
  matrix explicitly bounds the checked tier. The Go+-authored core supplies
  `Expr[T]`, finite existential `SomeExpr`, explicit effects/failures, typed
  environments, `Instruction[input,output]`, `Stack[n]`, and `Eq[n,m]` depth
  transport. Imported positive/negative fixtures prove bytecode composition,
  reject underflow and wrong instruction effects, and reject false equality
  witnesses. Differential, rejection, fuzz, race, generation, root/std, and
  allocation gates pass. In the recorded five-run gate compilation is at least
  4.72x faster with 59.4% fewer allocations, while the typed scalar VM is at
  least 2.81x faster and uses 0 rather than 3 allocations; the `map[string]any`
  migration facade also uses zero allocations.
- **Released** as `goforge.dev/expr` v0.1.0 (repository `brain-fuel/expr`,
  vanity path live, `go list -m goforge.dev/expr@v0.1.0` resolves through the
  proxy, tool page published). Build, vet, tests, `-race`, and `gen -check`
  all pass under goplus v0.157.0; `typed/typed_gp.go` regenerated to vintage
  v0.28.0, a header-only change with the generated Go byte-identical.
- **Why this is still not `(complete)`:** workflow step 6 requires at least
  one real GoForge consumer, and this goal has none — the status above
  records cross-module fixtures and gates, where each completed goal names an
  actual consuming project. Completion is now that one item.

### `/goals/07-gjson` - `tidwall/gjson` -> schema-aware GoForge JSON paths

- **Why seventh:** Roughly 15.5k stars and pervasive JSON use. Raw dynamic lookup
  remains available, while schema-aware callers should not repeatedly inspect
  result kinds and presence flags.
- **Destination:** GoForge first; a small `std/serde/path` abstraction is eligible
  only when CBOR/DAG-CBOR and another format share its laws.
- **Semantic rewrite:** Parsed immutable paths, exhaustive missing/null/value/error
  results, lossless number policy, streaming traversal, and an explicit modifier
  registry rather than ambient globals.
- **Dependent pressure:** `Path[S, T]`, presence-indexed results, existential
  paths for runtime strings, and schema-preserving path composition.
- **Gate:** GJSON path compatibility matrix and corpus, malformed-input fuzzing,
  zero-copy lifetime documentation/tests, allocation and throughput parity, and
  typed JSON/CBOR consumer demonstrations.
- **Status:** implemented as sibling module `goforge.dev/gjson`, pinned to
  v1.19.0 (`0fac2c9aa6eb5d5564bfaaaad513ce0d5d2314de`). Its reproducible
  inventory records all 45 exported declarations: 26 compatible, three global
  modifier declarations replaced by immutable `Registry`, and 16 explicitly
  deferred. Validated immutable documents retain lossless number text and an
  index of borrowed values; byte input is owned, JSON-lines traversal streams,
  and escaped strings decode into reusable caller storage. The Go+-authored
  core supplies `Path[S,T]`, `TypedDocument[S,D]`, `Lookup[T,p]`, finite
  existential paths, presence-only elimination, and schema-preserving
  composition. Imported fixtures accept a matched path/document pair and
  reject wrong path schemas, wrong document schemas, and use of missing
  evidence as present; erased Go boundaries recheck retained schema IDs. One
  `Path[S,int]` is consumed by both JSON and `std/cbor`. Differential and
  malformed fuzzing, zero-copy lifetime/ownership, race, generation, root/std,
  and allocation gates pass. In the recorded five-run gate the schema-typed
  borrowed-string query is at least 2.31x faster than GJSON v1.19.0 and uses
  0 rather than 1 allocation (100% fewer). The shipped module is
  `goforge.dev/gpgjson` (repository `brain-fuel/gpgjson`, released through
  v1.0.3), not `goforge.dev/gjson` as this entry previously implied.
- **Progress, and the one thing still blocking `(complete)`:** two real
  compatibility bugs are CLOSED, both found by the re-scoped gate.
  1. A hard-coded recovery collapsed `*.*.#.[".#|#.""""0"]` to `[]` where
     upstream projects to `[[],[]]`. It was written for a sibling where
     upstream does collapse; the discriminator is order — the empty-quote
     run must precede the projection marker.
  2. The BOOLEAN QUERY TABLE. `#[active<=0]` matched nothing where upstream
     selects every false element, because a guard refused any ordering
     comparison between a boolean field and a string operand, assuming the
     two would order once coerced. They never order: GJSON switches on the
     field's type and compares the operand TEXT, and the result is
     asymmetric — `true >= x` holds for every operand including `"x"` and
     `null`, while `true <= x` holds for none, not even `x == "true"`. The
     table was read off upstream by enumeration, because reasoning yields
     the plausible symmetric version and that version is wrong. 336
     combinations now agree exactly, where 40 did not.
  The gate is also split, which is what made those findable. Recorded seeds
  are replayed by `TestDynamicPathCorpus` under FULL parity — all 2412, no
  grammar bound — so nothing already achieved is at risk; the fuzzer bounds
  only NEW exploration, to eight rules read off the GJSON path reference
  (balanced quotes/brackets/parens; parens only as `#(` query heads;
  escapes only of `.`/`*`/`?`/`\`; no empty query or multipath; no path
  metacharacter inside a quoted key within a container or query; no quoted
  segment abutting another quoted segment or a bare token; literals only in
  multipath element position, which excludes `|!literal`).
  Two designs were tried and DISPROVED; neither should be retried. An
  instance-specific filter excluded 1 of 2412 entries and the fuzzer
  produced another divergence immediately. A valid-JSON invariant over the
  malformed remainder looked strong — upstream returns Raw `[0"]` for
  `[").[0A).0|!0"]` — until measurement showed gpgjson reproduces that same
  `[0"]` byte-for-byte elsewhere. The engine is deliberately BUG-FOR-BUG
  compatible, so that invariant would break the compatibility the package
  exists to provide.
  **What remains:** the fuzzer still reaches the grammar boundary
  (`*.*.#(*)#["."]`, `#(*)` as a condition, a multipath abutting `#(...)#`
  with no separator). Each is a question about what GJSON's path grammar
  actually admits, and answering them belongs in a systematic pass over the
  reference and upstream's parser — not in another round of tightening
  against the last counterexample, which is how the dozens of hand-written
  recovery branches in `compat_path.gp` accumulated. That pass is the
  completion criterion: a stated grammar, with parity asserted within it.

### `/goals/08-cron` - `robfig/cron` -> `std/schedule` (complete)

- **Why eighth:** Roughly 14k stars and a stable, compact scheduling domain whose
  parser ambiguity and runtime lifecycle are well suited to explicit types.
- **Destination:** Start in GoForge; promote the schedule value/parser to
  `std/schedule`, leaving goroutine ownership and logging adapters outside.
- **Semantic rewrite:** Separate standard and seconds-enabled grammars, validated
  immutable schedules, exhaustive next-run outcomes, explicit overlap policy,
  and lifecycle typestate.
- **Dependent pressure:** grammar-indexed `Schedule[F]`, nonempty field sets,
  parser-produced existential schedules, and `Cron[Stopped|Running]` transitions.
- **Gate:** Robfig parser/next-time corpus including DST, fake-clock concurrency
  tests, overlap laws, race tests, and compile-time lifecycle misuse rejection.
- **Status:** implemented as the Go+-authored `std/schedule` core and sibling
  module `goforge.dev/cron`, pinned to v3.0.1
  (`ccba498c397bb90a9c84945bbb0f7af2d72b6309`). The core separates
  `Schedule[5]` and `Schedule[6]`, seals nonempty `FieldSet[d]`, returns finite
  existential parses and exhaustive next outcomes, and matches Robfig across
  standard/seconds syntax, descriptors, named fields, leap years, locations,
  and DST gaps/repeats. The runner requires explicit parallel/skip/delay policy
  and enforces `Cron[0|1]` lifecycle transitions across package boundaries;
  generated Go retains runtime guards. `goforge.dev/cron` and `std/workflow`
  are independent production consumers. Fake-clock, overlap-law, fuzz, race,
  generation, vet, std-wide, and positive/negative compile gates pass. In the
  recorded five-run parser gate the rewrite is at least 2.26x faster and uses
  7 rather than 27 allocations (74.1% fewer).

### `/goals/09-resty` - `go-resty/resty` -> typed GoForge HTTP client (complete)

- **Why ninth:** Roughly 11.7k stars and more than 20k reported dependents. It can
  consolidate `std/http`, `std/retry`, streaming, and effect-boundary design.
- **Destination:** GoForge compatibility/client package over existing Go+ std
  primitives; promote only protocol-neutral request/response state machinery.
- **Semantic rewrite:** Immutable client policy, request builder typestate,
  exhaustive transport/status/decode outcomes, replayability-aware retries,
  and owned streaming bodies.
- **Dependent pressure:** method/body/replayability indices, response-code sums,
  and transitions that prevent retrying non-replayable bodies or decoding twice.
- **Gate:** Pin Resty v2/v3 target explicitly, publish compatibility matrix,
  HTTP conformance and cancellation tests, leak/race tests, retry safety tests,
  and compile-time illegal-state rejection.
- **Status:** implemented as `goforge.dev/resty`, pinned to Resty v2.17.2 at
  `b1b3aaa32811319f8180a4b211995a2edf21e2ea`. The Go+-authored core carries
  method, replayability, and decode-phase indices; quantity-1 request/response
  transitions; exhaustive outcomes; replay-safe 429/5xx/transport retries; and
  explicit body transfer. Go+ v0.26.0 adds cross-package inference for omitted
  natural indices on linear calls. `std/retry` gained shared normalized-attempt,
  cancellable-wait, and overflow-safe delay primitives. Anvil consumes the
  released module for ICS URL feeds. Generation, compile rejection,
  cancellation, body cleanup, allocation, race, vet, and compatibility gates
  pass. The conservative five-run end-to-end gate is at least 3.45x faster with
  13 versus 29 allocations, a 55.2% reduction.

### `/goals/10-participle` - `alecthomas/participle` -> typed parser construction (complete)

- **Why tenth:** A smaller audience (roughly 3.9k stars), but an excellent final
  forcing function: grammars, token streams, captures, and AST construction tie
  together nearly every staged dependent feature.
- **Destination:** GoForge package interoperating with `std/parsec`; promote only
  shared lexer/span/error primitives demonstrated by both consumers.
- **Semantic rewrite:** Grammar AST as a GADT, exhaustive lexer/parser failures,
  immutable source spans, explicit lookahead/commit semantics, and generated
  parsers whose output type is tied to the grammar.
- **Dependent pressure:** `Parser[G, T]`, grammar composition with FIRST-set
  evidence, token-count/indexed spans, and existential packaging of dynamically
  loaded grammars.
- **Gate:** Participle grammar compatibility corpus, ambiguity diagnostics,
  parser laws, differential/fuzz tests, generated-Go inspection, and realistic
  language benchmarks.
- **Status:** released as `goforge.dev/participle` v0.1.2, pinned to Participle
  v2.1.4 at `bcbb39153e17f8018257f17aba8eac628d396b64`. The Go+-authored core has a
  minimum-token-indexed grammar GADT, grammar/FIRST/parser identity indices,
  three-token immutable assignment spans, explicit lookahead and commit,
  exhaustive parse/build failures, and checked dynamic grammar binding. Go+
  v0.27.0 closes the cross-package consistency hole discovered by this forcing
  case: omitted natural witnesses are now validated against every indexed
  runtime argument. Positive and negative cross-module fixtures, differential
  fuzzing, race, vet, generated-source, ambiguity, erased-boundary, and
  allocation gates pass. Agile Frontier v0.2.1 is an independent Go+/WASM
  consumer. No `std/parsec` API was promoted because its rune positions and
  consumption model do not yet demonstrate the same token-count-indexed span
  abstraction with a second consumer. Across the recorded five-run semantic
  benchmark, the conservative gate is at least 38.2x faster with 263 versus
  8,237 allocations, a 96.8% reduction.

## Package workflow

Each goal follows the same sequence:

1. Pin an upstream version; record MIT provenance, API inventory, behavioral
   corpus, adoption evidence, and known incompatibilities.
2. Choose the semantic core before compatibility work. Illegal states become
   enums, refinements, indices, or explicit effects rather than comments.
3. Implement in `.gp`; generated Go is a checked distribution artifact. Keep a
   plain-Go boundary with runtime validation corresponding to erased proofs.
4. Add upstream differential tests, algebraic/property tests, fuzzing, race
   tests, serialization/interoperability checks, and comparative benchmarks.
5. Exercise the dependent surface across package boundaries with positive and
   negative compile fixtures. No feature counts as shipped if it only works in
   the declaring file.
6. Integrate at least one real GoForge consumer. Promotion to `std` additionally
   requires a second independent consumer sharing the same API and laws.
7. Audit licenses, generated-source reproducibility, module tidiness, vet, full
   root/std tests, coverage, performance, and migration documentation.

## Dependent-typing roadmap

Go+ remains a strict source superset of Go by making every feature opt-in in
`.gp` files (or explicit declarations) and by erasing proofs/indices to ordinary
Go plus boundary checks. Ordinary `.go` behavior and Go's type checker remain
unchanged.

**The shape.** A and B are shipped. C (matching), D (totality), and E
(automation) make the surface language good; F (kernel) is the threshold
for calling it dependently typed at all; G is interop and hardening; H is
metaprogramming, which is what Lean4 parity needs beyond F. Running
alongside them, and not a dependent-typing feature, is the effect-ordering
workstream that adds `do_dag`.

The dependencies that actually constrain the order: **D unblocks both F
and `do_dag`** — the kernel needs a totality predicate it can trust, and
`do_dag` needs one to know a binding is reorderable. Everything else is
sequenced by value rather than necessity, and C carries the most, because
its consumers (`std/vec`, `std/smt`) are already spelling around the gap.

### Stage A - decidable indexed programming (shipped foundation)

Retain refinements, GADTs, natural-number/value indices, cross-package markers,
index arithmetic normalization, structural matches, and runtime witnesses at
Go boundaries. Finish robustness for aliases, reassignment, generic wrappers,
methods, interfaces, separate compilation, and diagnostics. Goals 01-03 are the
forcing consumers.

### Stage B - propositions and validated witnesses

Add named predicate parameters, conjunction, and proof-preserving
functions. Equality witnesses shipped in v0.7.0 and ordering propositions
(`Le`, `Lt`) in v0.151.0, with `decide` as the general discharge witness —
the decider already settled inequalities, so this made them statable
rather than adding power. A proposition in scope became a HYPOTHESIS in
v0.152.0, which also allows an erased index to be forwarded to a call, so
bounds compose. As of v0.154.0 it is a hypothesis for MATCH
refinement too: a bound prunes a variant it excludes, and the generated
boundary guard agrees. Conjunction shipped in v0.155.0 as `And[P, Q]`,
whose parts are propositions: as a goal each is proved in turn, in scope
each becomes a hypothesis. Named propositions shipped in v0.156.0 as
`type InRange[i nat, n nat] prop { … }`: a name is an abbreviation, so a
use unfolds into the relations everything downstream already handled, and
the declaration erases to a marker so consumers can unfold it too.

**Stage B is complete, and consumed**: `std/vec`'s `AtIndex` and `Set`
carry their bound as `Lt[i, n]` rather than `Fin[n]` evidence, so the
proposition is a precondition at the call, a hypothesis that prunes the
impossible arm, and the fact that discharges the recursive step. What it deliberately does not include: an opt-in
Prop SORT distinct from Type, disjunction and negation, and quantifiers.
Those belong with Stage F's kernel, not with an SMT-free arithmetic
fragment.

The omitted-proof unsoundness found while building v0.152.0 is FIXED in
v0.153.0: obligations are settled on the first resolve iteration, before
any erasure exists, so a call that never carried a proof is
distinguishable from one whose proof was erased. A proof-carrying
function may also only be used in a direct call, since composition,
piping, partial application, and plain assignment all reached generated
Go without a proof. Ordinary erased indices keep their inference.

The ergonomic cost that rule imposed is repaid in v0.157.0. Erased
arguments were omitted as a GROUP, so naming the mandatory proof forced
spelling every index beside it — `AtIndex(i-1, n-1, decide, t)`. A call
may now omit exactly the INFERABLE erased arguments and still name its
proofs, and nothing is waved through: the index is inferred, then the
proposition is checked against it. The fix was in inference rather than
in the rule — unifying a parameter's `Vec[T, n]` against a caller's
`Vec[int, 3]` clashed at `T`, a type parameter rather than a dependent
variable, and abandoned the instantiation there, losing the one position
it wanted. `std/vec` is written in the shorter form, and the generated Go
is byte-identical.

Also in v0.157.0, and not a dependent-typing matter at all: generation
refuses to write an artifact that still contains a lowering carrier.
Pass 1 emits a SKELETON that resolution completes, resolution needs a
module, and without one the skeleton was written out as the finished
artifact — invalid Go from a command that exited 0. It parses, which is
why three goml scenarios and four goml tests had been asserting against
it. This is the rule typed holes already followed, generalized.

### Stage C - dependent matching that keeps its promises

Stage B made preconditions statable. Matching is where they are CONSUMED,
and it is the daily ergonomics gap against Idris2. Go+ already has GADT
matching, index refinement, exhaustiveness, and (v0.154.0) pruning driven
by a proposition in scope. What it does not have:

- **Guards and literal patterns** (`case P if cond:` / `| P if cond =>`).
  Reserved productions in both surfaces since the goml design, deferred
  because exhaustiveness over literals needs decider work. They land in
  `.gp` and goml simultaneously, per that decision.
- **Dot patterns** — a pattern position forced by unification, written as
  such rather than rebound. Today the forced value is either spelled again
  (and must be proved equal) or the arm cannot be written.
- **`with` abstraction / views** — matching on an intermediate whose type
  refines the scrutinee's indices. This is Idris2's central matching tool
  and has no Go+ spelling.
- **Explicit impossible arms.** Pruning is inferred from index clash; an
  author cannot say "this arm cannot occur" and have it checked.
- **Dependent motives** for a match in expression position.

Forcing consumers: `std/vec` and `std/smt`, both of which currently spell
around the gap.

### Stage D - totality, completed

`total func` already checks termination (`internal/core/total.go`,
v0.7.0): every self-recursive call must shrink an argument, structurally
or arithmetically. The gap is scope, not principle. The v1 surface admits
only `nat` parameters and a single `nat` result with no receiver, and only
SELF-recursion is inspected — mutual recursion is unsupported, and a
well-founded recursion with an author-supplied measure cannot be written
at all.

Stage D widens `total` to general types, adds mutual recursion and
measure-based descent, and adds productivity for codata if codata lands.
This is not a nicety: Stage F's kernel needs terminating reduction, so a
checked totality predicate is F's **precondition**. It is also what a
reorderable `do` block needs (see below), because `total` is pure today
only by accident of being restricted to nat arithmetic.

### Stage E - automation beyond the linear fragment

The arithmetic decider settles a linear fragment over naturals, and does
it well — it is `omega`-shaped, and four consecutive Stage B features
needed no new power from it. Outside that fragment there is nothing.
Lean4 has `simp`, `ring`, `decide`, and a tactic language; Idris2 has
proof search and `auto` implicits.

Go+ has a hook neither of them has in this form: **`class` laws are
already declared, and already property-tested**. A law is a rewrite rule
that the repo has independently gained evidence for. Stage E is:

1. **Law-driven rewriting** — a `simp`-shaped normalizer whose rule set is
   the laws in scope, so an equality the decider cannot see is discharged
   by rewriting rather than by arithmetic.
2. **Witness search** — instance resolution generalized to `auto`-style
   implicit proof arguments, so a proposition provable by composing
   in-scope facts need not be spelled.

Explicitly NOT a tactic language. Elaboration and automation stay outside
the trusted core, per Stage F's constraint.

### Where the Idris2 / Lean4 line actually is

Stages C through E make Go+ a strong refinement/indexed language with
matching, totality, and automation comparable in daily use. They do not
make it Idris2 or Lean4.

- **Idris2 parity** additionally needs Stage F's kernel (universes,
  Pi/Sigma, inductive families, intensional equality), plus codata and
  productivity.
- **Lean4 parity** needs all of that AND a metaprogramming layer —
  syntax quotation, macro expansion, elaborator reflection, and an
  interactive tactic surface. That is **Stage H**, listed separately
  rather than smuggled into F, because folding it into the kernel stage
  is how the kernel stage never ships.

Typed holes (v0.147.0) already supply the goal display both languages
have; the gap is what you can DO at a goal, not seeing it.

### Effect ordering - `do_dag`

A separate workstream from the dependent stages, and the one place the
roadmap adds a distinction neither Idris2 nor Lean4 draws.

**`do` is unchanged.** It stays the ordered block — the honest embedding
of Go statements (`let mut`, assignment, `defer`, `go`, loops, `?`),
executing in written order because that is what Go means and what every
ML-family and Lean-family reader expects of the keyword. There is no
rename and no `doseq`.

**`do_dag ... end` is the addition.** Its contents are ordered ONLY by
data dependency: the block is a DAG over its bindings, the compiler may
emit any topological order, and independent subtrees are candidates for
concurrent evaluation.

```
let Report (a Src, b Src) : Summary :=
  do_dag
    xs := Fetch a
    ys := Fetch b
    n  := Count xs        -- depends on xs, so it is scheduled after it
  end
  Merge n ys
```

The Haskell analogue is exact — `do_dag` is `ApplicativeDo` and `do` is
monadic `do`.

**The two blocks are not siblings, and that shapes the design.** An
ordered block holds STATEMENTS: effects, mutation, `defer`, `go`. A
dependency-ordered block cannot, because effects are precisely what
ordering exists to pin down. What remains is BINDINGS. A bare non-binding
statement inside `do_dag` is either an effect (rejected) or dead code
(rejected), so `do_dag` admits bindings and a result expression, not the
statement grammar `do` accepts.

**What the keyword earns.** Pure bindings are already reorderable in every
language and nobody marks them, so `do_dag` has to pay for itself twice
over:

1. a CHECKED assertion — everything in the block is independent, and being
   wrong is a diagnostic rather than a silent misordering;
2. a SCHEDULING permission — independent branches may be evaluated
   concurrently, which is the wall-clock payoff and the reason the
   construct is worth a keyword at all.

(2) implies (1): nothing can be scheduled concurrently without first
proving independence. It also sits cleanly beside what Go+ already has —
`go` is explicit unstructured concurrency the author manages, `do_dag` is
structured concurrency whose schedule is derived from the graph.

**The soundness precondition, and the sequencing it forces.** Reordering
is sound only if a binding's observable behaviour is exhausted by the
value it binds, so `do_dag` needs a purity predicate. Go+ has exactly one
candidate — `total func` — and it is pure today only by accident: the v1
surface restricts it to `nat` parameters and a single `nat` result, so
there is nothing it could do besides arithmetic. Widening `total` to
general types is Stage D. **`do_dag` therefore depends on Stage D**;
building it first means inventing a second effect discipline that Stage D
would then have to reconcile.

**Out of scope, and named so it is not assumed.** Effects that COMMUTE are
not admitted. Reordering commuting effects needs an effect algebra that
can prove two effects independent, which is a far larger commitment than a
purity predicate and is not part of this workstream. `do_dag` admits pure
bindings; anything effectful belongs in `do`.

### Stage F - total dependent core

To accurately call Go+ dependently typed, add an opt-in kernel with universes,
Pi and Sigma types, inductive families, intensional equality/J, normalization,
positivity checking, and termination/productivity checking. Kernel checking
must be small and deterministic; elaboration and automation remain outside the
trusted core. `total func` supplies executable proofs, while ordinary Go
functions remain partial and cannot reduce during type checking.

Stage D is F's precondition, not a duplicate of it: D makes the SURFACE
language's totality predicate real and general, which is what lets the
kernel trust that a `total func` reduces. The kernel still owns its own
positivity and productivity checks, because it must be checkable without
trusting the elaborator that produced the term.

### Stage G - Go interoperability and production hardening

Specify representation/erasure, dictionary and witness ABI stability, reflection
behavior, interface satisfaction, versioned package markers, proof-irrelevant
hashing, LSP goals/hover/completion, incremental checking, and resource limits.
Every exported dependent API must define what plain Go sees and where runtime
checks occur. Unsafe/foreign facts stay visibly labelled and auditable.

### Stage H - metaprogramming

Named as its own stage because Lean4 parity needs it and nothing in A-G
covers it: syntax quotation and antiquotation, macro expansion, elaborator
reflection, and an interactive surface for acting on a goal. Typed holes
(v0.147.0) already supply the goal DISPLAY both comparators have; the gap
is what an author can do once standing at one.

It is listed last deliberately. Folding metaprogramming into Stage F is
how Stage F never ships, and a macro layer over an unfinished kernel would
encode the kernel's provisional shape into every macro written against it.

The stopping point after Stage E is a useful refinement/indexed language. Stage
F is the threshold for a genuine dependently typed language; claiming that term
before a checked Pi/Sigma/equality/inductive/normalizing core would overstate the
implementation. Idris2 parity is F plus codata and productivity; Lean4 parity is
that plus Stage H. Stages C-E do not reach either comparator, and saying so is
the point of listing them separately.
