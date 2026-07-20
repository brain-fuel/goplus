package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net/http"

	"goforge.dev/goplus/std/websocket"
)

func main() {
	addr := flag.String("listen", ":9001", "listen address")
	flag.Parse()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := websocket.Upgrade(w, r, websocket.UpgradeOptions{
			Compression: &websocket.CompressionOptions{},
			Config:      websocket.ConnConfig{MaxFrame: 1 << 30, MaxMessage: 1 << 30},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer conn.Close()
		for {
			m, err := conn.ReadMessage()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					log.Printf("read: %v", err)
				}
				return
			}
			if _, unsolicited := m.(websocket.PongMessage); unsolicited {
				continue
			}
			if err = conn.WriteMessage(m); err != nil {
				log.Printf("write: %v", err)
				return
			}
		}
	})
	log.Printf("Go+ WebSocket Autobahn testee listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, h))
}
