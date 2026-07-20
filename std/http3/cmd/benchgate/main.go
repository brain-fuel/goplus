// Command benchgate executes the comparative HTTP/3 performance contract.
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
type result struct{ ns, bytes []float64 }

var line = regexp.MustCompile(`^Benchmark([^/]+)/([^\s-]+)-\d+\s+\d+\s+([0-9.]+) ns/op\s+([0-9.]+) B/op`)
var featureRow = regexp.MustCompile(`^\s*\|\s*([A-Za-z][A-Za-z0-9]*)\s*\|\s*([0-9]+(?:\.[0-9]+)?)\s*\|\s*$`)

func median(values []float64) float64 { sort.Float64s(values); return values[len(values)/2] }

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
	feature := flag.String("feature", "http3/features/performance.feature", "Gherkin performance contract")
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
	cmd := exec.Command("go", "test", "./http3", "-run", "^$", "-bench", "Benchmark("+strings.Join(names, "|")+")$", "-benchmem", "-benchtime=300ms", "-count=3")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		os.Exit(2)
	}
	results := make(map[string]map[string]*result)
	for _, raw := range strings.Split(string(out), "\n") {
		match := line.FindStringSubmatch(raw)
		if match == nil {
			continue
		}
		if results[match[1]] == nil {
			results[match[1]] = make(map[string]*result)
		}
		r := results[match[1]][match[2]]
		if r == nil {
			r = new(result)
			results[match[1]][match[2]] = r
		}
		ns, _ := strconv.ParseFloat(match[3], 64)
		allocated, _ := strconv.ParseFloat(match[4], 64)
		r.ns, r.bytes = append(r.ns, ns), append(r.bytes, allocated)
	}
	failed := false
	for _, requirement := range requirements {
		ours, reference := results[requirement.name]["goplus"], results[requirement.name]["quicgo"]
		if ours == nil || reference == nil {
			fmt.Printf("FAIL %-24s missing benchmark result\n", requirement.name)
			failed = true
			continue
		}
		ratio := median(reference.ns) / median(ours.ns)
		allocOK := median(ours.bytes) <= median(reference.bytes)
		ok := ratio >= requirement.ratio && allocOK
		status := "FAIL"
		if ok {
			status = "PASS"
		}
		fmt.Printf("%-4s %-24s %.2fx (required %.2fx), %.0f vs %.0f B/op\n", status, requirement.name, ratio, requirement.ratio, median(ours.bytes), median(reference.bytes))
		failed = failed || !ok
	}
	if failed {
		os.Exit(1)
	}
}
