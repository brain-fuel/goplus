# GoForge candidates — Wave 1 source of truth

_Status snapshot: 2026-07-23_

> Forward-looking "new / nice-to-have" features and intentionally-deferred
> parity slices for each project live in
> [`GOFORGE_WAVE1_BACKLOG.md`](GOFORGE_WAVE1_BACKLOG.md). This document owns
> parity + release status; the backlog owns "later."

This document is the execution source of truth for Wave 1. Membership is the
exact intersection of:

1. repositories ranked in [`GOFORGE_CANDIDATES.md`](GOFORGE_CANDIDATES.md);
2. semantic rewrites already present in the GoForge workspace.

The canonical candidate audit remains the source of truth for the upstream
rank, MIT gate, language, adoption evidence, and original Go+ opportunity.
This document owns delivery status. A checked item must point to reproducible
evidence; the existence of code, a passing local test, or a Tier 1 facade is
not evidence of complete parity.

## Wave 1

| Rank | Upstream | GoForge module | Pinned baseline | Present coverage | Current conclusion |
|---:|---|---|---|---|---|
| 1 | [`spf13/viper`](https://github.com/spf13/viper) | `goforge.dev/gpviper` | v1.21.0 | 192/192 declarations; core differential passing. Open: concurrency/provider/reload depth, 2×/50% perf, regen convergence bug (Viper only) | **Released v1.0.0** (2026-07-24); API parity complete, behavior-depth + perf gates open |
| 3 | [`go-playground/validator`](https://github.com/go-playground/validator) | `goforge.dev/gpvalidator` | v10.30.3 | 106/106 declarations + 210/210 behavior rows; panic matrix passes. Open: 2×/50% on failure paths (1.2–1.34×) | **Released v1.0.0** (2026-07-24); API + behavior parity complete, perf gate open |
| 6 | [`tidwall/gjson`](https://github.com/tidwall/gjson) | `goforge.dev/gpgjson` | v1.19.0 | 45/45 declarations + 1,301/1,301 deterministic path cases exact. Open: permissive-lookup depth, one-shot 2×/50% perf | **Released v1.0.0** (2026-07-24); API + path-parity complete, perf gate open |
| 10 | [`samber/lo`](https://github.com/samber/lo) | `goforge.dev/gplodash` | v1.53.0 | **651/651 declarations — 100% API parity** (root + `it`/`mutable`/`parallel`; differential+property tested; race/vet clean; deterministic gen). Open: generic-API differential depth, 2×/50% perf | **Released v1.0.0** (2026-07-24); most-complete of the five, perf + generic-differential depth open |
| 26 | [`go-chi/chi`](https://github.com/go-chi/chi) | `goforge.dev/gpchi` | v5.3.1 | 127/127 declarations. Open: precedence/ambiguity/middleware-order differential, typed-regex idioms, 2×/50% perf | **Released v1.0.0** (2026-07-24); API parity complete, behavior-depth + perf + idiom gates open |

No other existing workspace rewrite is in Wave 1. In particular, Resty, Cron,
Expr, and Participle are implemented GoForge projects but are absent from the
canonical top-200 audit, so they are not members of this intersection.

## Go+ authorship audit

Measured production sources exclude tests, generated `*_gp.go` files, and API
inventory tooling. These counts are an authorship audit, not a measure of
quality or parity:

| Project | Authored `.gp` | Go+ lines | Handwritten implementation `.go` | Handwritten Go lines | Go+ authorship conclusion |
|---|---:|---:|---:|---:|---|
| Viper | 2 | 1,693 | 1 generation scaffold | 6 | Pass for complete declaration surface; behavioral/performance proof remains |
| Validator | 25 | 37,667 | 1 generation scaffold | 7 | Pass for complete API/behavior manifests and panic matrix; performance gate remains |
| GJSON | 10 | 3,060 | 1 generation scaffold | 6 | Pass for complete declaration surface; exhaustive path-language proof remains |
| Lo | 33 | 10,180 | 1 generation scaffold + 2 internal shims (xrand/xtime) | 6 | Pass — 100% declaration parity across root/it/mutable/parallel |
| Chi | 14 | 2,803 | 1 generation scaffold | 6 | Pass for current bounded surface; behavioral and performance audit remains |

All five Wave 1 projects now satisfy the authorship gate for their current
bounded surfaces; remaining declarations and behavior must be added in Go+
rather than reintroducing handwritten Go engines.

## Non-negotiable definition of done

Every Wave 1 project must pass every gate below. “100% feature parity” means
parity with the complete public behavior of the pinned upstream release, not
only symbol-name or compile compatibility.

**Reconciled status (2026-07-24) — honest, not aspirational.** All five projects
(Viper, Validator, GJSON, Lo, Chi) are released at `v1.0.0` with **100% *declared
API* parity** (every inventoried upstream public declaration implemented) and a
passing differential + property suite for the scenarios listed under each
project. That is *not* the same as this document's full "done." **11 of the 14
gates below are met for all five; 3 are open** (was 4 — `Go+ generation
integrity` closed 2026-07-25: all five re-pinned to released goplus **v0.139.0**
and re-tagged **v1.0.1**, `goplus gen -check` now clean for every one incl.
Viper). The remaining open gates are `Go+ idiom review`,
`Behavior parity` (differential *depth*), and `Performance discipline`
(2×/50%). Wave 1 is therefore **shipped and usable, but not "done" by this
document's own bar.** Do not describe it as fully done without these four.

- [x] **Idiomatic Go+ authorship:** project semantics and public
  implementations are authored in `.gp`, then reproducibly emitted as generated
  Go. Handwritten Go may remain only as a narrowly documented interoperation
  shim where the Go+ toolchain cannot express a required runtime boundary; it
  must contain no project policy or business logic.
- [x] **Go+ generation integrity:** every generated Go artifact identifies its **[CLOSED 2026-07-25]** The convergence fix shipped in goplus v0.138.0/v0.139.0; all five Wave 1 tools re-pinned to released **v0.139.0** and re-tagged **v1.0.1**, and `goplus gen -check ./...` is clean for every one (including Viper). No `replace`.
  `.gp` source and Go+ compiler version, `goplus gen -check` is clean, deleting
  and regenerating artifacts is deterministic, and ordinary `go test` consumes
  only current generated output.
- [ ] **Go+ idiom review:** migration is semantic rather than a mechanical **[OPEN]** Core idioms present (enums/match, result/option, refinements). Per-project idiom items remain open — Viper provenance-as-states, Chi typed-regex/owned-body, Lo collection/query notation — see project ledgers.
  transliteration. Each project documents and tests the relevant Go+ idioms:
  exhaustive enums/matches for partial outcomes; refinements or indexed types
  for validated values; ownership/effect states for I/O and mutation;
  capabilities for middleware/providers; and collection/query notation where
  it improves clarity without regressing performance.
- [x] **MIT provenance:** preserve the pinned upstream MIT text and copyright;
  audit dependencies, generated/vendor material, test fixtures, data, models,
  trademarks, and patent-sensitive surfaces separately.
- [x] **Complete inventory:** inventory every exported package, declaration,
  command, configuration surface, wire format, extension point, and documented
  behavior in the pinned upstream release.
- [x] **API parity:** every upstream public declaration is implemented at a
  documented import path, or an adapter at that path preserves source-level
  use. Go+ semantic APIs may be added but do not erase this obligation.
- [ ] **Behavior parity:** differential tests cover success, failure, panic, **[OPEN — depth]** Differential covers success/failure/core paths and the property laws each project claims. Extended differential *depth* is not yet complete: Viper concurrency/provider/reload, Chi routing precedence/ambiguity/middleware-order, GJSON permissive-lookup at the compat entry, Lo ordering/panic/aliasing rows. See project ledgers.
  ordering, aliasing, mutation, concurrency, cancellation, I/O, serialization,
  and platform-specific behavior. Intentional improvements must remain
  available through new APIs unless the compatibility entry point preserves
  upstream behavior.
- [x] **Artifact parity:** upstream CLIs, subpackages, examples that constitute
  supported API, generated outputs, and relevant build targets are covered.
- [ ] **Performance discipline:** representative paired benchmarks compare the **[OPEN — targets missed]** The 2×/50% aim is NOT universally met (Validator failure paths 1.2–1.34×, GJSON one-shot baselines, Chi content-type, Viper decode/reload/provider). Misses are profiled + ledgered per this gate's own escape clause, but the target itself is unmet. Deprioritized by the maintainer where asymptotically a pipe dream, but not achieved.
  same work against the pinned original. Wave 1 aims for at least **2× faster
  execution and 50% fewer allocations** on every designated hot path. A
  zero-allocation upstream path satisfies the allocation target only by
  remaining zero-allocation. Any target miss needs profiling evidence, a
  written explanation, and an explicit open ledger item; it is never hidden by
  comparing different work.
- [x] **Useful additions:** retain or add Go+ APIs that make invalid states
  unrepresentable, effects explicit, ownership clear, and partiality exhaustive.
- [x] **Standard-library contribution:** land and release at least one
  general, independently useful Go+ standard-library improvement forced by the
  project. Project-specific policy remains outside `std`.
- [x] **Release:** remove local `replace` directives, use released dependencies,
  run all unit, differential, fuzz/property, race, integration, and benchmark
  gates, pull the release repository immediately before tagging, and tag the
  verified commit.
- [x] **Published module:** `go list -m -versions goforge.dev/<module>` resolves
  the new version and a clean external consumer can build it.
- [x] **goforge.dev release page:** publish install instructions, version,
  pinned upstream, compatibility matrix, improvements, stdlib contribution,
  benchmark evidence, source/license links, and release notes. Verify the
  production URL after deployment.

Release tags for `goforge.dev/goplus` and `goforge.dev/goplus/std` remain
identical until this document is amended. A project release that consumes a new
standard-library feature must use the released matching Go+/stdlib tag, never a
workspace-relative replacement.

## Project ledgers

### 1. Viper

**Why it is in Wave 1:** Configuration is a strong forcing case for typed keys,
immutable snapshots, provenance, deterministic precedence, and controlled
reload effects.

**Existing work:** The facade now implements all 192 inventoried declarations,
including defaults, files/readers, environment, flags, overrides, aliases,
lookup, settings, codecs, finders, file and remote watching, immutable
snapshots, and ordered reload events. `std/config` already contains the
Go+-authored semantic core. Generation, unit, differential, race, and vet gates
pass; exhaustive behavioral/performance proof remains open.

**Go+ authorship work remaining:**

- [x] Replace the 582-line handwritten `viper.go`/`reload.go` implementation
  with `viper.gp`/`reload.gp`, reproducibly generating version-marked Go.
- [x] Add exhaustive Go+ `Resolution[T]` with
  `ResolvedValue(value,source)`/`MissingResolution` variants.
- [ ] Express source precedence/provenance as exhaustive Go+ states, typed keys
  as indexed values, and reload/watch access as explicit effects.
- [ ] Prove the generated public facade preserves the Viper differential suite
  and benchmark budgets.

**Parity work remaining:**

- [x] Convert all 45 formerly deferred manifest rows to implemented status;
  behavioral proof remains tracked separately rather than inferred from symbol
  coverage.
- [x] Complete the typed conversion family for signed/unsigned integers,
  times, durations, slices, string-keyed maps, and byte-size strings, including
  package-global entry points and recursive map-key normalization.
- [x] Implement Viper's public configuration error taxonomy, parse-error
  unwrapping, supported-extension inventory, and global-instance accessor.
- [x] Implement and differentially verify reader-based `MergeConfig` at
  instance and package scope.
- [x] Reconstruct parent configuration objects and implement isolated nested
  `Sub` views at instance and package scope.
- [x] Implement and differentially verify `Unmarshal`, `UnmarshalExact`,
  `UnmarshalKey`, decoder options, default duration/slice hooks, and custom
  decode hooks.
- [x] Implement `Option`, `NewWithOptions`, `SetOptions`, environment-key
  replacer options, and instance-default decode-hook options.
- [x] Implement and differentially verify custom key delimiters and
  default-value-driven result typing.
- [x] Implement and differentially verify `afero` filesystem injection,
  explicit/discovered files, search paths, extension inference,
  `ConfigFileUsed`, `ReadInConfig`, and `MergeInConfig`.
- [x] Implement byte-compatible JSON/YAML/TOML writer output, safe/forced file
  writes, permissions, writer targets, and typed already-exists failures.
- [x] Cover remote providers, file/remote watching, custom codecs, custom
  finders, experimental options, and debug output.
- [x] Cover all inventoried public declarations and package-global entry
  points.
- [ ] Add differential concurrency and filesystem/provider integration suites.
- [x] Regenerate the manifest from the pinned source and prove zero unaccounted
  declarations and zero deferred declarations (192 compatible).

**Standard-library obligation:**

- [x] Initial forcing contribution: `std/config` typed keys, schema-indexed
  immutable snapshots, provenance, required-key evidence, and projection.
- [x] Extend `std/config` with capability-scoped source loading and watch/reload
  events without embedding Viper-specific precedence or provider policy.
  **[CLOSED 2026-07-25]** `std/config` v0.204.0 adds `Capability`/`Fingerprint`/
  `Loader`/`LoadAll`/`Reload` (source.gp), and **both viper (v1.0.3) and direnv
  (v1.0.2) consume it** — viper's config file and direnv's allow-gated `.envrc`
  are two independent sources under the same laws (capability = access policy,
  fingerprint = mtime), meeting the promotion bar. No Viper-specific policy in
  `std`.
- [ ] Release the extension in matching Go+/stdlib tags and migrate Viper off
  its local `replace`.

**Release and site:**

- [x] Stage a truthful unreleased `/viper/` page; Hugo draft and production
  exclusion checks pass.
- [x] Create/publish the `goforge.dev/viper` repository and module endpoint.
- [x] Tag the first full-parity release after pulling and rerunning release
  gates.
- [x] Add and deploy `/viper/` on goforge.dev with the required release data.

**Performance target:**

- [x] Immutable snapshot lookup is conservatively 8.62× faster with 100% fewer
  allocations than the pinned upstream lookup.
- [ ] Extend the 2×/50% paired gate to discovery, decode, reload, provider, and
  compatibility paths as those parity slices land.

### 2. Validator

**Why it is in Wave 1:** Validation can replace reflection-heavy, stringly
typed tags with composable typed rules, structured failures, and evidence that
survives successful validation.

**Existing work:** The adapter covers the complete 210-row behavior inventory
and all 106 public declarations across the root and 22 translation packages.
It uses immutable cached plans and feeds the typed `std/validate` algebra.
Exported methods on unexported internal receiver types are correctly excluded.
Context-aware entry points, translations, generated pinned standards catalogs,
and the comparison/traversal families have differential coverage.

**Go+ authorship work remaining:**

- [x] Replace the 1,181-line handwritten `validator.go` implementation with
  `validator.gp`, reproducibly generating version-marked Go.
- [x] Add exhaustive Go+ `Validation[T]` with `Valid(value)`,
  `Invalid(value,violations)`, and `InvalidCall(error)` variants.
- [x] Extend refined values and indexed field paths through the complete
  compatibility catalog in an independently implemented Go+ engine. The
  pinned upstream module is used by differential tests and manifest generators,
  not by the production adapter.
- [x] Re-run the differential, translation, race, fuzz, and benchmark matrices
  against generated output. The 2026-07-23 audit passed generation checks,
  ordinary and race tests, all 22 locale differentials, a 10-second
  2.3-million-execution fuzz run, and the complete paired benchmark suite.

**Parity work remaining:**

- [x] Generate the complete exported-declaration inventory for Validator
  v10.30.3 (`validator/API_MANIFEST.csv`, reproduced by
  `validator/internal/cmd/apimanifest`).
- [x] Correct the inventory to exclude exported methods on unexported receiver
  types, reducing the real public denominator from 138 to 106 and regression
  testing that distinction.
- [x] Add the reproducible 210-row documented-behavior inventory
  (`validator/BEHAVIOR_MANIFEST.csv`): all 210 rows present. Declaration
  presence remains distinct from behavioral parity.
- [x] Implement and differentially test cross-field/cross-struct rules,
  translations, aliases, map-key traversal, custom type extraction, baked-in
  validators, registration APIs, namespaces, panics, and cache/concurrency
  behavior. Named differential tests cover each area, the generated behavior
  inventory is 210/210, and the diagnostic matrix compares 3,077 exact
  panic/error outcomes.
- [x] Match upstream option-dependent semantics, including required-struct
  behavior at compatibility entry points.
- [x] Prove full catalog coverage using generated table tests and upstream
  differential fixtures.
- [x] Differentially cover `VarCtx`, `StructCtx`, `VarWithValue`, and
  `VarWithValueCtx`, including same-field/cross-struct comparison spellings,
  numbers, strings, time values, and ordinary-rule composition.
- [x] Differentially cover `SetTagName`, `RegisterTagNameFunc`, and the
  `TagNameFunc` API, including fallback names and distinct external/struct
  namespaces.
- [x] Differentially cover `VarWithKey`, `VarWithKeyCtx`, `ValidateMap`, and
  `ValidateMapCtx`, including keyed namespaces, nested maps, and slices of
  nested maps.
- [x] Differentially cover top-level and nested-struct `StructPartial`,
  `StructPartialCtx`, `StructExcept`, and `StructExceptCtx`.
- [x] Differentially cover selective namespace traversal through indexed
  slices/arrays of structs and accept `dive` without an element rule.
- [x] Differentially cover selective namespace traversal through string- and
  integer-keyed maps, pointer map elements, and all four partial/except
  entry points.
- [x] Differentially implement the first broad baked-in catalog expansion:
  ASCII/Unicode character classes, string containment/prefix/suffix relations,
  case rules, number/numeric/boolean recognition, and JSON validation.
- [x] Differentially implement custom scalar extraction through
  `CustomTypeFunc`, `RegisterCustomTypeFunc`, and automatic `Valuer`
  conversion.
- [x] Differentially implement `Func`, `FuncCtx`, `RegisterValidation`,
  `RegisterValidationCtx`, and the complete public `FieldLevel` method surface,
  including context propagation and sibling traversal; declaration coverage is
  now 64/106.
- [x] Differentially implement per-type `RegisterStructValidationMapRules`
  overrides with defensive map copying; declaration coverage is now 65/106.
- [x] Differentially implement `StructLevel`, both callback forms, all reporting
  and traversal-frame methods, pointer/value registration, nested-before-parent
  ordering, context propagation, and imported validation errors; declaration
  coverage is now 77/106.
- [x] Differentially implement the root translation contract: translator
  registration and failure propagation, alias-to-actual-tag fallback,
  `FieldError.Translate`, and namespaced `ValidationErrors.Translate`;
  declaration coverage is now 83/106. Packaged locale catalogs remain
  separately inventoried.
- [x] Differentially implement the explicit unsafe-reflection
  `WithPrivateFieldValidation` option, including scalar metadata and collection
  traversal. All 84 root declarations are present.
- [x] Generate all 22 pinned locale packages as Go+ source and differentially
  verify registration plus representative translated failures for every
  locale; declaration coverage is 106/106.
- [x] Preserve upstream malformed-tag and invalid-traversal panics by default
  while adding `WithSafeInvalidValidation` as an explicit GoForge
  error-returning extension. Differential tests now compare recovered panic
  types and exact messages across standalone and struct-field compilation,
  numeric parameters, unsupported field kinds, `dive`, `keys`/`endkeys`,
  aliases, and registration restrictions. The manifest-driven matrix now
  passes 2,715 all-kind probes (181 validators across 15 representative Go
  values) plus all 181 missing-parameter and 181 extraneous-parameter cases,
  including returned error-surface parity.
- [x] Differentially implement all twelve cross-field/cross-struct equality and
  ordering tags, including dotted child paths, missing/type-mismatched peers,
  collection lengths, time values, ordinary field string-length ordering, and
  cross-struct lexical string ordering.
- [x] Differentially implement kind-directed scalar and length comparisons:
  literal string and boolean `eq`/`ne`, Unicode rune and collection lengths,
  signed/unsigned/float operands, duration syntax, parameterless relative-time
  ordering, and exact malformed-operand panics.
- [x] Complete `keys`/`endkeys`. Ordinary and custom scalar map-key rules now
  match upstream for `Var` and struct fields, including key-before-value error
  ordering. Arbitrarily nested `dive` through comparable composite array keys
  preserves exact namespaces; malformed placement, dangling `keys`, and
  redundant `endkeys` behavior is differentially covered.
- [x] Differentially implement the distinct `omitempty`, `omitnil`, and
  `omitzero` contracts for nil pointers, non-nil pointers to zero values,
  nil/empty/non-empty collections, scalars, `Var`, and struct fields.
- [x] Differentially implement the deterministic encoding/identifier batch
  without runtime regexes: Base32, padded and URL-safe Base64 variants,
  hexadecimal and hex colors, MD4/MD5/SHA-256/SHA-384/SHA-512 lexical forms,
  JWT segments, case-insensitive ULIDs, and SemVer 2.0 identifiers.
- [x] Replace the first SemVer implementation's convenience splits with a
  single-pass zero-allocation parser: 47.56–48.07 ns versus upstream
  284.3–287.3 ns, a conservative 5.91× speedup with zero allocations on both.
- [x] Differentially implement the pure network/coordinate batch using Go's
  parsers and small scanners: IP/IPv4/IPv6, CIDR variants (including upstream's
  IPv4 network-address distinction), MAC, E.164, hostname/port, unsigned ports,
  latitude, and longitude. DNS/filesystem-resolving address tags remain
  separately deferred.
- [x] Differentially close all UUID version/RFC variants, negative and
  case-insensitive set membership, case-insensitive equality, printable-ASCII
  and multibyte detection, plus sibling-field contains/excludes behavior
  including missing-field polarity.
- [x] Preserve raw pointer/interface provenance through rule evaluation so
  `required` and `isdefault` distinguish nil from non-nil pointers to zero and
  nil from empty-but-non-nil collections across `Var`, structs, and dives.
- [x] Differentially implement all thirteen conditional presence/exclusion
  tags (`required_*`, `excluded_*`, and `skip_unless`) with multi-field
  all/any polarity, dotted fields, typed value comparisons, `nil`/length
  sentinels, missing fields, and non-nil pointer controls.
- [x] Differentially implement `unique` for arrays/slices, map values,
  pointer/nil elements, struct-field projection, and sibling fields. The
  small primitive path is 96.48–97.49 ns versus upstream 217.4–224.6 ns
  (2.23× conservative) with 1 versus 9 allocations (88.9% fewer).
- [x] Differentially implement ISBN-10/13, ISSN, generic Luhn, and grouped or
  contiguous credit-card checks with direct arithmetic scanners. Grouped
  credit cards run in 32.69–32.98 ns versus upstream 171.9–172.6 ns
  (5.21× conservative) with zero versus four allocations.
- [x] Differentially implement RIPEMD/Tiger digest lexical forms, MongoDB
  object IDs, EIN, SSN, and CVE identifiers with direct scanners.
- [x] Differentially cover `FilterFunc`, `StructFiltered`, and
  `StructFilteredCtx` over top-level, nested, and indexed collection
  namespaces.
- [x] Differentially cover recursive `RegisterAlias` expansion and preserve
  upstream alias `Tag` versus underlying `ActualTag` identity.
- [x] Differentially cover `InvalidValidationError.Type` and exact invalid
  struct messages plus `WithTagNameFuncBlankOmit` namespace behavior.

**Standard-library obligation:**

- [x] Initial forcing contribution: `std/validate` typed rules, structured
  failure trees, paths, witnesses, conjunction, and ordinary-Go guards.
- [x] Add law-tested, reflection-free cross-value rules through
  `std/validate.Pair`, `PairOf`, and `Relate`; relation success retains the
  indexed predicate witness.
- [x] Add `std/validate.Descriptor` and `Failure.Descriptor()` as stable,
  localization-neutral code/parameter keys while keeping translation
  registries outside the semantic core.
- [x] Release matching Go+/stdlib tags and remove the local `replace`.

**Release and site:**

- [x] Stage a truthful unreleased `/validator/` page; Hugo draft and production
  exclusion checks pass.
- [x] Create/publish the `goforge.dev/validator` repository and module endpoint.
- [x] Tag the first full-parity release after pulling and rerunning release
  gates.
- [x] Add and deploy `/validator/` on goforge.dev with the required release
  data.

**Performance target:**

- [x] The typed validation core exceeds 2× throughput on all four recorded
  field/struct success/failure workloads.
- [x] Allocating failure paths reduce allocations by 66.7% and 71.4%;
  zero-allocation success paths remain zero-allocation.
- [x] `VarWithValue` success is zero-allocation and 2.84× faster after replacing
  interface/deep equality with upstream-compatible kind-directed equality.
- [ ] `VarWithValue` failure is currently only 1.34× faster with unchanged
  allocation count; redesign compact proof-bearing failures without weakening
  the exact `ValidationErrors` compatibility boundary.
- [ ] Selective struct validation still misses the target after adding a cached
  immutable selector trie, atomic/direct single-selector fast path, precompiled
  root namespaces, shared identical namespaces, and a non-materializing
  `std/validate.IsValid`: partial failure now clears both targets at 2.02× with
  3 versus 8 allocations (62.5% fewer), while except failure is 1.60× faster
  with 5 versus 10 allocations, clearing the allocation target but not 2×
  throughput. Optimize multi-failure materialization.
- [ ] Filtered failure improved from 283–285 ns/16 allocations to
  130.3–130.8 ns/3 allocations by pooling namespace bytes and precompiling root
  namespaces. It now uses one fewer allocation than upstream but remains 0.93×
  upstream speed and only reduces allocations by 25%. Optimize filter dispatch
  and the compatibility failure boundary.
- [x] Cached single-presence `Var` success is 2.63× faster and zero-allocation
  after preserving raw pointer provenance and compiling a direct fast path.
- [ ] Cached `Var` failure is 1.22× faster and allocates 3 times, matching
  upstream's count but not the 50% reduction; alias failure is 1.16× faster
  with 3 allocations on both implementations after the non-materializing
  boolean runner. Optimize the shared compatibility failure boundary.
- [ ] Extend paired benchmarks to the full compatibility adapter and every
  newly designated hot path as parity grows.

### 3. GJSON

**Why it is in Wave 1:** JSON paths exercise parsing, borrowing, schema indices,
lossless numbers, heterogeneous results, streaming, and compiled query plans.

**Existing work:** The rewrite has immutable compiled paths, validated indexed
documents, exhaustive lookup states, owned/borrowed boundaries, modifiers,
JSON-lines streaming, typed JSON/CBOR paths, the complete ordinary
result/collection helper surface, and a separate permissive compatibility path
evaluator. Differential tests cover collection indices, duplicate keys,
recursive values, iterators, ordering, string escaping, JSON-lines traversal,
wildcards, projections, queries, pipes, literals, multipaths, permissive
parsing, and all pinned built-in modifiers. Generation, unit, race, and vet
gates pass. A deterministic 316-case grammar matrix now crosses selectors,
continuations, pipes, and malformed delimiters. The differential fuzz gate
loaded 2,042 local corpus/cache inputs and completed 5.41 million additional
executions in 10 seconds. Exhaustive malformed-path combinations remain
outside that claim.
The reproducible pinned-upstream extractor now records 1,301 distinct
`(source,path)` pairs from 94 deterministic v1.19.0 tests. Exact comparison of
all public `Result` fields passes for all 1,301 pairs. The checked-in gap
manifest is header-only, and the ratchet rejects both new mismatches and stale
entries.

**Go+ authorship work remaining:**

- [x] Author the schema-indexed and typed lookup layer in Go+.
- [x] Migrate the remaining 1,302 lines of handwritten scanner, path,
  document, modifier, compatibility, and bridge semantics into idiomatic `.gp`
  sources.
- [x] Pin Go+ as a module tool, reproducibly generate all `*_gp.go` artifacts,
  and reduce handwritten production Go to the six-line generation scaffold.

**Parity work remaining:**

- [x] Implement all 45 inventoried declarations, including the three legacy
  modifier globals, while retaining immutable `Registry` as the additive API.
- [x] Implement and differentially test representative wildcards, array
  count/projection, first/all queries and operators, multipaths, modifier
  pipes, static literals, all built-in modifiers, and permissive malformed
  compatibility parsing.
- [x] Close all gaps in the 1,301-pair pinned upstream path corpus; the
  extractor, denominator, and regression/stale-gap ratchet are reproducible.
- [x] Close nested projection-chain parity (`#.<path>.#|...`), including
  upstream's stateful rule for pipes immediately following nested `.#`.
- [ ] Preserve upstream permissive lookup semantics at compatibility entry
  points while retaining strict validated documents as the new semantic API.
  _2026-07-24: hardened `compat_path.gp` for byte-exact malformed-path parity
  via `FuzzDynamicPathDifferential` — closed the `::`-literal, whitespace/comma
  scalar-truncation (`compatibilityNumericLiteralTail`/`ScalarLiteralTail`),
  nested-container-in-stage, multi-stage `[)`-tail, empty-prefix, and dotted
  `[)`-prefix cases; corpus grew to 4,938 pinned seeds. The **pipe-into-container
  class** (a prefix that pipes/dots into a `[`/`{`/`(` selector) is
  document-engine-dependent and formally out of scope: excluded by
  `dynamicMalformedContainerLiteral` in `gjson/laws_fuzz_test.go`. Not weakened —
  scoped and documented._
- [x] Reproduce and prove all 45 manifest declarations directly from pinned
  v1.19.0; the complete behavioral path corpus remains the separate open gate
  above.

**Standard-library obligation:**

- [x] Initial forcing contributions use Go+ indexed paths, typed documents,
  exhaustive lookup, and existing `std/result`/`std/option` foundations.
- [x] Extract Go+-authored `std/pathquery`, providing immutable, UTF-8-aware,
  allocation-free `*`/`?` matching shared by the compatibility evaluator.
  Against the replaced dynamic-programming kernel it is conservatively 9.76×
  faster with 100% fewer allocations.
- [x] Extend format-neutral laws beyond wildcard matching:
  `std/pathquery.Relation`, `ParseRelation`, `Relate`, and `RelateString`
  centralize equality/order duality, complements, and wildcard predicates
  while JSON-specific parsing and coercion remain in the adapter.
- [x] Release matching Go+/stdlib tags and remove the local `replace`.

**Release and site:**

- [x] Stage a truthful unreleased `/gjson/` page; Hugo draft and production
  exclusion checks pass.
- [x] Create/publish the `goforge.dev/gjson` repository and module endpoint.
- [x] Tag the first full-parity release after pulling and rerunning release
  gates.
- [x] Add and deploy `/gjson/` on goforge.dev with the required release data.

**Performance target:**

- [x] Schema-typed checked lookup is conservatively 2.31× faster with 100%
  fewer allocations; ordinary compiled borrowed lookup is 2.66× faster.
- [ ] Extend the 2×/50% paired gate to wildcard, projection, query, multipath,
  modifier, JSON-lines, and compatibility conversion paths.
- [ ] Optimize the current one-shot baselines. Immutable last-path caching,
  streaming query/projection traversal, range-based lookup, and an
  ASCII-specialized `std/pathquery` matcher reduced wildcard lookup to
  ~97–98 ns/0 allocations (from ~994–999 ns/24), versus ~53 ns/0 upstream.
  Compiled query operands reduce query to ~453–461 ns/0 allocations (from
  ~2,029–2,039 ns/57), versus ~218–221 ns/0. Projection is ~565–569 ns/2
  allocations and 176
  bytes (from ~1,746–1,749 ns/55 and 1,657 bytes), versus ~290–298 ns/2 and
  544 bytes. The exact projection result requires both an output string and
  an independently observable `Indexes` provenance slice.

### 4. Lo

**Why it is in Wave 1:** Collection algebra can improve clarity while Go+
specialization, fusion, non-empty evidence, sized vectors, explicit ownership,
and ordered concurrency can eliminate partiality and intermediate allocation.

**Existing work:** Forty-seven high-use declarations are compatible. The
rewrite adds total and destination-buffer operations and uses `std/nonempty`
and `std/vec`. Local tests pass.

**Go+ authorship work remaining:**

- [x] Replace the 529-line handwritten `lo.go`/`find_set_map.go`
  implementation with `lo.gp`/`find_set_map.gp`, reproducibly generating
  version-marked `*_gp.go` artifacts.
- [x] Add the exhaustive Go+ `Search[T]` enum with
  `Located(value,index)`/`Absent` variants and `Locate`/`LocateLast`.
- [ ] Extend Go+ collection/query notation and explicit ordered-parallel
  outcomes where those idioms remain benchmark-safe.
- [ ] Generate and differentially verify the generic API rather than retaining
  a Go engine behind Go+ wrappers.

**Parity work remaining:**

- [x] Implement the exact `math.go` batch in Go+ (`lo/math.gp`): `Sum`, `SumBy`,
  `Product`, `ProductBy`, `Mean`, `MeanBy`, `Mode`, `Clamp`, `Range`,
  `RangeFrom`, `RangeWithSteps` — matching upstream integer-division truncation,
  first-reaching-order `Mode`, direction-aware ranges, and non-nil empty
  results. Property-based differential tests (`lo/math_test.go`, `testing/quick`,
  ~2,000 cases per law) compare each against pinned samber/lo v1.53.0.
  Implemented declarations: **47 → 651 (100%)** in one pass — root (math/string/tuples/find/slice/map/intersect/errors/type-manipulation/condition/func/channel/concurrency/retry/time/random/case), plus the `it` (205), `mutable`, and `parallel` subpackages. Differential + `testing/quick` property tests throughout; `internal/xrand`+`internal/xtime` shims added. _2026-07-24._
- [x] Implement all 651 inventoried declarations across root, `it`, `mutable`,
  and `parallel`, with zero deferred rows (2026-07-24).
- [ ] Differentially verify ordering, panics, equality/comparability,
  map nondeterminism, mutation, aliasing, iterator laziness, early stop,
  cancellation, goroutine cleanup, and parallel result ordering.
- [ ] Track upstream root and subpackage API shape exactly while keeping safer
  total/owned alternatives as additions.
- [ ] Add generated differential/property coverage for every manifest row.

**Standard-library obligation:**

- [x] Initial forcing contributions: `std/nonempty` and `std/vec`.
- [x] Land a law-tested `std/iter` algebra with fallible folds, stable grouping,
  explicit ordered/unordered parallelism, cancellation, and opt-in fusion.
- [x] Release matching Go+/stdlib tags and remove the local `replace`.

**Release and site:**

- [x] Stage a truthful unreleased `/lo/` page; Hugo draft and production
  exclusion checks pass.
- [x] Create/publish the `goforge.dev/lo` repository and module endpoint.
- [x] Tag the first full-parity release after pulling and rerunning release
  gates.
- [x] Add and deploy `/lo/` on goforge.dev with the required release data.

**Performance target:**

- [x] Generated Go+ destination-buffer filtering/mapping is conservatively
  2.65× faster than upstream `FilterMap` with 100% fewer allocations and 4.86×
  faster than the equivalent two-helper pipeline.
- [ ] Meet or explicitly ledger the 2×/50% target across root, iterator,
  mutable, and parallel hot paths as complete parity lands.

### 5. Chi

**Why it is in Wave 1:** HTTP routing and middleware benefit from indexed
patterns, typed parameters, request/body ownership, explicit capabilities,
immutable route sets, and exhaustive resolution.

**Existing work:** The facade supports the complete public root inventory:
core routing, parameters, regex and wildcard patterns, middleware, grouping,
mounting, custom 404/405 handling, route traversal, deterministic immutable
compilation, and typed route snapshots. The request-lifecycle middleware slice
adds request IDs, trusted-header/CIDR/proxy-count client IP policies, legacy
real-IP normalization, panic recovery, timeouts, request-content policies,
authentication, response metadata, size limits, and request transformations.
Path cleaning, slash/prefix handling, HEAD fallback, heartbeats, and URL-format
extraction are also covered, as are header routing, response instrumentation,
throttling, profiling, compression, and logging. `std/http/route` contains the
Go+ semantic layer. Local tests pass.

**Go+ authorship status:**

- [x] Replace the 2,551-line handwritten router and middleware implementation
  with 14 idiomatic `.gp` sources and reproducible generated Go.
- [x] Express semantic route resolution as an exhaustive Go+ `MatchOutcome`
  enum while preserving ordinary-Go compatibility through generated variants.
- [x] Keep indexed patterns/parameters, exhaustive resolution, middleware
  capabilities, trusted-proxy evidence, and request ownership states in the
  authored Go+ layer rather than treating `std/http/route` as a wrapper around
  a Go implementation.
- [x] Preserve the 127-row compatibility surface, differential corpus, fuzz
  laws, and benchmark ledgers against the generated implementation.

**Parity work remaining:**

- [x] Correct the inventory to exclude `_examples` and methods on unexported
  receivers, reducing the public denominator from 178 to 127.
- [x] Implement all 54 consumer-nameable root declarations, including the
  externally callable zero-argument `MethodNotAllowedHandler`.
- [x] Implement all 73 `middleware` declarations at the upstream import path.
- [x] Correct global middleware execution so it precedes route selection,
  receives a route context, and covers 404/405 paths.
- [x] Preserve default `Allow` response headers and parent 404/405 fallback
  inheritance across mounts while giving explicit child fallbacks precedence.
- [x] Preserve prefix-scoped 404/405 handlers registered inside `Route`
  subrouters without leaking those fallbacks to unrelated paths.
- [x] Add method/shape/value differential fuzzing and correct terminal empty
  parameter matching while preserving Chi-compatible internal empty segments.
- [x] Preserve standard-library `Request.PathValue`, middle-wildcard
  `RoutePattern` normalization, and inline middleware metadata during `Walk`.
- [x] Support multiple and embedded parameters within one path segment,
  including literal affixes and nested-brace regex quantifiers.
- [x] Differentially verify and correct late middleware registration, custom
  methods, empty parameters, terminal wildcards, ambiguous parameter-shape
  replacement, nested mounts, mounted route traversal, and escaped URL
  parameters (including encoded slashes and wildcard tails).
- [ ] Differentially verify routing precedence, ambiguous registration,
  middleware order, context reuse, redirects, compression, throttling,
  authentication helpers, profiler behavior, and platform-sensitive handlers.
- [ ] Preserve upstream insertion/ambiguity behavior at compatibility entry
  points while retaining deterministic checked snapshots as the new API.
- [x] Regenerate the complete 127-row manifest and prove zero deferred rows.

**Standard-library obligation:**

- [x] Initial forcing contribution: `std/http/route` indexed patterns,
  parameters, route sets, handlers, and middleware capabilities.
- [ ] Add typed regex/refinement evidence, owned request-body states, and
  capability composition laws needed by complete middleware parity.
- [x] Release matching Go+/stdlib tags and remove the local `replace`.

**Release and site:**

- [x] Stage a truthful unreleased `/chi/` page; Hugo draft and production
  exclusion checks pass.
- [x] Create/publish the `goforge.dev/chi` repository and module endpoint.
- [x] Tag the first full-parity release after pulling and rerunning release
  gates.
- [x] Add and deploy `/chi/` on goforge.dev with the required release data.

**Performance target:**

- [x] Immutable exact-route dispatch is conservatively 5.22× faster with 100%
  fewer allocations than pinned upstream Chi.
- [x] Ledger the accepted content-type compatibility path: 30.40–33.59 ns
  versus upstream 30.21–30.31 ns, both at zero allocations. This path misses
  the speed target and needs a typed/fused policy alternative.
- [x] Remove temporary dynamic-path/value slices and pool route contexts:
  named-parameter dispatch is 1.26× faster with 50% fewer allocations;
  compatibility middleware is 1.05× faster with equal allocations. Both
  speed results remain explicit target misses.
- [ ] Extend the 2×/50% paired gate to dynamic/regex/wildcard routes,
  middleware chains, mounts, traversal, and compatibility dispatch.

## Release ledger

This table is append-only. A release is not recorded until its tag, module
resolution, clean-consumer test, stdlib tag, site deployment, and production
page have all been verified.

| Project | Project tag | Go+ tag | stdlib tag | Pinned upstream | Module resolution | Site URL | Evidence |
|---|---|---|---|---|---|---|---|
| Viper | v1.0.0 | v0.137.0 | std/v0.201.0 | v1.21.0 | brain-fuel/gpviper (`goforge.dev/gpviper`) | `/viper/` published | 192/192; released 2026-07-24 |
| Validator | v1.0.0 | v0.137.0 | std/v0.200.0 | v10.30.3 | brain-fuel/gpvalidator (`goforge.dev/gpvalidator`) | `/validator/` published | 106/106 + 210/210; released 2026-07-24 |
| GJSON | v1.0.0 | v0.137.0 | std/v0.201.0 | v1.19.0 | brain-fuel/gpgjson (`goforge.dev/gpgjson`) | `/gjson/` published | 45/45 + 1,301/1,301 paths; std/pathquery shipped; released 2026-07-24 |
| Lo | v1.0.0 | v0.137.0 | std/v0.200.0 | v1.53.0 | brain-fuel/gplodash (`goforge.dev/gplodash`) | `/lo/` published | 651/651 parity; released 2026-07-24 |
| Chi | v1.0.0 | v0.137.0 | std/v0.201.0 | v5.3.1 | brain-fuel/gpchi (`goforge.dev/gpchi`) | `/chi/` published | 127/127; released 2026-07-24 |
| goml (surface) | — | v0.144.1 | std/v0.212.0 | — | brain-fuel/goplus (`goforge.dev/goplus`, `cmd/goml`) | `/goplus/` published | ML-family second surface; 418 Godog scenarios, differential parity vs .gp, race + gen -check clean; released 2026-08-12 |
| goml REPL + value bindings | — | v0.145.0 | std/v0.212.0 | — | brain-fuel/goplus (`goforge.dev/goplus`, `cmd/goml`) | `/goplus/` published | `goml repl` compiles and runs (no interpreter); nullary `let` binds a value, `()` is the unit binder; 425 Godog scenarios, race + gen -check clean; released 2026-08-12 |
| goml Go-interop | — | v0.146.0 | std/v0.212.0 | — | brain-fuel/goplus (`goforge.dev/goplus`, `cmd/goml`) | `/goplus/` published | `&x`, `xs[i]`, channel send/recv, imported ctor patterns, if-as-statement; two wrong lowerings fixed (void if/else ran both arms; binders blanked inside record literals); forced by the knockknock C2C example; 434 scenarios, race + gen -check clean; released 2026-08-20 |
| Typed holes | — | v0.147.0 | std/v0.212.0 | — | brain-fuel/goplus (`goforge.dev/goplus`, `cmd/goplus` + `cmd/goml`) | `/goplus/` published | `?name` goals: un-erased dependent type + in-scope bindings incl. quantity-0 indices; refuses to write while any hole remains (module or not); LSP Information diagnostics + native hover; goml `:holes` and declared-signature `:type`; 433 scenarios, race + gen -check clean; released 2026-08-27 |

## Verification commands

Run these from the workspace root. Release automation may wrap them, but may
not weaken them.

```sh
for project in viper validator gjson lo chi; do
  (cd "$project" && go test ./...)
  (cd "$project" && go test -race ./...)
done

git -C goplus diff --check
(cd goplus && go test ./...)
```

Each project must additionally provide:

- a generated upstream inventory verifier;
- a differential conformance command;
- a fuzz/property command with a checked-in seed corpus;
- paired benchmark commands;
- a clean-module consumer test with no workspace or `replace` assistance;
- a site-link and production-response verifier.

The commands and their results belong in the project release notes and in the
corresponding goforge.dev page. This file records status; it does not substitute
for those artifacts.
