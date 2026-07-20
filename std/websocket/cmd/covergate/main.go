package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	profile := flag.String("profile", "", "Go coverprofile to verify")
	feature := flag.String("feature", "websocket/features/coverage.feature", "Gherkin coverage contract")
	flag.Parse()
	if *profile == "" {
		fmt.Fprintln(os.Stderr, "covergate: -profile is required")
		os.Exit(2)
	}
	required, err := requiredCoverage(*feature)
	if err != nil {
		fmt.Fprintln(os.Stderr, "covergate:", err)
		os.Exit(2)
	}
	covered, total, err := coverage(*profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "covergate:", err)
		os.Exit(2)
	}
	percent := 100 * float64(covered) / float64(total)
	if percent+1e-12 < required {
		fmt.Fprintf(os.Stderr, "FAIL handwritten coverage %.2f%% (%d/%d), required %.2f%%\n", percent, covered, total, required)
		os.Exit(1)
	}
	fmt.Printf("PASS handwritten coverage %.2f%% (%d/%d)\n", percent, covered, total)
}

func requiredCoverage(path string) (float64, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	const prefix = "Then handwritten statement coverage is "
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) && strings.HasSuffix(line, " percent") {
			raw := strings.TrimSuffix(strings.TrimPrefix(line, prefix), " percent")
			return strconv.ParseFloat(raw, 64)
		}
	}
	return 0, fmt.Errorf("no coverage requirement in %s", path)
}

func coverage(path string) (covered, total int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			if !strings.HasPrefix(line, "mode: ") {
				return 0, 0, fmt.Errorf("invalid coverprofile header")
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return 0, 0, fmt.Errorf("invalid coverprofile line %q", line)
		}
		fileAndRange := fields[0]
		colon := strings.LastIndexByte(fileAndRange, ':')
		if colon < 0 {
			return 0, 0, fmt.Errorf("invalid coverprofile range %q", fileAndRange)
		}
		fileName := fileAndRange[:colon]
		if strings.HasSuffix(fileName, "_gp.go") {
			continue
		}
		statements, parseErr := strconv.Atoi(fields[1])
		if parseErr != nil {
			return 0, 0, parseErr
		}
		count, parseErr := strconv.ParseUint(fields[2], 10, 64)
		if parseErr != nil {
			return 0, 0, parseErr
		}
		total += statements
		if count != 0 {
			covered += statements
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}
	if total == 0 {
		return 0, 0, fmt.Errorf("profile contains no handwritten statements")
	}
	return covered, total, nil
}
