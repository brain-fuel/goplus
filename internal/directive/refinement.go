package directive

import (
	"strconv"
	"strings"
)

const refinementPrefix = "//goplus:refinement"

// RefinementMarker preserves a refinement declaration after it erases to a Go
// alias. Quoted fields make arbitrary base and predicate expressions durable.
type RefinementMarker struct {
	Name      string
	Binder    string
	Base      string
	Predicate string
}

func (m RefinementMarker) String() string {
	return refinementPrefix + " " + strconv.Quote(m.Name) + " " +
		strconv.Quote(m.Binder) + " " + strconv.Quote(m.Base) + " " +
		strconv.Quote(m.Predicate)
}

func ParseRefinementMarker(line string) (RefinementMarker, bool) {
	rest, ok := cutDirective(line, refinementPrefix)
	if !ok {
		return RefinementMarker{}, false
	}
	fields := make([]string, 0, 4)
	for len(strings.TrimSpace(rest)) > 0 {
		rest = strings.TrimSpace(rest)
		q, tail, ok := cutQuoted(rest)
		if !ok {
			return RefinementMarker{}, false
		}
		fields = append(fields, q)
		rest = tail
	}
	if len(fields) != 4 || fields[0] == "" || fields[1] == "" || fields[2] == "" || fields[3] == "" {
		return RefinementMarker{}, false
	}
	return RefinementMarker{Name: fields[0], Binder: fields[1], Base: fields[2], Predicate: fields[3]}, true
}

func cutQuoted(s string) (string, string, bool) {
	if len(s) == 0 || s[0] != '"' {
		return "", s, false
	}
	escaped := false
	for i := 1; i < len(s); i++ {
		switch {
		case escaped:
			escaped = false
		case s[i] == '\\':
			escaped = true
		case s[i] == '"':
			v, err := strconv.Unquote(s[:i+1])
			return v, s[i+1:], err == nil
		}
	}
	return "", s, false
}
