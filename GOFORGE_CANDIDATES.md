# GoForge Go+ candidate audit — top 200 MIT repositories

_Snapshot: 2026-07-23. This is a candidate landscape, not permission to copy code without repository-level due diligence._

## Method

This audit starts with GitHub repositories whose primary language is Go, Python, or Java and whose repository license is currently classified as SPDX `MIT`. MIT is a hard gate: anything not classified MIT was excluded. The final set removes tutorials, curated lists, interview collections, and obvious mirrors, then ranks the remaining projects with this weighting:

- **Go+ readability/performance leverage — 45 points.** Preference goes to protocols, parsers, state machines, workflows, validation, configuration, storage, typed clients, streaming, and concurrency where sums, refinements, indices, ownership, generation, and discipline materially improve the design.
- **Industry/adoption signal — 40 points.** GitHub stars and forks are reproducible public proxies, not proof of enterprise deployment. Libraries with broad ecosystem roles receive a small domain-specific boost.
- **Potential Go+ standard-library contribution — 15 points.** Credit is conservative and limited to general code or laws likely to serve multiple independent consumers.

The ranking is intentionally not raw star order. A less-famous validation or parser library can rank above a larger application when it is a stronger semantic forcing case. Before bringing any project under GoForge, re-check the exact tag, `LICENSE`, dependency licenses, generated/vendor trees, trademarks, patents, data/model licenses, and compatibility target. Repository-level MIT status does not make every bundled artifact MIT.

### Portfolio

- 200 candidates: 86 Go, 74 Python, 40 Java.
- Every entry passed the repository-level MIT gate in the 2026-07-23 GitHub metadata snapshot.
- A second license-text audit on 2026-07-24 found 186 canonical MIT grants at conventional root filenames, 11 semantically identical line-wrapped or headed MIT grants, and 3 MIT grants under alternate filenames or casing (`classgraph/classgraph`, `theonedev/onedev`, and `dreamhead/moco`). No candidate relies solely on a README badge or repository-search label for the hard gate.
- Scores are comparative triage aids, not claims that a rewrite should preserve every upstream API.

## Ranked candidates

### 1. [spf13/viper](https://github.com/spf13/viper) — 100/100

- **Why this library or tool exists:** Programs need deterministic configuration, provenance, and typed access without mutable package globals or stringly typed keys.
- **How it works:** It merges defaults, files, environment variables, flags, remote providers, and explicit overrides through a precedence chain, then decodes the resulting key space into typed values or structs while supporting aliases and live reloads.
- **What it does:** Go configuration with fangs.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 30,372 stars and 2,173 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use schema-indexed snapshots, typed keys, exhaustive provenance, and presence witnesses. Discipline separates reload effects from immutable reads and enables precompiled decoding paths.
- **Possible Go+ standard-library pressure:** `std/config`: source provenance, immutable snapshots, typed keys, deterministic precedence, and schema projection.

### 2. [pydantic/pydantic](https://github.com/pydantic/pydantic) — 100/100

- **Why this library or tool exists:** Applications need validation facts that survive beyond a boolean/error return and can be composed without losing field paths or predicate identity.
- **How it works:** Fast and extensible, Pydantic plays nicely with your linters/IDE/brain. Define how data should be in pure, canonical Python 3.10+; validate it with Pydantic.
- **What it does:** Data validation using Python type hints.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 28,365 stars and 2,809 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use `Rule[T,P]`, exhaustive failure trees, predicate-indexed `Validated[T,P]`, lawful conjunction, and proof-preserving transforms. Generated Go can cache adapters and remove repeated reflection on hot paths.
- **Possible Go+ standard-library pressure:** `std/validate`: typed rules, field paths, failure trees, predicate witnesses, and safe ordinary-Go revalidation boundaries.

### 3. [go-playground/validator](https://github.com/go-playground/validator) — 100/100

- **Why this library or tool exists:** Applications need validation facts that survive beyond a boolean/error return and can be composed without losing field paths or predicate identity.
- **How it works:** Package validator implements value validations for structs and individual fields based on tags.
- **What it does:** :100:Go Struct and Field validation, including Cross Field, Cross Struct, Map, Slice and Array diving.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 20,088 stars and 1,444 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use `Rule[T,P]`, exhaustive failure trees, predicate-indexed `Validated[T,P]`, lawful conjunction, and proof-preserving transforms. Generated Go can cache adapters and remove repeated reflection on hot paths.
- **Possible Go+ standard-library pressure:** `std/validate`: typed rules, field paths, failure trees, predicate witnesses, and safe ordinary-Go revalidation boundaries.

