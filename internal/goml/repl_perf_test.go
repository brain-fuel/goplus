package goml

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// The REPL compiles and runs every evaluation, so latency is a design
// constraint rather than an afterthought. This measures it and fails
// only on a pathological regression: the exact numbers are reported for
// the record, since they depend on the host.
func TestREPLLatencyBudget(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()

	measure := func(inputs []string) []time.Duration {
		var out []time.Duration
		for _, in := range inputs {
			var stdout, stderr bytes.Buffer
			start := time.Now()
			REPL(strings.NewReader(in+"\n:quit\n"), &stdout, &stderr, REPLOptions{Dir: dir})
			out = append(out, time.Since(start))
		}
		return out
	}

	// One warm-up run pays for the cold build cache.
	measure([]string{"0"})

	const iterations = 10
	exprs := make([]string, iterations)
	decls := make([]string, iterations)
	for i := range exprs {
		exprs[i] = fmt.Sprintf("%d + 1", i)
		decls[i] = fmt.Sprintf("let D%d (n : Int) : Int := n + %d", i, i)
	}

	report := func(label string, ds []time.Duration, ceiling time.Duration) {
		sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
		p50 := ds[len(ds)/2]
		p95 := ds[min(len(ds)*95/100, len(ds)-1)]
		t.Logf("%s: p50=%s p95=%s", label, p50.Round(time.Millisecond), p95.Round(time.Millisecond))
		if p50 > ceiling {
			t.Errorf("%s p50 = %s, over the %s ceiling", label, p50.Round(time.Millisecond), ceiling)
		}
	}
	// Generous ceilings: this catches a pathological regression (a lost
	// build cache, a re-run pipeline) without flaking on a loaded host.
	report("expression", measure(exprs), 4*time.Second)
	report("declaration", measure(decls), 3*time.Second)
}
