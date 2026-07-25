# pathquery

`pathquery` is the format-neutral wildcard kernel shared by dynamic path
evaluators. `*` matches zero or more Unicode code points and `?` matches one.
Matching is immutable, concurrency-safe, UTF-8 aware, and allocation-free.

`Relation`, `ParseRelation`, `Relate`, and `RelateString` provide the
format-neutral predicate algebra shared by query languages. Formats still own
operand parsing and coercion; the stdlib owns equality/order duality,
complements, and wildcard relation semantics.

It was extracted from the GoForge GJSON compatibility evaluator so JSON, CBOR,
configuration, and future document-query packages do not each grow subtly
different wildcard implementations.