### 4. [evanw/esbuild](https://github.com/evanw/esbuild) — 98.5/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** The main goal of the esbuild bundler project is to bring about a new era of build tool performance, and create an easy-to-use modern bundler along the way.
- **What it does:** An extremely fast bundler for the web.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 39,994 stars and 1,324 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 5. [temporalio/temporal](https://github.com/temporalio/temporal) — 96.5/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** Temporal is a durable execution platform that enables developers to build scalable applications without sacrificing productivity or reliability. The Temporal server executes units of application logic called Workflows in a resilient manner that automatically handles intermittent failures, and retries failed operations.
- **What it does:** Temporal service.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 21,809 stars and 1,760 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 6. [tidwall/gjson](https://github.com/tidwall/gjson) — 96.4/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** GJSON is a Go package that provides a fast and simple way to get values from a json document. It has features such as one line retrieval, dot notation paths, iteration, and parsing json lines.
- **What it does:** Get JSON values quickly - JSON parser for Go.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 15,546 stars and 906 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 7. [microsoft/markitdown](https://github.com/microsoft/markitdown) — 96.1/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** It dispatches each input format to a converter, extracts text and structural elements such as headings, tables, links, and metadata, and emits a normalized Markdown stream suitable for indexing or model input.
- **What it does:** Python tool for converting files and office documents to Markdown.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 168,563 stars and 12,169 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 8. [sqlc-dev/sqlc](https://github.com/sqlc-dev/sqlc) — 95.9/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** 1. You write queries in SQL. 1. You run sqlc to generate code with type-safe interfaces to those queries. 1. You write application code that calls the generated code.
- **What it does:** Generate type-safe code from SQL.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 18,061 stars and 1,074 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 9. [langflow-ai/langflow](https://github.com/langflow-ai/langflow) — 95.8/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** Users assemble typed components into a directed workflow graph; the runtime resolves component inputs, invokes models, tools, and data sources, streams events, and exposes the completed graph through API and MCP servers.
- **What it does:** Langflow is a powerful tool for building and deploying AI-powered agents and workflows.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 152,289 stars and 9,636 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 10. [samber/lo](https://github.com/samber/lo) — 95.4/100

- **Why this library or tool exists:** Programs need a compact, lawful vocabulary for transforming collections without obscuring allocation, ordering, error, or ownership behavior.
- **How it works:** A utility library based on Go 1.18+ generics that makes it easier to work with slices, maps, strings, channels, and functions. It provides dozens of handy methods to simplify common coding tasks and improve code readability. It may look like Lodash in some aspects.
- **What it does:** 💥 A Lodash-style Go library based on Go 1.18+ Generics (map, filter, contains, find...).
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 21,393 stars and 946 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing iterator and collection abstractions, fusion for pure pipelines, explicit ordered/unordered parallel forms, and ownership-aware views. Specialization can eliminate closures and intermediate slices in hot paths.
- **Possible Go+ standard-library pressure:** A small `std/iter` algebra with documented laws, fallible folds, stable grouping, and opt-in fusion; richer collection conveniences should remain external.

### 11. [github/spec-kit](https://github.com/github/spec-kit) — 95.1/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** An open source toolkit for building high-quality software with any AI coding agent — a ready-to-use spec-driven process (or bring your own), endlessly extensible, community-driven, and built for your whole organization.
- **What it does:** 💫 Toolkit to help you get started with Spec-Driven Development.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 123,497 stars and 11,016 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 12. [jhy/jsoup](https://github.com/jhy/jsoup) — 94.4/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** jsoup is a Java library that makes it easy to work with real-world HTML and XML. It offers an easy-to-use API for URL fetching, data parsing, extraction, and manipulation using DOM API methods, CSS, and xpath selectors.
- **What it does:** jsoup: the Java HTML parser, built for HTML editing, cleaning, scraping, and XSS safety.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 11,379 stars and 2,294 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 13. [bytedance/deer-flow](https://github.com/bytedance/deer-flow) — 93.6/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** DeerFlow (Deep Exploration and Efficient Research Flow) is an open-source super agent harness that orchestrates sub-agents, memory, and sandboxes to do almost anything — powered by extensible skills.
- **What it does:** An open-source long-horizon SuperAgent harness that researches, codes, and creates. With the help of sandboxes, memories, tools, skill, subagents and message gateway, it handles different levels of tasks that could take minutes to hours.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 77,705 stars and 10,587 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 14. [nektos/act](https://github.com/nektos/act) — 93.3/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** It parses GitHub Actions workflow YAML, resolves jobs and actions, prepares GitHub-compatible event and environment data, and runs job steps locally in containers with dependency and artifact handling.
- **What it does:** Run your GitHub Actions locally 🚀.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 71,189 stars and 1,981 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 15. [docling-project/docling](https://github.com/docling-project/docling) — 93.0/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** Docling simplifies document processing by parsing diverse formats — including advanced PDF understanding — and providing seamless integrations with the generative AI ecosystem.
- **What it does:** Get your documents ready for gen AI.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 63,687 stars and 4,514 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 16. [uber/NullAway](https://github.com/uber/NullAway) — 93.0/100

- **Why this library or tool exists:** Applications need validation facts that survive beyond a boolean/error return and can be composed without losing field paths or predicate identity.
- **How it works:** As an Error Prone checker, it combines `@Nullable` contracts with intraprocedural data-flow analysis and initialization checks, rejecting dereferences or assignments that cannot be proven non-null at compile time.
- **What it does:** A tool to help eliminate NullPointerExceptions (NPEs) in your Java code with low build-time overhead.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 4,081 stars and 351 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use `Rule[T,P]`, exhaustive failure trees, predicate-indexed `Validated[T,P]`, lawful conjunction, and proof-preserving transforms. Generated Go can cache adapters and remove repeated reflection on hot paths.
- **Possible Go+ standard-library pressure:** `std/validate`: typed rules, field paths, failure trees, predicate witnesses, and safe ordinary-Go revalidation boundaries.

### 17. [pocketbase/pocketbase](https://github.com/pocketbase/pocketbase) — 92.8/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** - embedded database (SQLite) with realtime subscriptions - built-in files and users management - convenient Admin dashboard UI - and simple REST-ish API
- **What it does:** Open Source realtime backend in 1 file.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 60,184 stars and 3,582 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 18. [crewAIInc/crewAI](https://github.com/crewAIInc/crewAI) — 92.6/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** - CrewAI Crews: Optimize for autonomy and collaborative intelligence with role-based AI agents. - CrewAI Flows: Build event-driven automations that combine precise workflow control, single LLM calls, and native support for Crews.
- **What it does:** Framework for orchestrating role-playing, autonomous AI agents. By fostering collaborative intelligence, CrewAI empowers agents to work together seamlessly, tackling complex tasks.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 56,036 stars and 7,927 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 19. [HKUDS/nanobot](https://github.com/HKUDS/nanobot) — 91.9/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** 🐈 nanobot is an open-source, ultra-lightweight personal AI agent you can truly own. It keeps the agent core small and readable while giving you the practical pieces for real long-running work: WebUI, chat channels, tools, memory, MCP, model routing, automation, and deployment.
- **What it does:** Lightweight, open-source AI agent for your tools, chats, and workflows.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 46,133 stars and 8,155 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 20. [psf/black](https://github.com/psf/black) — 91.6/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** Black is the uncompromising Python code formatter. By using it, you agree to cede control over minutiae of hand-formatting. In return, Black gives you speed, determinism, and freedom from pycodestyle nagging about formatting. You will save time and mental energy for more important matters.
- **What it does:** The uncompromising Python code formatter.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 41,757 stars and 2,825 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 21. [langchain-ai/langgraph](https://github.com/langchain-ai/langgraph) — 91.3/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** Trusted by companies shaping the future of agents – including Klarna, Replit, Elastic, and more – LangGraph is a low-level orchestration framework for building, managing, and deploying long-running, stateful agents.
- **What it does:** Build resilient agents.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 37,956 stars and 6,380 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 22. [twpayne/chezmoi](https://github.com/twpayne/chezmoi) — 91.3/100

- **Why this library or tool exists:** Programs need deterministic configuration, provenance, and typed access without mutable package globals or stringly typed keys.
- **How it works:** Parse multiple sources into immutable snapshots, apply a deterministic precedence relation, retain source provenance, and decode through typed schemas.
- **What it does:** Manage your dotfiles across multiple diverse machines, securely.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 20,803 stars and 670 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use schema-indexed snapshots, typed keys, exhaustive provenance, and presence witnesses. Discipline separates reload effects from immutable reads and enables precompiled decoding paths.
- **Possible Go+ standard-library pressure:** `std/config`: source provenance, immutable snapshots, typed keys, deterministic precedence, and schema projection.

### 23. [fxsjy/jieba](https://github.com/fxsjy/jieba) — 91.0/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** "Jieba" (Chinese for "to stutter") Chinese text segmentation: built to be the best Python Chinese word segmentation module.
- **What it does:** 结巴中文分词.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 35,078 stars and 6,693 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 24. [python-poetry/poetry](https://github.com/python-poetry/poetry) — 91.0/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** Poetry replaces setup.py, requirements.txt, setup.cfg, MANIFEST.in and Pipfile with a simple pyproject.toml based project format.
- **What it does:** Python packaging and dependency management made easy.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 34,292 stars and 2,462 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 25. [magic-wormhole/magic-wormhole](https://github.com/magic-wormhole/magic-wormhole) — 90.6/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** This package provides a library and a command-line tool named wormhole, which makes it possible to get arbitrary-sized files and directories (or short pieces of text) from one computer to another. The two endpoints are identified by using identical "wormhole codes": in general, the sending machine generates and displays the code, which must then be typed...
- **What it does:** get things from one computer to another, safely.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 22,745 stars and 750 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 26. [go-chi/chi](https://github.com/go-chi/chi) — 90.6/100

- **Why this library or tool exists:** Service authors need routing, middleware, request decoding, and response ownership that remain fast without making invalid combinations easy.
- **How it works:** chi is a lightweight, idiomatic and composable router for building Go HTTP services. It's especially good at helping you write large REST API services that are kept maintainable as your project grows and changes. chi is built on the new context package introduced in Go 1.7 to handle signaling, cancelation and request-scoped values across a handler chain.
- **What it does:** lightweight, idiomatic and composable router for building Go HTTP services.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 22,572 stars and 1,136 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use pattern-indexed routes, capability-explicit middleware, typed parameter environments, request/body typestates, and exhaustive response outcomes. Compile immutable routing tables and binders ahead of time.
- **Possible Go+ standard-library pressure:** `std/http/route`, typed request/response ownership, middleware capabilities, and protocol-neutral status/decode outcomes.

### 27. [pion/webrtc](https://github.com/pion/webrtc) — 90.6/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** It implements the WebRTC protocol stack in Go—ICE, STUN/TURN, DTLS, SCTP, SRTP, RTP/RTCP, media tracks, and data channels—coordinated through peer-connection negotiation and explicit transport state machines.
- **What it does:** Pure Go implementation of the WebRTC API.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 16,660 stars and 1,872 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 28. [mikefarah/yq](https://github.com/mikefarah/yq) — 90.4/100

- **Why this library or tool exists:** Programs need deterministic configuration, provenance, and typed access without mutable package globals or stringly typed keys.
- **How it works:** A lightweight and portable command-line YAML, JSON, INI and XML processor. yq uses jq (a popular JSON processor) like syntax but works with yaml files as well as json, kyaml, xml, ini, properties, csv and tsv. It doesn't yet support everything jq does - but it does support the most common operations and functions, and more is being added continuously.
- **What it does:** yq is a portable command-line YAML, JSON, XML, CSV, TOML, HCL and properties processor.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 15,737 stars and 799 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use schema-indexed snapshots, typed keys, exhaustive provenance, and presence witnesses. Discipline separates reload effects from immutable reads and enables precompiled decoding paths.
- **Possible Go+ standard-library pressure:** `std/config`: source provenance, immutable snapshots, typed keys, deterministic precedence, and schema projection.

### 29. [openai/openai-agents-python](https://github.com/openai/openai-agents-python) — 90.3/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** The OpenAI Agents SDK is a lightweight yet powerful framework for building multi-agent workflows. It is provider-agnostic, supporting the OpenAI Responses and Chat Completions APIs, as well as 100+ other LLMs.
- **What it does:** A lightweight, powerful framework for multi-agent workflows.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 28,122 stars and 4,369 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 30. [direnv/direnv](https://github.com/direnv/direnv) — 90.3/100

- **Why this library or tool exists:** Programs need deterministic configuration, provenance, and typed access without mutable package globals or stringly typed keys.
- **How it works:** direnv is an extension for your shell. It augments existing shells with a new feature that can load and unload environment variables depending on the current directory.
- **What it does:** unclutter your .profile.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 15,300 stars and 802 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use schema-indexed snapshots, typed keys, exhaustive provenance, and presence witnesses. Discipline separates reload effects from immutable reads and enables precompiled decoding paths.
- **Possible Go+ standard-library pressure:** `std/config`: source provenance, immutable snapshots, typed keys, deterministic precedence, and schema projection.

### 31. [traefik/traefik](https://github.com/traefik/traefik) — 90.0/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** Traefik (pronounced traffic) is a modern HTTP reverse proxy and load balancer that makes deploying microservices easy. Traefik integrates with your existing infrastructure components (Docker, Swarm mode, Kubernetes, Consul, Etcd, Rancher v2, Amazon ECS, ...) and configures itself automatically and dynamically. Pointing Traefik at your orchestrator should...
- **What it does:** The Cloud Native Application Proxy.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 64,083 stars and 6,087 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 32. [pypa/pipenv](https://github.com/pypa/pipenv) — 89.9/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** Pipenv: Python Development Workflow for Humans ==============================================
- **What it does:** Python Development Workflow for Humans.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 25,047 stars and 1,880 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 33. [Mojang/brigadier](https://github.com/Mojang/brigadier) — 89.7/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** Brigadier is available to Maven & Gradle via libraries.minecraft.net. Its group is com.mojang, and artifact name is brigadier.
- **What it does:** Brigadier is a command parser & dispatcher, designed and developed for Minecraft: Java Edition.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 3,710 stars and 408 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 34. [gin-gonic/gin](https://github.com/gin-gonic/gin) — 89.1/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** We're excited to announce the release of Gin 1.12.0! This release brings new features, performance improvements, and important bug fixes. Check out the release announcement on our official blog for the full details.
- **What it does:** Gin is a high-performance HTTP web framework written in Go. It provides a Martini-like API but with significantly better performance—up to 40 times faster—thanks to httprouter. Gin is designed for building REST APIs, web applications, and microservices.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 88,945 stars and 8,649 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 35. [rqlite/rqlite](https://github.com/rqlite/rqlite) — 88.8/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Use rqlite to reliably store your most important data, ensuring it's always available to your applications -- think etcd, but with relational modeling available. Whether you're deploying resilient services in the cloud or reliable applications at the edge, rqlite provides a robust solution for critical data.
- **What it does:** The lightweight, fault-tolerant database built on SQLite. Designed to keep your data highly available with minimal effort.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 17,644 stars and 797 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 36. [fastapi/fastapi](https://github.com/fastapi/fastapi) — 88.5/100

- **Why this library or tool exists:** Service authors need routing, middleware, request decoding, and response ownership that remain fast without making invalid combinations easy.
- **How it works:** FastAPI framework, high performance, easy to learn, fast to code, ready for production
- **What it does:** FastAPI framework, high performance, easy to learn, fast to code, ready for production.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 100,808 stars and 9,673 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use pattern-indexed routes, capability-explicit middleware, typed parameter environments, request/body typestates, and exhaustive response outcomes. Compile immutable routing tables and binders ahead of time.
- **Possible Go+ standard-library pressure:** `std/http/route`, typed request/response ownership, middleware capabilities, and protocol-neutral status/decode outcomes.

### 37. [goreleaser/goreleaser](https://github.com/goreleaser/goreleaser) — 88.5/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** We handle the complexities of releasing so you can focus in building what really matters: your software.
- **What it does:** Release engineering, simplified.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 15,956 stars and 1,094 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 38. [go-task/task](https://github.com/go-task/task) — 88.5/100

- **Why this library or tool exists:** Long-running work needs resumability, idempotency, cancellation, and explicit transition outcomes across unreliable effect boundaries.
- **How it works:** A fast, cross-platform build tool inspired by Make, designed for modern workflows.
- **What it does:** A fast, cross-platform build tool inspired by Make, designed for modern workflows.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 15,870 stars and 869 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use exhaustive transition sums, capability-checked effects, saga journals, CAS observations, and typestates for lifecycle. Deterministic scheduling and compact event logs can reduce coordination and allocation cost.
- **Possible Go+ standard-library pressure:** `std/workflow`, `std/cas`, `std/process`, and `std/fsatomic`: journals, observed mutation, cancellation, redaction, and durable replacement.

### 39. [schollz/croc](https://github.com/schollz/croc) — 88.3/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** croc is a tool that allows any two computers to simply and securely transfer files and folders. AFAIK, croc is the only CLI file-transfer tool that does all of the following:
- **What it does:** Easily and securely send things from one computer to another :crocodile: :package:.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 38,103 stars and 1,510 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 40. [explosion/spaCy](https://github.com/explosion/spaCy) — 87.9/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** spaCy is a library for advanced Natural Language Processing in Python and Cython. It's built on the very latest research, and was designed from day one to be used in real products.
- **What it does:** 💫 Industrial-strength Natural Language Processing (NLP) in Python.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 33,768 stars and 4,699 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 41. [rclone/rclone](https://github.com/rclone/rclone) — 87.7/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Rclone ("rsync for cloud storage") is a command-line program to sync files and directories to and from different cloud storage providers.
- **What it does:** "rsync for cloud storage" - Google Drive, S3, Dropbox, Backblaze B2, One Drive, Swift, Hubic, Wasabi, Google Cloud Storage, Azure Blob, Azure Files, Yandex Files.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 58,664 stars and 5,245 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 42. [run-llama/llama_index](https://github.com/run-llama/llama_index) — 87.3/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** LlamaIndex OSS (by LlamaIndex) is an open-source framework to build agentic applications. Parse is our enterprise platform for agentic OCR, parsing, extraction, indexing and more. You can use LlamaParse with this framework or on its own; see LlamaParse below for signup and product links.
- **What it does:** LlamaIndex is the leading document agent and OCR platform.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 51,040 stars and 7,803 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 43. [gogs/gogs](https://github.com/gogs/gogs) — 87.0/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Map requests to transactions or log operations, maintain indexes/caches, encode durable records, and expose consistency and failure through a client or query layer.
- **What it does:** The painless way to host your own Git service.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 47,693 stars and 5,075 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 44. [microsoft/qlib](https://github.com/microsoft/qlib) — 87.0/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** We are excited to announce the release of RD-Agent📢, a powerful tool that supports automated factor mining and model optimization in quant investment R&D.
- **What it does:** Qlib is an AI-oriented Quant investment platform that aims to use AI tech to empower Quant Research, from exploring ideas to implementing productions. Qlib supports diverse ML modeling paradigms, including supervised learning, market dynamics modeling, and RL, and is now equipped with https://github.com/microsoft/RD-Agent to automate R&D process.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 46,564 stars and 7,409 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 45. [pyg-team/pytorch_geometric](https://github.com/pyg-team/pytorch_geometric) — 86.8/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** Documentation | PyG 1.0 Paper | PyG 2.0 Paper | Colab Notebooks | External Resources | OGB Examples
- **What it does:** Graph Neural Network Library for PyTorch.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 23,913 stars and 4,020 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 46. [modelcontextprotocol/python-sdk](https://github.com/modelcontextprotocol/python-sdk) — 86.8/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** It has a Get started guide, What's new in v2, the API reference, and the migration guide.
- **What it does:** The official Python SDK for Model Context Protocol servers and clients.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 23,700 stars and 3,688 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 47. [ccxt/ccxt](https://github.com/ccxt/ccxt) — 86.7/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** A crypto trading API with more than 100 exchanges and prediction markets in JavaScript / TypeScript / Python / C# / PHP / Go / Java.
- **What it does:** A unified trading API with more than 100 crypto exchanges and prediction markets in JavaScript / TypeScript / Python / C# / PHP / Go / Java.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 43,380 stars and 8,754 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 48. [gofiber/fiber](https://github.com/gofiber/fiber) — 86.5/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Fiber is an Express inspired web framework built on top of Fasthttp , the fastest HTTP engine for Go . Designed to ease things up for fast development with zero memory allocation and performance in mind.
- **What it does:** ⚡️ Express inspired web framework written in Go.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 40,010 stars and 2,012 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 49. [go-gorm/gorm](https://github.com/go-gorm/gorm) — 86.5/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Full-Featured ORM Associations (Has One, Has Many, Belongs To, Many To Many, Polymorphism, Single-table inheritance) Hooks (Before/After Create/Save/Update/Delete/Find) Eager loading with Preload, Joins Transactions, Nested Transactions, Save Point, RollbackTo to Saved Point Context, Prepared Statement Mode, DryRun Mode Batch Insert, FindInBatches, Find...
- **What it does:** The fantastic ORM library for Golang, aims to be developer friendly.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 39,884 stars and 4,165 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 50. [apernet/hysteria](https://github.com/apernet/hysteria) — 86.5/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** 🛠️ Jack of all trades Wide range of modes including SOCKS5, HTTP Proxy, TCP/UDP Forwarding, Linux TProxy, TUN - with more features being added constantly.
- **What it does:** Hysteria is a powerful, lightning fast and censorship resistant proxy.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 22,161 stars and 2,239 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 51. [bluenviron/mediamtx](https://github.com/bluenviron/mediamtx) — 86.1/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** MediaMTX is a ready-to-use and zero-dependency live media server and media proxy that allows to publish, read, proxy, record and playback real-time video and audio streams. It has been conceived as a "media router" that routes media streams from one end to the other, with a focus on efficiency and portability.
- **What it does:** Ready-to-use Media-over-QUIC / SRT / WebRTC / RTSP / RTMP / LL-HLS / MPEG-TS / RTP live media server and media proxy that allows to read, publish, proxy, record and playback real-time video and audio streams.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 19,602 stars and 2,306 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 52. [yudai/gotty](https://github.com/yudai/gotty) — 86.1/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** GoTTY is a simple command line tool that turns your CLI tools into web applications.
- **What it does:** Share your terminal as a web application.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 19,518 stars and 1,410 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 53. [NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent) — 86.0/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** The self-improving AI agent built by Nous Research. It's the only agent with a built-in learning loop — it creates skills from experience, improves them during use, nudges itself to persist knowledge, searches its own past conversations, and builds a deepening model of who you are across sessions. Run it on a $5 VPS, a GPU cluster, or serverless...
- **What it does:** The agent that grows with you.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 219,500 stars and 41,655 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 54. [sherlock-project/sherlock](https://github.com/sherlock-project/sherlock) — 86.0/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** Generate or execute isolated scenarios, capture structured observations, compare them with expectations, and report reproducible diagnostics.
- **What it does:** Hunt down social media accounts by username across social networks.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 86,993 stars and 10,204 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 55. [v2fly/v2ray-core](https://github.com/v2fly/v2ray-core) — 86.0/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Project V Project V is a set of network tools that helps you to build your own computer network. It secures your network connections and thus protects your privacy.
- **What it does:** A platform for building proxies to bypass network restrictions.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 34,363 stars and 5,082 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 56. [VectifyAI/PageIndex](https://github.com/VectifyAI/PageIndex) — 86.0/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** - 🔥 Agentic Vectorless RAG — A simple agentic, vectorless RAG example with self-hosted PageIndex, using OpenAI Agents SDK. - Scale PageIndex to Millions of Documents — PageIndex File System is a file-level tree indexing layer that lets PageIndex reason over an entire corpus, not just a single document, enabling massive-scale document search. - PageIndex...
- **What it does:** 📑 PageIndex: Document Index for Vectorless, Reasoning-based RAG.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 34,194 stars and 2,990 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 57. [ginuerzh/gost](https://github.com/ginuerzh/gost) — 85.9/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** 多端口监听 可设置转发代理，支持多级转发(代理链) 支持标准HTTP/HTTPS/HTTP2/SOCKS4(A)/SOCKS5代理协议 Web代理支持探测防御 支持多种隧道类型 SOCKS5代理支持TLS协商加密 Tunnel UDP over TCP TCP/UDP透明代理 本地/远程TCP/UDP端口转发 支持Shadowsocks(TCP/UDP)协议 支持SNI代理 权限控制 负载均衡 路由控制 DNS解析和代理 TUN/TAP设备
- **What it does:** GO Simple Tunnel - a simple tunnel written in golang.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 18,126 stars and 2,642 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 58. [Col-E/Recaf](https://github.com/Col-E/Recaf) — 85.9/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** An easy to use modern Java bytecode editor that abstracts away the complexities of Java programs.
- **What it does:** The modern Java bytecode editor.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 7,295 stars and 532 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 59. [junegunn/fzf](https://github.com/junegunn/fzf) — 85.8/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** fzf is a general-purpose command-line fuzzy finder and an interactive terminal toolkit.
- **What it does:** :cherry_blossom: A command-line fuzzy finder.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 81,935 stars and 2,820 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 60. [labstack/echo](https://github.com/labstack/echo) — 85.8/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Echo is built on Go's standard net/http — and interoperates with it via echo.WrapHandler / echo.WrapMiddleware — adding the parts the standard library leaves to you: a fast radix-tree router, request binding (with a pluggable validator), a deep middleware ecosystem, and centralized error handling. Actively maintained, with v5 as the current release line...
- **What it does:** High performance, minimalist Go web framework.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 32,548 stars and 2,344 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 61. [stanford-oval/storm](https://github.com/stanford-oval/storm) — 85.6/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** - [2025/01] We add litellm integration for language models and embedding models in knowledge-storm v1.1.0.
- **What it does:** An LLM-powered knowledge curation system that researches a topic and generates a full-length report with citations.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 30,283 stars and 2,830 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 62. [jpillora/chisel](https://github.com/jpillora/chisel) — 85.5/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** Chisel is a fast TCP/UDP tunnel, transported over HTTP, secured via SSH. Single executable including both client and server. Written in Go (golang). Chisel is mainly useful for passing through firewalls, though it can also be used to provide a secure endpoint into your network.
- **What it does:** A fast TCP/UDP tunnel over HTTP.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 16,276 stars and 1,602 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 63. [micro-editor/micro](https://github.com/micro-editor/micro) — 85.4/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** micro is a terminal-based text editor that aims to be easy to use and intuitive, while also taking advantage of the capabilities of modern terminals. It comes as a single, batteries-included, static binary with no dependencies; you can download and use it right now!
- **What it does:** A modern and intuitive terminal-based text editor.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 29,088 stars and 1,342 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 64. [graphql-java/graphql-java](https://github.com/graphql-java/graphql-java) — 85.4/100

- **Why this library or tool exists:** Service authors need routing, middleware, request decoding, and response ownership that remain fast without making invalid combinations easy.
- **How it works:** It parses and validates GraphQL schemas and operations, builds an execution plan, invokes field data fetchers, applies instrumentation, and assembles typed response data and structured errors.
- **What it does:** GraphQL Java implementation.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 6,228 stars and 1,141 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use pattern-indexed routes, capability-explicit middleware, typed parameter environments, request/body typestates, and exhaustive response outcomes. Compile immutable routing tables and binders ahead of time.
- **Possible Go+ standard-library pressure:** `std/http/route`, typed request/response ownership, middleware capabilities, and protocol-neutral status/decode outcomes.

### 65. [ollama/ollama](https://github.com/ollama/ollama) — 85.3/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** You'll be prompted to run a model or connect Ollama to your existing agents or applications such as Claude Code, OpenClaw, OpenCode , Codex, Copilot, and more.
- **What it does:** Get up and running with Kimi-K2.6, GLM-5.2, MiniMax, DeepSeek, gpt-oss, Qwen, Gemma and other models.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 176,733 stars and 17,078 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 66. [passteque/gluetun](https://github.com/passteque/gluetun) — 85.3/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** ⚠️ This and gluetun-wiki are the only websites for Gluetun, other websites claiming to be official are scams ⚠️
- **What it does:** VPN client in a thin Docker container for multiple VPN providers, written in Go, and using OpenVPN or Wireguard, DNS over TLS, with a few proxy servers built-in.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 14,945 stars and 600 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 67. [oauth2-proxy/oauth2-proxy](https://github.com/oauth2-proxy/oauth2-proxy) — 85.2/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** OAuth2 Proxy is a flexible, open-source tool that can act as either a standalone reverse proxy or a middleware component integrated into existing reverse proxy or load balancer setups. It provides a simple and secure way to protect your web applications with OAuth2 / OIDC authentication. As a reverse proxy, it intercepts requests to your application and...
- **What it does:** A reverse proxy that provides authentication with Google, Azure, OpenID Connect and many more identity providers.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 14,711 stars and 2,156 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 68. [nsqio/nsq](https://github.com/nsqio/nsq) — 85.0/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Producers publish messages to `nsqd`; topics fan out into channels, consumers pull with in-flight timeouts and requeue semantics, and `nsqlookupd` supplies decentralized service discovery without a central broker coordinator.
- **What it does:** A realtime distributed messaging platform.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 25,756 stars and 2,891 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 69. [zeromicro/go-zero](https://github.com/zeromicro/go-zero) — 84.9/100

- **Why this library or tool exists:** Service authors need routing, middleware, request decoding, and response ownership that remain fast without making invalid combinations easy.
- **How it works:** go-zero is a web and rpc framework with lots of builtin engineering practices. It’s born to ensure the stability of the busy services with resilience design and has been serving sites with tens of millions of users for years.
- **What it does:** A cloud-native Go microservices framework with cli tool for productivity.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 33,213 stars and 4,312 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use pattern-indexed routes, capability-explicit middleware, typed parameter environments, request/body typestates, and exhaustive response outcomes. Compile immutable routing tables and binders ahead of time.
- **Possible Go+ standard-library pressure:** `std/http/route`, typed request/response ownership, middleware capabilities, and protocol-neutral status/decode outcomes.

### 70. [karpathy/minGPT](https://github.com/karpathy/minGPT) — 84.9/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** A PyTorch re-implementation of GPT, both training and inference. minGPT tries to be small, clean, interpretable and educational, as most of the currently available GPT model implementations can a bit sprawling. GPT is not a complicated model and this implementation is appropriately about 300 lines of code (see mingpt/model.py). All that's going on is...
- **What it does:** A minimal PyTorch re-implementation of the OpenAI GPT (Generative Pretrained Transformer) training.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 24,731 stars and 3,298 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 71. [SYSTRAN/faster-whisper](https://github.com/SYSTRAN/faster-whisper) — 84.9/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** faster-whisper is a reimplementation of OpenAI's Whisper model using CTranslate2, which is a fast inference engine for Transformer models.
- **What it does:** Faster Whisper transcription with CTranslate2.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 24,484 stars and 1,992 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 72. [plotly/dash](https://github.com/plotly/dash) — 84.8/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Built on top of Plotly.js, React and Flask, Dash ties modern UI elements like dropdowns, sliders, and graphs directly to your analytical Python code. Read our tutorial (proudly crafted ❤️ with Dash itself).
- **What it does:** Data Apps & Dashboards for Python. No JavaScript Required.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 24,341 stars and 2,308 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 73. [jenkinsci/configuration-as-code-plugin](https://github.com/jenkinsci/configuration-as-code-plugin) — 84.8/100

- **Why this library or tool exists:** Programs need deterministic configuration, provenance, and typed access without mutable package globals or stringly typed keys.
- **How it works:** Setting up Jenkins is a complex process, as both Jenkins and its plugins require some tuning and configuration, with dozens of parameters to set within the web UI manage section.
- **What it does:** Jenkins Configuration as Code Plugin.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 2,790 stars and 752 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use schema-indexed snapshots, typed keys, exhaustive provenance, and presence witnesses. Discipline separates reload effects from immutable reads and enables precompiled decoding paths.
- **Possible Go+ standard-library pressure:** `std/config`: source provenance, immutable snapshots, typed keys, deterministic precedence, and schema projection.

### 74. [valyala/fasthttp](https://github.com/valyala/fasthttp) — 84.7/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** fasthttp was designed for some high performance edge cases. Unless your server/client needs to handle thousands of small to medium requests per second and needs a consistent low millisecond response time fasthttp might not be for you. For most cases net/http is much better as it's easier to use and can handle more cases. For most cases you won't even...
- **What it does:** Fast HTTP package for Go. Tuned for high performance. Zero memory allocations in hot paths. Up to 10x faster than net/http.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 23,419 stars and 1,841 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 75. [JanDeDobbeleer/oh-my-posh](https://github.com/JanDeDobbeleer/oh-my-posh) — 84.7/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** A shell hook gathers contextual segments such as Git state, runtime versions, path, and exit status; a theme configuration formats those segments and renders the prompt consistently across supported shells.
- **What it does:** The most customisable and low-latency cross platform/shell prompt renderer.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 23,139 stars and 2,765 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 76. [langchain-ai/langchain](https://github.com/langchain-ai/langchain) — 84.6/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** LangChain is a framework for building agents and LLM-powered applications. It helps you chain together interoperable components and third-party integrations to simplify AI application development — all while future-proofing decisions as the underlying technology evolves.
- **What it does:** The agent engineering platform.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 142,446 stars and 23,711 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 77. [go-kit/kit](https://github.com/go-kit/kit) — 84.2/100

- **Why this library or tool exists:** Service authors need routing, middleware, request decoding, and response ownership that remain fast without making invalid combinations easy.
- **How it works:** Go kit is a programming toolkit for building microservices (or elegant monoliths) in Go. We solve common problems in distributed systems and application architecture so you can focus on delivering business value.
- **What it does:** A standard library for microservices.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 27,431 stars and 2,444 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use pattern-indexed routes, capability-explicit middleware, typed parameter environments, request/body typestates, and exhaustive response outcomes. Compile immutable routing tables and binders ahead of time.
- **Possible Go+ standard-library pressure:** `std/http/route`, typed request/response ownership, middleware capabilities, and protocol-neutral status/decode outcomes.

### 78. [TooTallNate/Java-WebSocket](https://github.com/TooTallNate/Java-WebSocket) — 84.2/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** Parse wire messages into protocol states, drive asynchronous I/O and timers, negotiate capabilities, and route bytes through bounded buffers and owned connections.
- **What it does:** A barebones WebSocket client and server implementation written in 100% Java.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 10,812 stars and 2,586 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 79. [go-kratos/kratos](https://github.com/go-kratos/kratos) — 84.0/100

- **Why this library or tool exists:** Service authors need routing, middleware, request decoding, and response ownership that remain fast without making invalid combinations easy.
- **How it works:** Kratos is a lightweight Go framework for building cloud-native microservices. It provides small, explicit APIs for transport, middleware, registry, configuration, logging, encoding, and code generation so applications can focus on business logic.
- **What it does:** Your ultimate Go microservices framework for the cloud-native era.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 25,806 stars and 4,168 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use pattern-indexed routes, capability-explicit middleware, typed parameter environments, request/body typestates, and exhaustive response outcomes. Compile immutable routing tables and binders ahead of time.
- **Possible Go+ standard-library pressure:** `std/http/route`, typed request/response ownership, middleware capabilities, and protocol-neutral status/decode outcomes.

### 80. [jmoiron/sqlx](https://github.com/jmoiron/sqlx) — 83.8/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** sqlx is a library which provides a set of extensions on go's standard database/sql library. The sqlx versions of sql.DB, sql.TX, sql.Stmt, et al. all leave the underlying interfaces untouched, so that their interfaces are a superset on the standard ones. This makes it relatively painless to integrate existing codebases using database/sql with sqlx.
- **What it does:** general purpose extensions to golang's database/sql.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 17,702 stars and 1,118 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 81. [slackhq/nebula](https://github.com/slackhq/nebula) — 83.8/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Nebula is a scalable overlay networking tool with a focus on performance, simplicity and security. It lets you seamlessly connect computers anywhere in the world. Nebula is portable, and runs on Linux, OSX, Windows, iOS, and Android. It can be used to connect a small number of computers, but is also able to connect tens of thousands of computers.
- **What it does:** A scalable overlay networking tool with a focus on performance, simplicity and security.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 17,553 stars and 1,161 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 82. [browser-use/browser-use](https://github.com/browser-use/browser-use) — 83.6/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** Browser Use lets an AI agent use a web browser the same way you do — it opens pages, clicks buttons, types, and fills in forms. You describe the task, and it completes it. For example, you can have it:
- **What it does:** 🌐 Make websites accessible for AI agents. Automate tasks online with ease.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 106,380 stars and 11,699 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 83. [theonedev/onedev](https://github.com/theonedev/onedev) — 83.3/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Language aware symbol search and navigation in any commit. Click symbol to show occurrences in current file. Fast code search with regular expression. Try It
- **What it does:** The Unified and Autonomous Development Platform.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 15,110 stars and 964 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 84. [fastapi/typer](https://github.com/fastapi/typer) — 83.2/100

- **Why this library or tool exists:** Service authors need routing, middleware, request decoding, and response ownership that remain fast without making invalid combinations easy.
- **How it works:** Typer is a library for building CLI applications that users will love using and developers will love creating. Based on Python type hints.
- **What it does:** Typer, build great CLIs. Easy to code. Based on Python type hints.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 19,803 stars and 951 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use pattern-indexed routes, capability-explicit middleware, typed parameter environments, request/body typestates, and exhaustive response outcomes. Compile immutable routing tables and binders ahead of time.
- **Possible Go+ standard-library pressure:** `std/http/route`, typed request/response ownership, middleware capabilities, and protocol-neutral status/decode outcomes.

### 85. [soxoj/maigret](https://github.com/soxoj/maigret) — 83.1/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** Maigret collects a dossier on a person by username only, checking for accounts on a huge number of sites and gathering all the available information from web pages. No API keys required. AI profiling (demo).
- **What it does:** 🕵️‍♂️ Collect a dossier on a person by username from 3000+ sites.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 35,706 stars and 2,730 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 86. [classgraph/classgraph](https://github.com/classgraph/classgraph) — 83.0/100

- **Why this library or tool exists:** Text and language tooling needs explicit grammars, spans, typed intermediate forms, and predictable failure rather than ad-hoc branching.
- **How it works:** ClassGraph is an uber-fast parallelized classpath scanner and module scanner for Java, Scala, Kotlin and other JVM languages.
- **What it does:** An uber-fast parallelized Java classpath scanner and module scanner.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 2,999 stars and 307 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use grammar-indexed parsers, GADT ASTs, existential packaging for runtime syntax, typed passes, total visitors, and arena/streaming representations. Illegal phase combinations become unrepresentable.
- **Possible Go+ standard-library pressure:** `std/parsec` plus shared source spans, lexer tokens, grammar evidence, and deterministic diagnostic primitives.

### 87. [redis/jedis](https://github.com/redis/jedis) — 82.6/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Are you looking for a high-level library to handle object mapping? See redis-om-spring!
- **What it does:** Redis Java client.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 12,346 stars and 3,910 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 88. [auth0/java-jwt](https://github.com/auth0/java-jwt) — 82.4/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** It constructs and verifies JSON Web Tokens by encoding headers and claims, signing with a selected algorithm/key, parsing compact token segments, and applying configurable signature and claim validation.
- **What it does:** Java implementation of JSON Web Token (JWT).
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 6,231 stars and 949 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 89. [FoundationAgents/MetaGPT](https://github.com/FoundationAgents/MetaGPT) — 82.3/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** [ En | Assign different roles to GPTs to form a collaborative entity for complex tasks.
- **What it does:** 🌟 The Multi-Agent Framework: First AI Software Company, Towards Natural Language Programming.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 69,492 stars and 8,865 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 90. [locustio/locust](https://github.com/locustio/locust) — 82.3/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** Locust is an open source performance/load testing tool for HTTP and other protocols. Its developer-friendly approach lets you define your tests in regular Python code.
- **What it does:** Write scalable load tests in plain Python 🚗💨.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 28,014 stars and 3,226 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 91. [stretchr/testify](https://github.com/stretchr/testify) — 82.1/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** Go code (golang) set of packages that provide many tools for testifying that your code will behave as you intend.
- **What it does:** A toolkit with common assertions and mocks that plays nicely with the standard library.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 26,096 stars and 1,799 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 92. [GeyserMC/Geyser](https://github.com/GeyserMC/Geyser) — 82.1/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** Geyser is a bridge between Minecraft: Bedrock Edition and Minecraft: Java Edition, closing the gap from those wanting to play true cross-platform.
- **What it does:** A bridge/proxy allowing you to connect to Minecraft: Java Edition servers with Minecraft: Bedrock Edition.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 5,708 stars and 850 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 93. [commaai/openpilot](https://github.com/commaai/openpilot) — 82.0/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** openpilot is an operating system for robotics. Currently, it upgrades the driver assistance system in 300+ supported cars.
- **What it does:** openpilot is an operating system for robotics. Currently, it upgrades the driver assistance system on 300+ supported cars.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 63,196 stars and 11,185 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 94. [tsenart/vegeta](https://github.com/tsenart/vegeta) — 82.0/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** Vegeta is a versatile HTTP load testing tool built out of a need to drill HTTP services with a constant request rate. It's over 9000!
- **What it does:** HTTP load testing tool and library. It's over 9000!
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 25,127 stars and 1,417 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 95. [github/copilot-sdk](https://github.com/github/copilot-sdk) — 82.0/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Embed Copilot's agentic workflows in your application with the GitHub Copilot SDK for Python, TypeScript, Go, .NET, Java, and Rust.
- **What it does:** Multi-platform SDK for integrating GitHub Copilot Agent into apps and services.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 10,041 stars and 1,356 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 96. [scribejava/scribejava](https://github.com/scribejava/scribejava) — 82.0/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** Who said OAuth/OAuth2 was difficult? Configuring ScribeJava is so easy your grandma can do it! check it out:
- **What it does:** Simple OAuth library for Java.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 5,456 stars and 1,642 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 97. [karpathy/nanoGPT](https://github.com/karpathy/nanoGPT) — 81.9/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** Update Nov 2025 nanoGPT has a new and improved cousin called nanochat. It is very likely you meant to use/find nanochat instead. nanoGPT (this repo) is now very old and deprecated but I will leave it up for posterity.
- **What it does:** The simplest, fastest repository for training/finetuning medium-sized GPTs.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 61,457 stars and 10,574 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 98. [mitmproxy/mitmproxy](https://github.com/mitmproxy/mitmproxy) — 81.8/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** mitmproxy is an interactive, SSL/TLS-capable intercepting proxy with a console interface for HTTP/1, HTTP/2, and WebSockets.
- **What it does:** An interactive TLS-capable intercepting HTTP proxy for penetration testers and software developers.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 44,427 stars and 4,640 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 99. [openai/whisper](https://github.com/openai/whisper) — 81.6/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** It converts audio to log-Mel spectrograms and feeds them to an encoder-decoder Transformer whose token protocol represents transcription, translation, language, timestamps, and decoding controls.
- **What it does:** Robust Speech Recognition via Large-Scale Weak Supervision.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 105,480 stars and 12,812 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 100. [FoundationAgents/OpenManus](https://github.com/FoundationAgents/OpenManus) — 81.6/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** Manus is incredible, but OpenManus can achieve any idea without an Invite Code 🛫!
- **What it does:** No fortress, purely open ground. OpenManus is Coming.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 57,568 stars and 10,020 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 101. [karpathy/nanochat](https://github.com/karpathy/nanochat) — 81.6/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** For questions about the repo, I recommend either using DeepWiki from Devin/Cognition to ask questions about the repo, or use the Discussions tab, or come by the #nanochat channel on Discord.
- **What it does:** The best ChatGPT that $100 can buy.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 56,587 stars and 7,828 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 102. [smicallef/spiderfoot](https://github.com/smicallef/spiderfoot) — 81.2/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** SpiderFoot is an open source intelligence (OSINT) automation tool. It integrates with just about every data source available and utilises a range of methods for data analysis, making that data easy to navigate.
- **What it does:** SpiderFoot automates OSINT for threat intelligence and mapping your attack surface.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 19,834 stars and 3,224 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 103. [3b1b/manim](https://github.com/3b1b/manim) — 81.1/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** Manim is an engine for precise programmatic animations, designed for creating explanatory math videos.
- **What it does:** Animation engine for explanatory math videos.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 88,807 stars and 7,406 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 104. [floci-io/floci](https://github.com/floci-io/floci) — 80.7/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** Light, fluffy, and always free No account. No auth token. No feature gates. Just docker compose up .
- **What it does:** Light, fluffy, and always free - The AWS Local Emulator alternative.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 17,006 stars and 1,697 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 105. [ffuf/ffuf](https://github.com/ffuf/ffuf) — 80.6/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** It substitutes one or more fuzz markers into HTTP request components, drives requests concurrently from wordlists, filters or matches responses by status, size, words, lines, time, or regex, and reports discovered endpoints.
- **What it does:** Fast web fuzzer written in Go.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 16,424 stars and 1,580 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 106. [modelcontextprotocol/java-sdk](https://github.com/modelcontextprotocol/java-sdk) — 80.6/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** A set of projects that provide Java SDK integration for the Model Context Protocol. This SDK enables Java applications to interact with AI models and tools through a standardized interface, supporting both synchronous and asynchronous communication patterns.
- **What it does:** The official Java SDK for Model Context Protocol servers and clients. Maintained in collaboration with Spring AI.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 3,584 stars and 981 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 107. [projectdiscovery/nuclei](https://github.com/projectdiscovery/nuclei) — 80.5/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** Nuclei is a modern, high-performance vulnerability scanner that leverages simple YAML-based templates. It empowers you to design custom vulnerability detection scenarios that mimic real-world conditions, leading to zero false positives.
- **What it does:** Nuclei is a fast, customizable vulnerability scanner powered by the global security community and built on a simple YAML-based DSL, enabling collaboration to tackle trending vulnerabilities on the internet. It helps you find vulnerabilities in your applications, APIs, networks, DNS, and cloud configurations.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 29,952 stars and 3,630 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 108. [mailhog/MailHog](https://github.com/mailhog/MailHog) — 80.5/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** Generate or execute isolated scenarios, capture structured observations, compare them with expectations, and report reproducible diagnostics.
- **What it does:** Web and API based SMTP testing.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 16,094 stars and 1,177 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 109. [mockito/mockito](https://github.com/mockito/mockito) — 80.4/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** It creates runtime test doubles for classes and interfaces, records method invocations, routes calls through configured stubs or answers, and later verifies interactions and argument constraints.
- **What it does:** Most popular Mocking framework for unit tests written in Java.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 15,443 stars and 2,662 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 110. [kevinsawicki/http-request](https://github.com/kevinsawicki/http-request) — 80.4/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** A simple convenience library for using a HttpURLConnection to make requests and access the response.
- **What it does:** Java HTTP Request Library.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 3,393 stars and 825 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 111. [HKUDS/LightRAG](https://github.com/HKUDS/LightRAG) — 80.3/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** Figure 1: LightRAG Indexing Flowchart - Img Caption : Source Figure 2: LightRAG Retrieval and Querying Flowchart - Img Caption : Source
- **What it does:** [EMNLP2025] "LightRAG: Simple and Fast Retrieval-Augmented Generation".
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 38,040 stars and 5,353 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 112. [gitleaks/gitleaks](https://github.com/gitleaks/gitleaks) — 80.3/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** Gitleaks is a tool for detecting secrets like passwords, API keys, and tokens in git repos, files, and whatever else you wanna throw at it via stdin. If you wanna learn more about how the detection engine works check out this blog: Regex is (almost) all you need.
- **What it does:** Find secrets with Gitleaks 🔑.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 28,279 stars and 2,157 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 113. [stanfordnlp/dspy](https://github.com/stanfordnlp/dspy) — 80.2/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** DSPy is the framework for programming—rather than prompting—language models. It allows you to iterate fast on building modular AI systems and offers algorithms for optimizing their prompts and weights, whether you're building simple classifiers, sophisticated RAG pipelines, or Agent loops.
- **What it does:** DSPy: The framework for programming—not prompting—language models.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 36,329 stars and 3,125 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 114. [redis/lettuce](https://github.com/redis/lettuce) — 80.2/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Lettuce is a scalable thread-safe Redis client for synchronous, asynchronous and reactive usage. Multiple threads may share one connection if they avoid blocking and transactional operations such as BLPOP and MULTI/EXEC. Lettuce is built with netty. Supports advanced Redis features such as Sentinel, Cluster, Pipelining, Auto-Reconnect and Redis data models.
- **What it does:** Advanced Java Redis client for thread-safe sync, async, and reactive usage. Supports Cluster, Sentinel, Pipelining, and codecs.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 5,769 stars and 1,094 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 115. [microsoft/graphrag](https://github.com/microsoft/graphrag) — 80.0/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** The GraphRAG project is a data pipeline and transformation suite that is designed to extract meaningful, structured data from unstructured text using the power of LLMs.
- **What it does:** A modular graph-based Retrieval-Augmented Generation (RAG) system.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 34,775 stars and 3,663 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 116. [sirupsen/logrus](https://github.com/sirupsen/logrus) — 80.0/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** Logrus is a structured logger for Go (golang), completely API compatible with the standard library logger.
- **What it does:** Structured, pluggable logging for Go.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 25,750 stars and 2,286 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 117. [tinygrad/tinygrad](https://github.com/tinygrad/tinygrad) — 79.9/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** tinygrad: For something between PyTorch and karpathy/micrograd. Maintained by tiny corp.
- **What it does:** You like pytorch? You like micrograd? You love tinygrad! ❤️.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 33,326 stars and 4,229 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 118. [uber-go/zap](https://github.com/uber-go/zap) — 79.9/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** In contexts where performance is nice, but not critical, use the SugaredLogger. It's 4-10x faster than other structured logging packages and includes both structured and printf-style APIs.
- **What it does:** Blazing fast, structured, leveled logging in Go.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 24,581 stars and 1,536 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 119. [oshi/oshi](https://github.com/oshi/oshi) — 79.9/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** Supported Platforms --------------------------- - Windows - macOS - Linux (Android) - UNIX (AIX, DragonFly BSD, FreeBSD, NetBSD, OpenBSD, Solaris (illumos))
- **What it does:** Native Operating System and Hardware Information.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 5,250 stars and 921 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 120. [RVC-Boss/GPT-SoVITS](https://github.com/RVC-Boss/GPT-SoVITS) — 79.8/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** 1. Zero-shot TTS: Input a 5-second vocal sample and experience instant text-to-speech conversion.
- **What it does:** 1 min voice data can also be used to train a good TTS model! (few shot voice cloning).
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 60,053 stars and 6,542 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 121. [Delgan/loguru](https://github.com/Delgan/loguru) — 79.8/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** Did you ever feel lazy about configuring a logger and used print() instead?... I did, yet logging is fundamental to every application and eases the process of debugging. Using Loguru you have no excuse not to use logging from the start, this is as simple as from loguru import logger.
- **What it does:** Python logging made (stupidly) simple.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 24,039 stars and 803 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 122. [henrygd/beszel](https://github.com/henrygd/beszel) — 79.8/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** Beszel is a lightweight server monitoring platform that includes Docker statistics, historical data, and alert functions.
- **What it does:** Lightweight server monitoring with historical data, docker stats, and alerts.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 23,703 stars and 907 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 123. [jesseduffield/lazygit](https://github.com/jesseduffield/lazygit) — 79.7/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** Special thanks to: Warp, the intelligent terminal Available for macOS and Linux Visit warp.dev to learn more. Tuple, the premier screen sharing app for developers on macOS and Windows. I (Jesse) co-founded Subble to save your company time and money by finding unused and over-provisioned SaaS licences. Check it out!
- **What it does:** simple terminal UI for git commands.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 80,670 stars and 2,935 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 124. [Kong/unirest-java](https://github.com/Kong/unirest-java) — 79.7/100

- **Why this library or tool exists:** Protocol software needs exhaustive connection states, owned streams, bounded retries, and cancellation-safe I/O.
- **How it works:** Unirest 4 is build on modern Java standards, and as such requires at least Java 11.
- **What it does:** Unirest in Java: Simplified, lightweight HTTP client library.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 2,704 stars and 590 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use protocol GADTs, connection typestates, linear body/stream capabilities, replayability indices, and exhaustive timeout/close outcomes. Pool owned buffers and specialize parsers to reduce copying.
- **Possible Go+ standard-library pressure:** Owned streams/bodies, replayability, deadlines, backoff, framed codecs, and protocol-state helpers; concrete protocols stay in focused packages.

### 125. [wagoodman/dive](https://github.com/wagoodman/dive) — 79.5/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** A tool for exploring a Docker image, layer contents, and discovering ways to shrink the size of your Docker/OCI image.
- **What it does:** A tool for exploring each layer in a docker image.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 54,351 stars and 1,991 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 126. [ScrapeGraphAI/Scrapegraph-ai](https://github.com/ScrapeGraphAI/Scrapegraph-ai) — 79.4/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** English | 中文 | 日本語 | 한국어 | Русский | Türkçe | Deutsch | Español | français | Português | Italiano
- **What it does:** Python scraper based on AI.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 28,595 stars and 2,789 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 127. [TecharoHQ/anubis](https://github.com/TecharoHQ/anubis) — 79.3/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** Stream artifacts or events through detectors, normalize findings, correlate evidence, and emit structured reports or enforcement decisions.
- **What it does:** Weighs the soul of incoming HTTP requests to stop AI crawlers.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 20,873 stars and 656 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 128. [microsoft/VibeVoice](https://github.com/microsoft/VibeVoice) — 79.2/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** 2026-03-06: 🚀 VibeVoice ASR is now part of a Transformers release ! You can now use our speech recognition model directly through the Hugging Face Transformers library for seamless integration into your projects.
- **What it does:** Open-Source Frontier Voice AI.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 50,448 stars and 5,639 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 129. [oraios/serena](https://github.com/oraios/serena) — 79.2/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** Serena provides essential semantic code retrieval, editing, refactoring and debugging tools that are akin to an IDE's capabilities, operating at the symbol level and exploiting relational structure. It integrates with any client/LLM via the model context protocol (MCP).
- **What it does:** A powerful MCP toolkit for coding, providing semantic retrieval and editing capabilities - the IDE for your agent.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 26,801 stars and 1,773 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 130. [langchain-ai/deepagents](https://github.com/langchain-ai/deepagents) — 79.2/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** Deep Agents is an open source agent harness — an opinionated agent that runs out of the box. Extend, override, or replace any piece.
- **What it does:** The batteries-included agent harness.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 26,734 stars and 3,743 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 131. [SWE-agent/SWE-agent](https://github.com/SWE-agent/SWE-agent) — 79.2/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** SWE-agent enables your language model of choice (e.g. GPT-4o or Claude Sonnet 4) to autonomously use tools to fix issues in real GitHub repositories, find cybersecurity vulnerabilities, or perform any custom task.
- **What it does:** SWE-agent takes a GitHub issue and tries to automatically fix it, using your LM of choice. It can also be employed for offensive cybersecurity or competitive coding challenges. [NeurIPS 2024].
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 19,899 stars and 2,174 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 132. [roboflow/supervision](https://github.com/roboflow/supervision) — 79.1/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** It wraps model detections in common typed containers, provides geometry and tracking operations, and composes annotators, zone logic, counters, and video-processing callbacks into vision pipelines.
- **What it does:** We write your reusable computer vision tools. 💜.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 48,320 stars and 4,439 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 133. [mudler/LocalAI](https://github.com/mudler/LocalAI) — 79.0/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** LocalAI is the open-source AI engine. Run any model - LLMs, vision, voice, image, video - on any hardware. No GPU required.
- **What it does:** LocalAI is the open-source AI engine. Run any model - LLMs, vision, voice, image, video - on any hardware. No GPU required.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 47,778 stars and 4,277 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 134. [knowm/XChange](https://github.com/knowm/XChange) — 79.0/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** XChange is a Java library providing a simple and consistent API for interacting with 60+ Bitcoin and other cryptocurrency exchanges, providing a consistent interface for trading and accessing market data.
- **What it does:** XChange is a Java library providing a streamlined API for interacting with 60+ Bitcoin and Altcoin exchanges providing a consistent interface for trading and accessing market data.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 4,075 stars and 2,001 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 135. [hiroi-sora/Umi-OCR](https://github.com/hiroi-sora/Umi-OCR) — 78.9/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** - 免费：本项目所有代码开源，完全免费。 - 方便：解压即用，离线运行，无需网络。 - 高效：自带高效率的离线OCR引擎，内置多种语言识别库。 - 灵活：支持命令行、HTTP接口等外部调用方式。 - 功能：截图OCR / 批量OCR / PDF识别 / 二维码 / 公式识别
- **What it does:** OCR software, free and offline. 开源、免费的离线OCR软件。支持截屏/批量导入图片，PDF文档识别，排除水印/页眉页脚，扫描/生成二维码。内置多国语言库。.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 46,209 stars and 4,541 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 136. [9001/copyparty](https://github.com/9001/copyparty) — 78.9/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** turn almost any device into a file server with resumable uploads/downloads using any web browser
- **What it does:** Portable file server with accelerated resumable uploads, dedup, WebDAV, SFTP, FTP, TFTP, zeroconf, media indexer, thumbnails++ all in one file.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 45,831 stars and 1,871 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 137. [bcicen/ctop](https://github.com/bcicen/ctop) — 78.8/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** ctop provides a concise and condensed overview of real-time metrics for multiple containers:
- **What it does:** Top-like interface for container metrics.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 17,799 stars and 589 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 138. [influxdata/telegraf](https://github.com/influxdata/telegraf) — 78.8/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** Telegraf is an agent for collecting, processing, aggregating, and writing metrics, logs, and other arbitrary data.
- **What it does:** Agent for collecting, processing, aggregating, and writing metrics, logs, and other arbitrary data.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 17,721 stars and 5,831 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 139. [Textualize/rich](https://github.com/Textualize/rich) — 78.6/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** English readme • 简体中文 readme • 正體中文 readme • Lengua española readme • Deutsche readme • Läs på svenska • 日本語 readme • 한국어 readme • Français readme • Schwizerdütsch readme • हिन्दी readme • Português brasileiro readme • Italian readme • Русский readme • Indonesian readme • فارسی readme • Türkçe readme • Polskie readme
- **What it does:** Rich is a Python library for rich text and beautiful formatting in the terminal.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 56,924 stars and 2,261 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 140. [karatelabs/karate](https://github.com/karatelabs/karate) — 78.6/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** The open-source tool that combines API testing, mocks, performance testing, and UI automation into a single, unified framework.
- **What it does:** Test Automation Made Simple.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 8,902 stars and 2,042 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 141. [testcontainers/testcontainers-java](https://github.com/testcontainers/testcontainers-java) — 78.5/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** Hashicorp Vault module is (c) 2017 - 2021 Capital One Services, LLC and other authors.
- **What it does:** Testcontainers is a Java library that supports JUnit tests, providing lightweight, throwaway instances of common databases, Selenium web browsers, or anything else that can run in a Docker container.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 8,692 stars and 1,887 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 142. [ManimCommunity/manim](https://github.com/ManimCommunity/manim) — 78.4/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** Manim is an animation engine for explanatory math videos. It's used to create precise animations programmatically, as demonstrated in the videos of 3Blue1Brown.
- **What it does:** A community-maintained Python framework for creating mathematical animations.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 39,660 stars and 2,972 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 143. [jesseduffield/lazydocker](https://github.com/jesseduffield/lazydocker) — 78.3/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** Special thanks to: Warp, the intelligent terminal Available for MacOS and Linux Visit warp.dev to learn more. Tuple, the premier screen sharing app for developers on macOS and Windows. Click here to learn more
- **What it does:** The lazier way to manage everything docker.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 52,137 stars and 1,660 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 144. [facebook/prophet](https://github.com/facebook/prophet) — 78.3/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** 2023 Update: We discuss our plans for the future of Prophet in this blog post: facebook/prophet in 2023 and beyond
- **What it does:** Tool for producing high quality forecasts for time series data that has multiple seasonality with linear or non-linear growth.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 20,325 stars and 4,634 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 145. [myshell-ai/OpenVoice](https://github.com/myshell-ai/OpenVoice) — 78.2/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** As we detailed in our paper and website, the advantages of OpenVoice are three-fold:
- **What it does:** Instant voice cloning by MIT and MyShell. Audio foundation model.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 37,009 stars and 4,138 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 146. [tlaplus/tlaplus](https://github.com/tlaplus/tlaplus) — 78.0/100

- **Why this library or tool exists:** Stateful systems need clear transaction, ownership, query, replication, and consistency contracts around persistence.
- **How it works:** This repository hosts the core TLA⁺ command line interface (CLI) Tools and the Toolbox integrated development environment (IDE). Its development is managed by the TLA⁺ Foundation. See http://tlapl.us for more information about TLA⁺ itself. For the TLA⁺ proof manager, see http://proofs.tlapl.us.
- **What it does:** TLC is a model checker for specifications written in TLA+. The TLA+Toolbox is an IDE for TLA+.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 2,972 stars and 261 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use transaction/lease typestates, schema- and query-indexed values, owned buffers, exhaustive consistency outcomes, and proof-carrying migrations. Specialize codecs and query plans at generation time.
- **Possible Go+ standard-library pressure:** Protocol-neutral transaction/lease outcomes, owned byte views, and migration witnesses; storage engines and query dialects should remain outside `std`.

### 147. [cli/cli](https://github.com/cli/cli) — 77.9/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** gh is GitHub on the command line. It brings pull requests, issues, and other GitHub concepts to the terminal next to where you are already working with git and your code.
- **What it does:** GitHub’s official command line tool.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 45,382 stars and 8,749 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 148. [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) — 77.8/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** The fun, functional and stateful way to build terminal apps. A Go framework based on [The Elm Architecture][elm]. Bubble Tea is well-suited for simple and complex terminal applications, either inline, full-window, or a mix of both.
- **What it does:** A powerful little TUI framework 🏗.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 43,899 stars and 1,272 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 149. [iawia002/lux](https://github.com/iawia002/lux) — 77.7/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** Site: YouTube youtube.com Title: Rick Astley - Never Gonna Give You Up (Video) Type: video Stream: [248] ------------------- Quality: 1080p video/webm; codecs="vp9" Size: 63.93 MiB (67038963 Bytes)
- **What it does:** 👾 Fast and simple video download library and CLI tool written in Go.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 31,561 stars and 3,315 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 150. [bcgit/bc-java](https://github.com/bcgit/bc-java) — 77.7/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** The Bouncy Castle Crypto package is a Java implementation of cryptographic algorithms, it was developed by the Legion of the Bouncy Castle, a registered Australian Charity, with a little help! The Legion, and the latest goings on with this package, can be found at https://www.bouncycastle.org.
- **What it does:** Bouncy Castle Java Distribution (Mirror).
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 2,669 stars and 1,264 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 151. [larksuite/cli](https://github.com/larksuite/cli) — 77.4/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** The official Lark/Feishu CLI tool, maintained by the larksuite team — built for humans and AI Agents. Covers core business domains including Messenger, Docs, Base, Sheets, Slides, Calendar, Mail, Tasks, Meetings, Markdown, and more, with 200+ commands and 26 AI Agent Skills.
- **What it does:** The official Lark/Feishu CLI tool, maintained by the larksuite team — built for humans and AI Agents. Covers core business domains including Messenger, Docs, Base, Sheets, Calendar, Mail, Tasks, Meetings, and more, with 200+ commands and 20+ AI Agent Skills.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 15,757 stars and 1,152 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 152. [plandex-ai/plandex](https://github.com/plandex-ai/plandex) — 77.4/100

- **Why this library or tool exists:** AI systems need explicit model, tool, memory, streaming, and execution states instead of loosely coordinated dictionaries and callbacks.
- **How it works:** 💻  Plandex is a terminal-based AI development tool that can plan and execute large coding tasks that span many steps and touch dozens of files. It can handle up to 2M tokens of context directly (~100k per file), and can index directories with 20M tokens or more using tree-sitter project maps.
- **What it does:** Open source AI coding agent. Designed for large projects and real world tasks.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 15,543 stars and 1,166 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed tool schemas, provider/effect capabilities, agent-state sums, model/tensor shape indices, owned streams, and deterministic event logs. A Go rewrite can replace interpreter overhead in orchestration hot paths.
- **Possible Go+ standard-library pressure:** Typed tool-call/result envelopes, cancellable event streams, tensor-shape witnesses, and explicit provider capability interfaces only where broadly reusable.

### 153. [deezer/spleeter](https://github.com/deezer/spleeter) — 77.3/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** Spleeter is Deezer source separation library with pretrained models written in Python and uses Tensorflow. It makes it easy to train source separation model (assuming you have a dataset of isolated sources), and provides already trained state of the art model for performing various flavour of separation :
- **What it does:** Deezer source separation library including pretrained models.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 28,334 stars and 3,059 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 154. [ArchiveBox/ArchiveBox](https://github.com/ArchiveBox/ArchiveBox) — 77.3/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** ArchiveBox is a self-hosted app that lets you preserve content from websites in a variety of formats.
- **What it does:** 🗃 Open source self-hosted web archiving. Takes URLs/browser history/bookmarks/Pocket/Pinboard/etc., saves HTML, JS, PDFs, media, and more...
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 27,995 stars and 1,554 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 155. [Textualize/textual](https://github.com/Textualize/textual) — 77.2/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** Build cross-platform user interfaces with a simple Python API. Run your apps in the terminal or a web browser.
- **What it does:** The lean application framework for Python. Build sophisticated user interfaces with a simple Python API. Run your apps in the terminal and a web browser.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 36,711 stars and 1,278 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 156. [danielgatis/rembg](https://github.com/danielgatis/rembg) — 76.8/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** It decodes images, runs a selected segmentation model to estimate foreground alpha or masks, post-processes edges, and emits cutouts through a Python API, CLI, batch mode, stream mode, or HTTP service.
- **What it does:** Rembg is a tool to remove images background.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 23,973 stars and 2,361 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 157. [frohoff/ysoserial](https://github.com/frohoff/ysoserial) — 76.6/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** A proof-of-concept tool for generating payloads that exploit unsafe Java object deserialization.
- **What it does:** A proof-of-concept tool for generating payloads that exploit unsafe Java object deserialization.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 8,990 stars and 1,860 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 158. [charmbracelet/vhs](https://github.com/charmbracelet/vhs) — 76.3/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** Tape files consist of a series of commands. The commands are instructions for VHS to perform on its virtual terminal. For a list of all possible commands see the command reference.
- **What it does:** Your CLI home video recorder 📼.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 20,439 stars and 447 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 159. [charmbracelet/glow](https://github.com/charmbracelet/glow) — 76.1/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** Glow is a terminal based markdown reader designed from the ground up to bring out the beauty—and power—of the CLI.
- **What it does:** Render markdown on the CLI, with pizzazz! 💅🏻.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 26,474 stars and 730 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 160. [asdf-vm/asdf](https://github.com/asdf-vm/asdf) — 76.0/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** Manage multiple runtime versions with a single CLI tool, extendable via plugins - docs at asdf-vm.com
- **What it does:** Extendable version manager with support for Ruby, Node.js, Elixir, Erlang & more.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 25,483 stars and 928 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 161. [go-delve/delve](https://github.com/go-delve/delve) — 75.9/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** The GitHub issue tracker is for bugs only. Please use the developer mailing list for any feature proposals and discussions.
- **What it does:** Delve is a debugger for the Go programming language.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 24,849 stars and 2,213 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 162. [urfave/cli](https://github.com/urfave/cli) — 75.8/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** urfave/cli is a declarative, simple, fast, and fun package for building command line tools in Go featuring:
- **What it does:** A declarative, simple, fast, and fun package for building command line tools in Go.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 24,174 stars and 1,802 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 163. [charmbracelet/gum](https://github.com/charmbracelet/gum) — 75.8/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** A tool for glamorous shell scripts. Leverage the power of Bubbles and Lip Gloss in your scripts and aliases without writing any Go code!
- **What it does:** A tool for glamorous shell scripts 🎀.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 24,083 stars and 530 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 164. [projectdiscovery/katana](https://github.com/projectdiscovery/katana) — 75.7/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** - Fast And fully configurable web crawling - Standard and Headless mode - JavaScript parsing / crawling - Customizable automatic form filling - Scope control - Preconfigured field / Regex - Knowledge base - ML page-type / form classification (auto-downloaded model) - Customizable output - Preconfigured fields - INPUT - STDIN, URL and LIST - OUTPUT -...
- **What it does:** A next-generation crawling and spidering framework.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 17,213 stars and 1,159 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 165. [antonmedv/fx](https://github.com/antonmedv/fx) — 75.3/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** - walk – terminal file manager - howto – terminal command LLM helper - countdown – terminal countdown timer
- **What it does:** Terminal JSON viewer & processor.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 20,544 stars and 483 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 166. [yorukot/superfile](https://github.com/yorukot/superfile) — 75.0/100

- **Why this library or tool exists:** Developer tools need responsive incremental state, cancellation, deterministic rendering, and recoverable external-process boundaries.
- **How it works:** A terminal event loop maintains panels and selection state, performs filesystem operations through commands, and renders previews, metadata, and progress in a keyboard-driven interface.
- **What it does:** Pretty fancy and modern terminal file manager.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 18,959 stars and 582 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use an exhaustive update algebra, immutable commands, typed key/action maps, cancellation capabilities, and owned render buffers. Incremental diffing and pooled cells can reduce redraw and allocation costs.
- **Possible Go+ standard-library pressure:** A small event/update algebra, terminal cell buffers, cancellation-aware command execution, and input/key decoding if reused across several tools.

### 167. [cucumber/cucumber-jvm](https://github.com/cucumber/cucumber-jvm) — 74.9/100

- **Why this library or tool exists:** Teams need repeatable evidence that behavior holds across examples, properties, integrations, and failure paths.
- **How it works:** Cucumber is a tool for running automated tests written in plain language. Because they're written in plain language, they can be read by anyone on your team. Because they can be read by anyone, you can use them to help improve communication, collaboration and trust on your team.
- **What it does:** Cucumber for the JVM.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 2,827 stars and 2,013 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use law-bearing interfaces, typed generators, exhaustive outcomes, and resource-scoped fixtures. Shared immutable scenario plans can reduce reflection and setup churn.
- **Possible Go+ standard-library pressure:** Potential law/property-test primitives and structured test outcomes, but only after multiple independent consumers prove a stable API.

### 168. [mbechler/marshalsec](https://github.com/mbechler/marshalsec) — 73.7/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** If you came here for Log4Shell/CVE-2021-44228, you may want to read about the exploitation vectors and affected Java runtime versions:
- **What it does:** mbechler/marshalsec is an established open-source project.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 3,695 stars and 680 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 169. [qos-ch/slf4j](https://github.com/qos-ch/slf4j) — 72.5/100

- **Why this library or tool exists:** Operators need high-throughput inspection with explicit findings, provenance, severity, and policy outcomes.
- **How it works:** The Simple Logging Facade for Java (SLF4J) serves as a simple facade or abstraction for various logging frameworks (e.g. java.util.logging, logback, reload4j, log4j 2.x, logevents, penna, rainbowgum, tinylog) allowing the end user to plug in the desired logging framework at deployment time.
- **What it does:** Simple Logging Facade for Java.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 2,518 stars and 1,024 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use typed evidence, severity/policy sums, streaming detectors, refinement-checked offsets, and deterministic rule plans. Zero-copy scanning and compiled matchers can improve throughput.
- **Possible Go+ standard-library pressure:** Structured evidence spans, streaming matcher interfaces, redaction-safe diagnostics, and immutable rule plans if shared by multiple tools.

### 170. [nvbn/thefuck](https://github.com/nvbn/thefuck) — 72.4/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** The Fuck is a magnificent app, inspired by a @liamosaur tweet, that corrects errors in previous console commands.
- **What it does:** Magnificent app which corrects your previous console command.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 97,551 stars and 3,953 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 171. [coobird/thumbnailator](https://github.com/coobird/thumbnailator) — 72.0/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** October 1, 2025: Thumbnailator 0.4.21 has been released! See Changes for details.
- **What it does:** Thumbnailator - a thumbnail generation library for Java.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 5,425 stars and 794 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 172. [abi/screenshot-to-code](https://github.com/abi/screenshot-to-code) — 71.4/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** Convert screenshots, mockups, Figma designs, and screen recordings into clean, functional code using AI. The easiest way to try this is using the official, hosted product at screenshottocode.com →
- **What it does:** Drop in a screenshot and convert it to clean code (HTML/Tailwind/React/Vue).
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 73,428 stars and 9,031 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 173. [JabRef/jabref](https://github.com/JabRef/jabref) — 71.3/100

- **Why this library or tool exists:** Media pipelines need typed formats, dimensions, timing, ownership, and streaming stages without unnecessary copying.
- **How it works:** JabRef is an open-source, cross-platform citation and reference management tool.
- **What it does:** Desktop app for managing BibTeX and BibLaTeX (.bib) libraries.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 4,437 stars and 3,469 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Use format-, dimension-, and clock-indexed frames, ownership-aware buffers, exhaustive codec states, and staged kernels. Specialization and buffer reuse can remove Python/JVM dispatch and allocation overhead.
- **Possible Go+ standard-library pressure:** Owned buffer/view primitives, bounded dimensions, clocks/durations, and streaming codec outcomes; specific codecs remain external.

### 174. [usememos/memos](https://github.com/usememos/memos) — 70.9/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** Memos is an open-source, self-hosted note-taking app built for quick capture. It is Markdown-native, lightweight, and keeps your data under your control.
- **What it does:** Open-source, self-hosted note-taking tool built for quick capture. Markdown-native, lightweight, and fully yours.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 61,739 stars and 4,585 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 175. [MemPalace/mempalace](https://github.com/MemPalace/mempalace) — 70.7/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** Local-first AI memory. Verbatim storage, pluggable backend, 96.6% R@5 raw on LongMemEval — zero API calls.
- **What it does:** The best-benchmarked open-source AI memory system. And it's free.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 57,660 stars and 7,429 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 176. [go-gitea/gitea](https://github.com/go-gitea/gitea) — 70.6/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** The goal of Gitea is to make the easiest, fastest, and most painless way of setting up a self-hosted all-in-one software development service, including Git hosting, code management, code review, issue tracking, project kanban, wiki, team collaboration, package registry and CI/CD which can reuse GitHub Actions.
- **What it does:** Git with a cup of tea! Painless self-hosted all-in-one software development service, including Git hosting, code review, team collaboration, package registry and CI/CD.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 56,996 stars and 6,935 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 177. [ageitgey/face_recognition](https://github.com/ageitgey/face_recognition) — 70.6/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** You can also read a translated version of this file in Chinese 简体中文版 or in Korean 한국어 or in Japanese 日本語.
- **What it does:** The world's simplest facial recognition api for Python and the command line.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 56,611 stars and 13,699 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 178. [coreybutler/nvm-windows](https://github.com/coreybutler/nvm-windows) — 70.0/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** The original nvm is a completely separate project for Mac/Linux only. This project uses an entirely different philosophy and is not just a clone of nvm. Details are listed in Why another version manager? and what&#39;s the big difference?.
- **What it does:** A node.js version management utility for Windows. Ironically written in Go.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 47,143 stars and 3,857 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 179. [vnpy/vnpy](https://github.com/vnpy/vnpy) — 69.8/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** VeighNa是一套基于Python的开源量化交易系统开发框架，在开源社区持续不断的贡献下一步步成长为多功能量化交易平台，自发布以来已经积累了众多来自金融机构或相关领域的用户，包括私募基金、证券公司、期货公司等。
- **What it does:** 基于Python的开源量化交易平台开发框架.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 43,805 stars and 12,232 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 180. [danielmiessler/Fabric](https://github.com/danielmiessler/Fabric) — 69.7/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** Special thanks to: Warp, built for coding with multiple AI agents Available for macOS, Linux and Windows
- **What it does:** Fabric is an open-source framework for augmenting humans using AI. It provides a modular system for solving specific problems using a crowdsourced set of AI prompts that can be used anywhere.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 43,182 stars and 4,212 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 181. [mingrammer/diagrams](https://github.com/mingrammer/diagrams) — 69.7/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** Diagrams lets you draw the cloud system architecture in Python code. It was born for prototyping a new system architecture design without any design tools. You can also describe or visualize the existing system architecture as well. Diagrams currently supports main major providers including: AWS, Azure, GCP, Kubernetes, Alibaba Cloud, Oracle Cloud etc......
- **What it does:** :art: Diagram as Code for prototyping cloud system architectures.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 42,454 stars and 2,730 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 182. [wailsapp/wails](https://github.com/wailsapp/wails) — 69.1/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** English · 简体中文 · 日本語 · 한국어 · Español · Português · Русский · Francais · Uzbek · Deutsch · Türkçe · Bahasa Indonesia
- **What it does:** Create beautiful applications using Go.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 35,315 stars and 1,767 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 183. [j-easy/easy-rules](https://github.com/j-easy/easy-rules) — 68.9/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** As of December 2020, Easy Rules is in maintenance mode. This means only bug fixes will be addressed from now on. Version 4.1.x is the only supported version. Please consider upgrading to this version at your earliest convenience.
- **What it does:** The simple, stupid rules engine for Java.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 5,256 stars and 1,105 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 184. [github/github-mcp-server](https://github.com/github/github-mcp-server) — 68.7/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** The GitHub MCP Server connects AI tools directly to GitHub's platform. This gives AI agents, assistants, and chatbots the ability to read repositories and code files, manage issues and PRs, analyze code, and automate workflows. All through natural language interactions.
- **What it does:** GitHub's official MCP Server.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 31,661 stars and 4,636 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 185. [abiosoft/colima](https://github.com/abiosoft/colima) — 68.5/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** - Simple CLI interface with sensible defaults - Automatic Port Forwarding - Volume mounts - Multiple instances - Support for multiple container runtimes - Docker (with optional Kubernetes) - Containerd (with optional Kubernetes) - Incus (containers and virtual machines) - GPU accelerated containers for AI workloads
- **What it does:** Container runtimes on macOS (and Linux) with minimal setup.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 30,062 stars and 593 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 186. [squidfunk/mkdocs-material](https://github.com/squidfunk/mkdocs-material) — 68.2/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** Compose domain services behind a CLI, server, or UI; persist application state; and integrate external systems through adapters.
- **What it does:** Documentation that simply works.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 27,138 stars and 4,116 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 187. [resemble-ai/chatterbox](https://github.com/resemble-ai/chatterbox) — 68.0/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** Chatterbox is a family of state-of-the-art, open-source text-to-speech models by Resemble AI.
- **What it does:** SoTA open-source TTS.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 25,656 stars and 3,416 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 188. [jenkinsci/jenkins](https://github.com/jenkinsci/jenkins) — 68.0/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** In a nutshell, Jenkins is the leading open-source automation server. Built with Java, it provides over 2,000 plugins to support automating virtually anything, so that humans can spend their time doing things machines cannot.
- **What it does:** Jenkins automation server.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 25,609 stars and 9,624 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 189. [serengil/deepface](https://github.com/serengil/deepface) — 67.7/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** DeepFace is a lightweight face recognition and facial attribute analysis (age, gender, emotion and race) framework for python. It is a hybrid face recognition framework wrapping state-of-the-art models: VGG-Face, FaceNet, OpenFace, DeepFace, DeepID, ArcFace, Dlib, SFace, GhostFaceNet, BuffaloL.
- **What it does:** A Lightweight Face Recognition and Facial Attribute Analysis (Age, Gender, Emotion and Race) Library for Python.
- **Language:** Python (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 23,144 stars and 3,145 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 190. [mislav/hub](https://github.com/mislav/hub) — 67.7/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** hub is a command line tool that wraps git in order to extend it with extra features and commands that make working with GitHub easier.
- **What it does:** A command-line tool that makes git easier to use with GitHub.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 22,957 stars and 2,222 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 191. [matryer/xbar](https://github.com/matryer/xbar) — 66.9/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** xbar (the BitBar reboot) lets you put the output from any script/program in your macOS menu bar.
- **What it does:** Put the output from any script or program into your macOS Menu Bar (the BitBar reboot).
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 18,054 stars and 653 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 192. [justauth/JustAuth](https://github.com/justauth/JustAuth) — 66.8/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** JustAuth 集成了诸如：Github、Gitee、支付宝、新浪微博、微信、Google、Facebook、Twitter、StackOverflow等国内外数十家第三方平台。更多请参考 已集成的平台
- **What it does:** 🏆Gitee 最有价值开源项目 🚀:100: 小而全而美的第三方登录开源组件。目前已支持Github、Gitee、微博、钉钉、百度、Coding、腾讯云开发者平台、OSChina、支付宝、QQ、微信、淘宝、Google、Facebook、抖音、领英、小米、微软、今日头条、Teambition、StackOverflow、Pinterest、人人、华为、企业微信、酷家乐、Gitlab、美团、饿了么、推特、飞书、京东、阿里云、喜马拉雅、Amazon、Slack和 Line 等第三方平台的授权登录。 Login, so easy!
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 17,476 stars and 2,855 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 193. [jeessy2/ddns-go](https://github.com/jeessy2/ddns-go) — 66.7/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** - DDNS-GO - 特性 - 系统中使用 - Docker中使用 - 使用IPv6 - Webhook - Callback - 界面 - 开发\&自行编译
- **What it does:** Simple and easy to use DDNS. Support Aliyun, Tencent Cloud, Dnspod, Cloudflare, Callback, Huawei Cloud, Baidu Cloud, Porkbun, GoDaddy, Namecheap, NameSilo...
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 17,111 stars and 1,859 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 194. [dutchcoders/transfer.sh](https://github.com/dutchcoders/transfer.sh) — 66.5/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** A client streams an upload to an HTTP endpoint; the server stores it under a generated or requested path and returns a short URL, with optional retention, authentication, and object-storage backends.
- **What it does:** Easy and fast file sharing from the command-line.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 15,872 stars and 1,583 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 195. [jackc/pgx](https://github.com/jackc/pgx) — 66.1/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** The pgx driver is a low-level, high performance interface that exposes PostgreSQL-specific features such as LISTEN / NOTIFY and COPY. It also includes an adapter for the standard database/sql interface.
- **What it does:** PostgreSQL driver and toolkit for Go.
- **Language:** Go (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 14,060 stars and 1,083 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 196. [pagehelper-org/Mybatis-PageHelper](https://github.com/pagehelper-org/Mybatis-PageHelper) — 65.6/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** A MyBatis interceptor examines mapped statements and pagination parameters, selects a database dialect, rewrites SQL with count and limit/offset clauses, and packages rows with page metadata.
- **What it does:** Mybatis通用分页插件.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 12,357 stars and 3,089 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 197. [crossoverJie/cim](https://github.com/crossoverJie/cim) — 64.8/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** Clients maintain Netty connections to gateway nodes; messages are authenticated and routed through server-side services, persisted or forwarded through the configured backend, and delivered with heartbeats and reconnect handling.
- **What it does:** 📲cim(cross IM) 适用于开发者的分布式即时通讯系统.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 9,495 stars and 2,872 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 198. [mik3y/usb-serial-for-android](https://github.com/mik3y/usb-serial-for-android) — 63.1/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** This is a driver library for communication with Arduinos and other USB serial hardware on Android, using the Android USB Host Mode (OTG) available since Android 3.1 and working reliably since Android 4.2.
- **What it does:** Android USB host serial driver library for CDC, FTDI, Arduino and other devices.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 5,647 stars and 1,693 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 199. [rubenlagus/TelegramBots](https://github.com/rubenlagus/TelegramBots) — 63.0/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** It maps Telegram Bot API methods and update payloads to Java types, supports long polling and webhooks, dispatches incoming updates to bot handlers, and serializes outbound requests through an HTTP client.
- **What it does:** Java library to create bots using Telegram Bots API.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 5,487 stars and 1,359 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

### 200. [dreamhead/moco](https://github.com/dreamhead/moco) — 62.3/100

- **Why this library or tool exists:** The project solves a widely encountered operational or end-user problem with a reusable open-source implementation.
- **How it works:** It loads declarative JSON configurations or a Java DSL into request matchers and response handlers, then runs HTTP, HTTPS, WebSocket, or socket listeners that return deterministic stub responses.
- **What it does:** Easy Setup Stub Server.
- **Language:** Java (GitHub primary-language classification; mixed-language repositories require a composition audit).
- **Industry/adoption signal:** 4,447 stars and 1,082 forks in the snapshot. This is a public adoption proxy, not a deployment census.
- **Ability to improve readability or performance using Go+ and discipline:** Separate pure domain decisions from effects, encode lifecycle and authorization as sums/typestates, use immutable snapshots, and generate stable Go boundaries. Focus performance work on measured serialization and coordination hot paths.
- **Possible Go+ standard-library pressure:** No immediate `std` proposal. Keep the first rewrite in GoForge and promote only independently reused, law-bearing primitives.

## Evidence and maintenance

- Candidate links above are the primary repository sources.
- License gating used GitHub repository search with `license:mit` plus each result's `license.spdx_id == MIT` metadata, followed by direct inspection of the license grant at the repository default branch. This proves the repository-level gate for triage; a selected revision still needs a tag-pinned provenance and bundled-artifact audit before code is imported.
- Stars, forks, primary language, default branch, topics, and descriptions were captured on 2026-07-23.
- READMEs were retrieved for all 200 candidates and used to check the mechanism summary. Corrupt extractions such as badges, navigation text, contributor instructions, and advisory fragments were discarded rather than treated as architecture.
- “Industry/adoption signal” intentionally reports reproducible public stars and forks. It does **not** turn popularity into an unsupported claim that a named company runs the software. Named production users should be added only from a project-maintained adopters list, a public dependency graph, or a first-party engineering source.
- Refresh this audit before each rewrite tranche. Projects move, relicense, archive, change primary languages, or acquire non-MIT assets.

## Recommended first tranches

1. **Semantic standard-library forcing cases:** validation, configuration, parsing, workflows, typed routing, protocol ownership, and storage transaction state.
2. **Performance forcing cases:** media pipelines, zero-copy protocol parsers, compiler passes, high-throughput observability, and Python/JVM orchestration hot paths.
3. **Discipline forcing cases:** release/build tools, durable agents, terminal update loops, and large application boundaries where pure decisions can be separated from effects.
