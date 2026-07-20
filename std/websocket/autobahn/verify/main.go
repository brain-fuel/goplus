package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	reports := flag.String("reports", "reports", "Autobahn report directory")
	agent := flag.String("agent", "", "only verify reports for this agent")
	expect := flag.Int("expect", 0, "require exactly this many case reports")
	flag.Parse()
	failed, total := 0, 0
	err := filepath.Walk(*reports, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".json") || info.Name() == "index.json" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc map[string]any
		if json.Unmarshal(body, &doc) != nil {
			return nil
		}
		behavior, ok := doc["behavior"].(string)
		if !ok {
			return nil
		}
		if *agent != "" && doc["agent"] != *agent {
			return nil
		}
		total++
		switch strings.ToUpper(behavior) {
		case "OK", "NON-STRICT", "INFORMATIONAL":
		default:
			failed++
			fmt.Printf("FAIL %s: %s\n", path, behavior)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	if total == 0 {
		fmt.Fprintln(os.Stderr, "no Autobahn case results found")
		os.Exit(2)
	}
	if *expect > 0 && total != *expect {
		fmt.Fprintf(os.Stderr, "expected %d Autobahn cases, found %d\n", *expect, total)
		os.Exit(2)
	}
	fmt.Printf("Autobahn: %d cases, %d failures\n", total, failed)
	if failed != 0 {
		os.Exit(1)
	}
}
