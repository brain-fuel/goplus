// Package cas provides a typed compare-and-swap vocabulary for stores whose
// values may change between observation and mutation.
package cas

// Observation binds a key to the version and value seen by a reader.
// Backends must compare Version atomically when Apply is attempted.
type Observation[K comparable, V any] struct {
	Key     K
	Version string
	Value   V
	Exists  bool
}

// Result describes every outcome of a compare-and-swap operation.
type Result[V any] enum {
	Updated(value V, version string)
	Changed(actual V, version string)
	Missing()
}

// Store is the minimal backend contract. Implementations define their version
// representation; callers treat it as opaque.
type Store[K comparable, V any] interface {
	Observe(K) (Observation[K, V], error)
	Apply(Observation[K, V], V) (Result[V], error)
}
