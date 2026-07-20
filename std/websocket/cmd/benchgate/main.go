// Command benchgate executes the comparative performance contract.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type requirement struct {
	name  string
	ratio float64
}

type result struct {
	ns    []float64
	bytes []float64
}

var line = regexp.MustCompile(`^Benchmark([^/]+)/([^\s-]+)-\d+\s+\d+\s+([0-9.]+) ns/op\s+([0-9.]+) B/op`)
var featureRow = regexp.MustCompile(`^\s*\|\s*([A-Za-z][A-Za-z0-9]*)\s*\|\s*([0-9]+(?:\.[0-9]+)?)\s*\|\s*$`)

func median(v []float64) float64 {
	sort.Float64s(v)
	return v[len(v)/2]
}

func readRequirements(path string) ([]requirement, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var requirements []requirement
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		match := featureRow.FindStringSubmatch(scanner.Text())
		if match == nil || match[1] == "benchmark" {
			continue
		}
		ratio, err := strconv.ParseFloat(match[2], 64)
		if err != nil || ratio <= 0 {
			return nil, fmt.Errorf("invalid performance ratio in %q", scanner.Text())
		}
		requirements = append(requirements, requirement{name: match[1], ratio: ratio})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(requirements) == 0 {
		return nil, fmt.Errorf("no benchmark requirements in %s", path)
	}
	return requirements, nil
}

func main() {
	feature := flag.String("feature", "websocket/features/performance.feature", "Gherkin performance contract")
	flag.Parse()
	requirements, err := readRequirements(*feature)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	names := make([]string, len(requirements))
	for i, requirement := range requirements {
		names[i] = regexp.QuoteMeta(requirement.name)
	}
	cmd := exec.Command("go", "test", "./websocket", "-run", "^$", "-bench", "Benchmark("+strings.Join(names, "|")+")$", "-benchmem", "-benchtime=300ms", "-count=3")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		os.Exit(2)
	}
	results := make(map[string]map[string]*result)
	for _, raw := range strings.Split(string(out), "\n") {
		m := line.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		if results[m[1]] == nil {
			results[m[1]] = make(map[string]*result)
		}
		r := results[m[1]][m[2]]
		if r == nil {
			r = &result{}
			results[m[1]][m[2]] = r
		}
		ns, _ := strconv.ParseFloat(m[3], 64)
		bytes, _ := strconv.ParseFloat(m[4], 64)
		r.ns, r.bytes = append(r.ns, ns), append(r.bytes, bytes)
	}
	failed := false
	for _, req := range requirements {
		ours, ref := results[req.name]["goplus"], results[req.name]["gobwas"]
		if ours == nil || ref == nil {
			fmt.Printf("FAIL %-18s missing benchmark result\n", req.name)
			failed = true
			continue
		}
		ratio := median(ref.ns) / median(ours.ns)
		allocOK := median(ours.bytes) <= median(ref.bytes)
		ok := ratio >= req.ratio && allocOK
		fmt.Printf("%-4s %-18s %.2fx (required %.2fx), %.0f vs %.0f B/op\n", map[bool]string{true: "PASS", false: "FAIL"}[ok], req.name, ratio, req.ratio, median(ours.bytes), median(ref.bytes))
		failed = failed || !ok
	}
	if failed {
		os.Exit(1)
	}
}
