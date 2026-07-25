// Package pathquery supplies format-neutral matching primitives for dynamic
// path evaluators. It is authored in Go+ and generated into portable Go.
package pathquery

import "unicode/utf8"

// Pattern is an immutable wildcard pattern. '*' matches zero or more Unicode
// code points and '?' matches exactly one code point.
type Pattern struct{ source string }

func Compile(source string) Pattern { return Pattern{source: source} }
func (pattern Pattern) String() string { return pattern.source }
func (pattern Pattern) Match(value string) bool { return Match(pattern.source, value) }

// Relation is the format-neutral algebra used by path-query predicates.
// Concrete formats retain responsibility for parsing and coercing operands.
type Relation uint8

const (
	Equal Relation = iota
	NotEqual
	Less
	LessOrEqual
	Greater
	GreaterOrEqual
	Like
	NotLike
)

// ParseRelation recognizes the conventional dynamic-query spellings.
func ParseRelation(source string) (Relation, bool) {
	switch source {
	case "=", "==":
		return Equal, true
	case "!=":
		return NotEqual, true
	case "<":
		return Less, true
	case "<=":
		return LessOrEqual, true
	case ">":
		return Greater, true
	case ">=":
		return GreaterOrEqual, true
	case "%":
		return Like, true
	case "!%":
		return NotLike, true
	}
	return Equal, false
}

// Relate evaluates equality and ordering without imposing a format's scalar
// representation. Like and NotLike are intentionally handled by RelateString.
func Relate[T any](
	left, right T,
	relation Relation,
	equal func(T, T) bool,
	less func(T, T) bool,
) bool {
	switch relation {
	case Equal:
		return equal(left, right)
	case NotEqual:
		return !equal(left, right)
	case Less:
		return less(left, right)
	case LessOrEqual:
		return !less(right, left)
	case Greater:
		return less(right, left)
	case GreaterOrEqual:
		return !less(left, right)
	}
	return false
}

// RelateString adds wildcard Like/NotLike to ordinary lexical relations.
func RelateString(left, right string, relation Relation) bool {
	if relation == Like || relation == NotLike {
		matched := Match(right, left)
		return matched == (relation == Like)
	}
	return Relate(left, right, relation,
		func(left, right string) bool { return left == right },
		func(left, right string) bool { return left < right })
}

// Match compares UTF-8 strings without allocating. Invalid UTF-8 bytes are
// treated consistently as RuneError code points.
func Match(pattern, value string) bool {
	if ascii(pattern) && ascii(value) {
		return matchASCII(pattern, value)
	}
	return matchUTF8(pattern, value)
}

func matchUTF8(pattern, value string) bool {
	patternOffset, valueOffset := 0, 0
	afterStar, starValue := -1, -1
	for valueOffset < len(value) {
		if patternOffset < len(pattern) {
			patternRune, patternSize := utf8.DecodeRuneInString(pattern[patternOffset:])
			escaped := false
			if patternRune == '\\' && patternOffset+patternSize < len(pattern) {
				escaped = true
				nextRune, nextSize := utf8.DecodeRuneInString(pattern[patternOffset+patternSize:])
				patternRune = nextRune
				patternSize += nextSize
			}
			if patternRune == '*' && !escaped {
				afterStar = patternOffset + patternSize
				starValue = valueOffset
				patternOffset = afterStar
				continue
			}
			valueRune, valueSize := utf8.DecodeRuneInString(value[valueOffset:])
			if patternRune == '?' && !escaped || patternRune == valueRune {
				patternOffset += patternSize
				valueOffset += valueSize
				continue
			}
		}
		if afterStar < 0 {
			return false
		}
		_, valueSize := utf8.DecodeRuneInString(value[starValue:])
		starValue += valueSize
		valueOffset = starValue
		patternOffset = afterStar
	}
	for patternOffset < len(pattern) {
		patternRune, patternSize := utf8.DecodeRuneInString(pattern[patternOffset:])
		if patternRune != '*' {
			return false
		}
		patternOffset += patternSize
	}
	return true
}

func ascii(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func matchASCII(pattern, value string) bool {
	patternOffset, valueOffset := 0, 0
	afterStar, starValue := -1, -1
	for valueOffset < len(value) {
		if patternOffset < len(pattern) {
			token := pattern[patternOffset]
			size := 1
			escaped := false
			if token == '\\' && patternOffset+1 < len(pattern) {
				escaped = true
				token = pattern[patternOffset+1]
				size = 2
			}
			if token == '*' && !escaped {
				afterStar = patternOffset + size
				starValue = valueOffset
				patternOffset = afterStar
				continue
			}
			if token == '?' && !escaped || token == value[valueOffset] {
				patternOffset += size
				valueOffset++
				continue
			}
		}
		if afterStar < 0 {
			return false
		}
		starValue++
		valueOffset = starValue
		patternOffset = afterStar
	}
	for patternOffset < len(pattern) {
		if pattern[patternOffset] != '*' {
			return false
		}
		patternOffset++
	}
	return true
}
