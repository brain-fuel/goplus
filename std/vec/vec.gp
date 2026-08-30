// Package vec is a length-indexed sequence: Vec[T, n] carries its
// length in its type, so First and Rest exist only for non-empty
// vectors — the checker proves it, erasure drops it, and the generated
// Go stays a plain cons-list any Go program can use (guarded at the
// exported boundary).
package vec

type Vec[T any, n nat] enum {
	Nil() Vec[T, 0]
	Cons(Head T, Tail Vec[T, n]) Vec[T, n+1]
}

// Pair is the shape-preserving product used by Zip.
type Pair[A any, B any] struct { First A; Second B }

// Fin[n] is evidence that an index is strictly smaller than n.
type Fin[n nat] enum {
	Zero() Fin[n+1]
	Succ(Prev Fin[n]) Fin[n+1]
}

// First returns the head of a non-empty vector. The Nil arm is
// impossible — the index says so, and now the source does too.
func First[T any](0 n nat, v Vec[T, n+1]) T {
	match v {
	case Cons(h, t):
		_ = t
		return h
	case Nil():
		impossible
	}
}

// Rest returns everything after the head of a non-empty vector.
func Rest[T any](0 n nat, v Vec[T, n+1]) Vec[T, n] {
	match v {
	case Cons(h, t):
		_ = h
		return t
	case Nil():
		impossible
	}
}

// Length walks the spine; the result equals the index by construction.
func Length[T any](0 n nat, v Vec[T, n]) int {
	total := 0
	match v {
	case Nil():
	case Cons(h, t):
		_ = h
		total = Length(t) + 1
	}
	return total
}

// Replicate builds the vector of n copies of x.
func Replicate[T any](n nat, x T) Vec[T, n] {
	if n == 0 {
		return Nil[T]()
	}
	return Cons(x, Replicate(n-1, x))
}

// Concat appends b to a; the indices add.
func Concat[T any](0 n nat, 0 m nat, a Vec[T, n], b Vec[T, m]) Vec[T, n+m] {
	match a {
	case Nil():
		return b
	case Cons(h, t):
		return Cons(h, Concat(t, b))
	}
}

// Map transforms every element, preserving the length.
func Map[T any, U any](0 n nat, f func(T) U, v Vec[T, n]) Vec[U, n] {
	match v {
	case Nil():
		return Nil[U]()
	case Cons(h, t):
		return Cons(f(h), Map(f, t))
	}
}

// Zip requires equal shapes and preserves that shape in its result.
func Zip[A any, B any](0 n nat, left Vec[A, n], right Vec[B, n]) Vec[Pair[A, B], n] {
	if Length(left) != Length(right) {
		panic("vec.Zip: vectors have different lengths")
	}
	return zipSame(left, right)
}

func zipSame[A any, B any](0 n nat, left Vec[A, n], right Vec[B, n]) Vec[Pair[A, B], n] {
	match left {
	case Nil():
		_ = right
		return Nil[Pair[A, B]]()
	case Cons(a, as):
		return Cons(Pair[A, B]{First: a, Second: First(right)}, zipSame(as, Rest(right)))
	}
}

// At is total: Fin[n] makes an out-of-bounds index unrepresentable in Go+.
func At[T any](0 n nat, index Fin[n], values Vec[T, n]) T {
	match index {
	case Zero():
		return First(values)
	case Succ(previous):
		return At(previous, Rest(values))
	}
}

// AtIndex selects by an ordinary number rather than Peano evidence. The
// bound travels as a proposition, so it is proved at the call and then
// erased: no Fin to build, and an out-of-range index is a compile error
// rather than a panic. Prefer Fin when the evidence is already in hand
// (recursing over an index), and AtIndex when you have a plain number.
func AtIndex[T any](i nat, 0 n nat, 0 p Lt[i, n], values Vec[T, n]) T {
	match values {
	case Cons(h, t):
		if i == 0 {
			return h
		}
		return AtIndex(i-1, decide, t)
	}
}

// Set replaces the element at i, the dual of AtIndex and bounded the
// same way. The length is preserved, so the result keeps its index.
func Set[T any](i nat, 0 n nat, 0 p Lt[i, n], x T, values Vec[T, n]) Vec[T, n] {
	match values {
	case Cons(h, t):
		if i == 0 {
			return Cons(x, t)
		}
		return Cons(h, Set(i-1, decide, x, t))
	}
}
