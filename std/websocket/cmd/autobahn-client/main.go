// Command autobahn-client runs this implementation against an Autobahn
// fuzzingserver instance.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"strconv"

	"goforge.dev/goplus/std/websocket"
)

func endpoint(base, path string, query url.Values) string {
	u, err := url.Parse(base)
	if err != nil {
		log.Fatal(err)
	}
	u.Path = path
	u.RawQuery = query.Encode()
	return u.String()
}

func readOne(rawURL string) websocket.Message {
	c, _, err := websocket.Dial(context.Background(), rawURL, websocket.DialOptions{Compression: autobahnCompression()})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	m, err := c.ReadMessage()
	if err != nil {
		log.Fatal(err)
	}
	return m
}

func autobahnCompression() *websocket.CompressionOptions {
	// Advertising the maximum value lets the fuzzing server exercise every
	// legal client_max_window_bits response (8..15) in one run.
	return &websocket.CompressionOptions{ClientMaxWindowBits: 15}
}

func main() {
	base := flag.String("url", "ws://127.0.0.1:9001", "Autobahn fuzzingserver URL")
	agent := flag.String("agent", "goplus-std-websocket", "report agent name")
	flag.Parse()
	countMessage := readOne(endpoint(*base, "/getCaseCount", nil))
	countText, ok := countMessage.(websocket.TextMessage)
	if !ok {
		log.Fatalf("case count message is %T", countMessage)
	}
	count, err := strconv.Atoi(string(countText.Payload))
	if err != nil {
		log.Fatal(err)
	}
	for caseNumber := 1; caseNumber <= count; caseNumber++ {
		rawURL := endpoint(*base, "/runCase", url.Values{"case": {strconv.Itoa(caseNumber)}, "agent": {*agent}})
		c, _, err := websocket.Dial(context.Background(), rawURL, websocket.DialOptions{Compression: autobahnCompression()})
		if err != nil {
			log.Printf("case %d dial: %v", caseNumber, err)
			continue
		}
		for {
			m, readErr := c.ReadMessage()
			if readErr != nil {
				if readErr != io.EOF {
					log.Printf("case %d read: %v", caseNumber, readErr)
				}
				break
			}
			if _, unsolicited := m.(websocket.PongMessage); unsolicited {
				continue
			}
			if err = c.WriteMessage(m); err != nil {
				log.Printf("case %d write: %v", caseNumber, err)
				break
			}
		}
		_ = c.Close()
		fmt.Printf("case %d/%d\r", caseNumber, count)
	}
	report := endpoint(*base, "/updateReports", url.Values{"agent": {*agent}})
	c, _, err := websocket.Dial(context.Background(), report, websocket.DialOptions{})
	if err != nil {
		log.Fatal(err)
	}
	_ = c.Close()
	fmt.Printf("\ncompleted %d cases\n", count)
}
