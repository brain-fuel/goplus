# GoForge candidates — Wave 2

_Drafted 2026-07-24. Membership/sizing/ordering below remain proposals except
where a ledger says otherwise. **Started 2026-07-24: direnv (candidate 1) —
differential-first foundation landed.**_

## Progress

| # | Project | Module | Status |
|---:|---|---|---|
| 1 | [`direnv/direnv`](https://github.com/direnv/direnv) v2.37.1 | `goforge.dev/gpdirenv` | **In progress** — **6 packages at byte-exact parity**, differential + property tested, race/vet/determinism clean: 3 library (`dotenv`, `sri`, `gzenv`) + 3 pure engine cores in Go+ (`env` diff/patch/reverse, `shell` w/ byte-exact `BashEscape` + json/gzenv, `xdg`). Pending: 9 more shells, effectful engine (`rc`/allow-deny, `watches`, `config`), CLI (0/23), stdlib (0/59), `std/config` extension. See [`../direnv/PARITY.md`](../direnv/PARITY.md). |
| 2–5 | sqlc / go-task / yq / goreleaser | — | Not started (proposals below) |

**Meta-tools leveled up while building direnv (2026-07-25):**
- **goforge** gained (a) **Go+/`.gp` awareness** — `deps`/boundary/worker-type analysis now scans `.gp` files (tolerant import scanner; prefers generated `_gp.go` when present), so a Go+ brick is no longer invisible; (b) **`goforge check format:json`** — machine-readable diagnostics for CI/agents (the explicit Wave 2 item); (c) a **`goforge gen`** command wrapping `go tool goplus gen ./...`. Verified end-to-end on a scaffolded `.gp` workspace. (Warrants a goforge minor bump at release.)
- **cupel** became **model-agnostic** — a new `openaichat` component (OpenAI Chat Completions wire format, configurable base URL + key) behind the existing `completer` seam covers **Claude, Codex, ollama, LM Studio, vLLM, and any local or remote OpenAI-compatible endpoint**; provider selection via `provider:`/`CUPEL_PROVIDER`, `base-url:`/`CUPEL_BASE_URL`. Claude path unchanged; all offline-tested via `httptest`. (Warrants a cupel minor bump at release.)

direnv was picked first per the selection thesis: smallest, differential-
friendly, and the second `std/config` consumer that retires Viper's open Wave 1
stdlib item. Building the pure cores first establishes the differential harness
before the CLI and engine land on top.

Wave 2 is drawn from the same ranked audit as Wave 1
([`GOFORGE_CANDIDATES.md`](GOFORGE_CANDIDATES.md)). Wave 1 members were ranks
1 (viper), 3 (validator), 6 (gjson), 10 (lo), and 26 (chi). Below-40 projects
already in the workspace — `expr`, `cron`, `resty`, `participle`, and
`std/decimal` — are complete and out of scope here.

## Selection thesis

Wave 1 proved five *library-sized* rewrites and forced five `std` cores
(`std/config`, `std/validate`, `std/pathquery`, `std/http/route`, and a partial
`std/iter`/`std/nonempty`/`std/vec`). Wave 2 should:

1. **Prefer native-Go upstreams.** A same-language rewrite is a semantic
   migration; a Python/Java port is a reimplementation with its own correctness
   burden. Ports are deferred to a later wave (list below).
2. **Harden the existing `std` cores with real second consumers, and close the
   one gap.** The audit's `std/` pressure across ranks 1–40 collapses into eight
   proposals. Most already exist in `goplus/std`: `parsec`, `workflow`, `cas`,
   `process`, `fsatomic`, `schedule`, `pathquery`, `config`, and `validate` are
   all present. Several have exactly one consumer today (e.g. `tbd` consumes
   `workflow`/`cas`/`process`; `viper` and `tbd` consume `config`; `gjson`
   consumes `pathquery`). **Promotion into `std` requires a second independent
   consumer sharing the same laws**, so Wave 2's highest-leverage move is to
   supply those second consumers — which simultaneously retires open Wave 1
   stdlib obligations. The one genuinely-missing core is **`std/iter`** (Lo's
   open obligation: fallible folds, stable grouping, ordered/unordered
   parallelism, opt-in fusion) — a Wave 2 collection/pipeline consumer would
   force it into existence.
3. **Favor tool/library sizing over application scale.** Application-scale
   systems (temporal, traefik, gin-the-framework, pocketbase, esbuild) force
   great `std` pressure but are multi-quarter reimplementations and poor first
   proofs of a new `std` core. Prefer self-contained CLIs/libraries whose *core*
   is the forcing case.
4. **Be built with the meta-tooling from day one.** Unlike Wave 1's flat
   modules, each Wave 2 project should be a **`goforge` workspace** (worker-type
   discipline enforced by `goforge check`) whose test suites are seeded by
   **`cupel`** from English behavior specs. This is the first real dogfood of
   GoForge + Cupel on greenfield rewrites and will surface what those tools
   still need (see "Tooling obligations").

## Proposed Wave 2 (native-Go, five slots)

| Rank | Upstream | Proposed module | ~Size | `std` core (all exist unless noted) | Why this one |
|---:|---|---|---|---|---|
| 8 | [`sqlc-dev/sqlc`](https://github.com/sqlc-dev/sqlc) | `goforge.dev/sqlc` | Large | **`std/parsec`** (2nd real consumer) | Highest-value parser forcing case short of esbuild; SQL→type-safe Go is a GADT-AST + typed-codegen showcase that stresses `std/parsec` beyond its current surface |
| 38 | [`go-task/task`](https://github.com/go-task/task) | `goforge.dev/task` | Medium | **`std/workflow`** + `std/schedule` (2nd consumer beyond `tbd`) | Self-contained DAG task runner: lifecycle typestates, dependency ordering, journaled runs, overlap policy (reuses `std/schedule` lessons from cron) |
| 30 | [`direnv/direnv`](https://github.com/direnv/direnv) | `goforge.dev/direnv` | Small | **`std/config`** (independent 2nd consumer) | Small enough to finish; the *second `std/config` consumer* that retires Viper's open stdlib item (capability-scoped source loading + watch/reload) |
| 28 | [`mikefarah/yq`](https://github.com/mikefarah/yq) | `goforge.dev/yq` | Medium | `std/config` + `std/pathquery` (extend) | YAML/JSON/XML/TOML processor; extends the `std/pathquery` + document-model line from gjson into multi-format, and stresses `std/config` writers |
| 37 | [`goreleaser/goreleaser`](https://github.com/goreleaser/goreleaser) | `goforge.dev/goreleaser` | Large | `std/process` + `std/fsatomic` (extend) | Release-pipeline effects: ordered stages, process capabilities, atomic artifact writes, redaction — the effect side of `std/workflow` |

> The one genuinely-missing core, **`std/iter`**, is forced by finishing Lo
> (a Wave 1 project, 47/651 declarations done) rather than by a new Wave 2
> project; a collection-heavy Wave 2 pick could become its second consumer.

Alternates, if any of the above are judged too large or off-theme:

- **`pion/webrtc` (27)** or **`schollz/croc` (39)** — the protocol/streams `std`
  line (owned streams, replayability, framed codecs, connection typestates).
  `croc` is the tractable one; `pion/webrtc` is a deep protocol library.
- **`twpayne/chezmoi` (22)** — `std/config` + `std/fsatomic`; a strong
  `std/fsatomic` forcing case if `direnv` is deemed too small.
- **`rqlite` (35)** — the storage/transaction-lease `std` line, but distributed
  consensus makes it application-scale.

## Deferred to a later wave

**Cross-language ports (higher risk — reimplementation, not migration):**
Python — pydantic(2), markitdown(7), langflow(9), spec-kit(11), deer-flow(13),
docling(15), crewAI(18), nanobot(19), black(20), langgraph(21), jieba(23),
poetry(24), magic-wormhole(25), openai-agents(29), pipenv(32), fastapi(36),
spaCy(40). Java — jsoup(12), NullAway(16), brigadier(33). Of these, **black(20)**
and **brigadier(33)** are the cleanest `std/parsec` follow-ons once that core
exists, and **pydantic(2)** is the natural `std/validate` second-language proof.

**Application-scale native-Go (great `std` pressure, multi-quarter):**
esbuild(4), temporal(5), act(14), pocketbase(17), traefik(31), gin(34),
rqlite(35).

## Per-candidate ledgers (proposed)

### sqlc (rank 8) → `std/parsec`

- **Forcing case:** a real SQL grammar with dialect variants, typed schema
  binding, and code generation. Illegal parser/analyzer phases should be
  unrepresentable; spans and tokens should be shared, not re-derived per pass.
- **`std/parsec` first cut:** source spans, lexer tokens, grammar-evidence
  combinators, GADT-indexed AST nodes, and deterministic diagnostic primitives —
  general enough for esbuild, black, and brigadier to reuse later (the second and
  third consumers that would justify promotion into `std`).
- **Go+ idioms:** GADT ASTs (a node's shape indexes its legal children),
  exhaustive analysis-outcome sums, refinement-typed identifiers, effect-free
  codegen with an explicit output boundary.
- **Sizing/risk:** large. Scope the first release to one dialect (Postgres) and
  the type-inference core; ledger the rest as follow-on parity.

### go-task (rank 38) → `std/workflow`

- **Forcing case:** a task DAG with dependencies, preconditions, status
  checks, and controlled re-execution. Reuses the overlap-policy and lifecycle
  lessons already proven in `std/schedule` + `goforge.dev/cron`.
- **`std/workflow` first cut:** transition sums, a journaled run log, lifecycle
  typestates (`defined → scheduled → running → done|failed|skipped`), and
  explicit ordered/parallel execution with cancellation.
- **Go+ idioms:** exhaustive run outcomes, capability-scoped shell/file effects,
  a topologically-indexed task set that makes cyclic graphs unrepresentable.
- **Sizing/risk:** medium; self-contained.

### direnv (rank 30) → `std/config` (second consumer)

- **Forcing case:** per-directory environment loading with precedence, allow
  lists, and reload-on-change — a *different* consumer of the same immutable-
  snapshot/provenance abstraction Viper needs, which is exactly the promotion
  bar (`std` requires a second independent consumer sharing the laws).
- **Retires:** the open Wave 1 Viper item "extend `std/config` with
  capability-scoped source loading and watch/reload events."
- **Go+ idioms:** capability-scoped source effects, immutable env snapshots with
  provenance, exhaustive allow/deny/stale states.
- **Sizing/risk:** small — the best first Wave 2 win and the fastest GoForge +
  Cupel dogfood.

### yq (rank 28) → `std/config` + document path

- **Forcing case:** structural edits across YAML/JSON/XML/TOML/CSV with a path
  expression language — a multi-format sibling to gjson's `std/pathquery`.
- **`std` pressure:** generalize `std/pathquery` toward a format-neutral
  document model + editing witnesses; stress `std/config` typed writers.
- **Go+ idioms:** format-indexed documents, exhaustive edit outcomes, owned vs
  borrowed node views.
- **Sizing/risk:** medium; the path-language surface is large.

### goreleaser (rank 37) → `std/process` + `std/fsatomic`

- **Forcing case:** a multi-stage release pipeline with shell/process effects,
  artifact writes, checksums, and secret redaction — the *effect* half of the
  workflow story that go-task's DAG deliberately leaves out.
- **`std` pressure:** `std/process` (process capabilities, deadlines, captured
  output) and `std/fsatomic` (atomic multi-file publish, provenance, redaction).
- **Go+ idioms:** capability-gated effects, exhaustive stage outcomes, owned
  artifact buffers, redaction as a type, not a convention.
- **Sizing/risk:** large; scope to the build/checksum/archive core first.

## Tooling obligations (GoForge + Cupel dogfood)

Building Wave 2 as `goforge` workspaces seeded by `cupel` is the point, and it
is a second goal in its own right: make these tools "what they need to be to
properly structure projects for humans or AI." Expected gaps to close *while*
using them:

- **`goforge`:** confirm worker-type propagation covers effect-heavy bricks
  (`std/process`, `std/fsatomic`); consider `goforge create` emitting an
  `AGENTS.md`/`CLAUDE.md` stub so every new brick is AI-legible from birth; make
  `goforge check` output machine-readable for CI/agents.
- **`cupel`:** the English-spec → three-test-styles flow (example, table-driven,
  and **property-based via `testing/quick`**) is the natural way to build each
  rewrite's differential/property corpus; verify it scales to a real parser/DAG
  spec, and that its `goforge check` gate composes with a project's generation
  (`goplus gen`) step.
- **Property-based testing is a first-class gate, not an afterthought.** Use the
  stdlib `testing/quick` for simple invariants and `pgregory.net/rapid` (the
  workspace's shrinking generator library, already used in `rune/harness`) for
  stateful/structured properties. Every semantic law a rewrite claims — parser
  round-trips (`parse ∘ pretty = id`), DAG topological soundness, config
  precedence associativity, iterator fusion equivalence — should be a checked
  property, ideally cupel-seeded from the English spec. Wave 1 already carries
  `laws_fuzz_test.go` and `*_gp_laws_test.go` files; Wave 2 makes the property
  suite part of the definition of done.
- **Structure Wave 2 rewrites as workspaces from day one**, not flat modules, so
  the semantic core (`std/*`) and the compatibility facade are separate bricks
  with declared worker types.

## Open decisions for the maintainer

1. **Five slots, or fewer/more?** Wave 1 was five. A tighter Wave 2 of three
   (sqlc, go-task, direnv) would prove `std/parsec` + `std/workflow` +
   retire `std/config`, with yq/goreleaser held for Wave 2.5.
2. **`std/parsec` scope.** Land it minimal with sqlc, or design the full
   combinator/GADT surface up front against esbuild/black/brigadier needs?
3. **Application-scale ambition.** Do we ever take temporal/traefik/esbuild, or
   keep GoForge a library/tool portfolio and let those force `std` pressure only
   as reference designs?
4. **Ports.** Is a Python/Java port (pydantic, jsoup) in charter at all, or is
   GoForge Go-upstreams-only?

## Definition of done

Wave 2 inherits the Wave 1 non-negotiable gates verbatim (see
[`GOFORGE_CANDIDATES_WAVE1.md`](GOFORGE_CANDIDATES_WAVE1.md) §"Non-negotiable
definition of done"), with two additions:

- Each project is a valid `goforge` workspace (`goforge check` clean; worker
  types declared and effect-propagation-correct).
- Each project's behavior corpus is generated from English specs via `cupel`
  and committed alongside the differential suite.
- Every claimed semantic law is a checked property test (`testing/quick` and/or
  `pgregory.net/rapid`), and shrinking is exercised on failure. A law without a
  property test is not "done."
