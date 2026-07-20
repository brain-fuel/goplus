package websocket

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

type featureWorld struct {
	accept      string
	wire        []byte
	side        Side
	header      Header
	consumed    int
	err         error
	payload     []byte
	original    []byte
	mask        [4]byte
	assembler   Assembler
	message     Message
	text        string
	category    string
	compression CompressionOptions
	extension   string
	settings    compressionSettings
}

func (w *featureWorld) reset() { *w = featureWorld{} }

func TestFeatures(t *testing.T) {
	w := &featureWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) { w.reset(); return ctx, nil })
			sc.Step(`^I compute the accept key for "([^"]*)"$`, func(key string) { w.accept = AcceptKey(key) })
			sc.Step(`^the accept key is "([^"]*)"$`, func(want string) error {
				if w.accept != want {
					return fmt.Errorf("accept=%q", w.accept)
				}
				return nil
			})
			sc.Step(`^a final binary frame of length (\d+) received by a client$`, func(raw string) error {
				n, _ := strconv.ParseInt(raw, 10, 64)
				w.header = Header{FIN: true, Opcode: OpBinary, Length: n}
				w.side = ClientSide
				return nil
			})
			sc.Step(`^I encode and parse the header$`, func() {
				w.wire, w.err = AppendHeader(nil, w.header)
				if w.err != nil {
					return
				}
				w.header, w.consumed, w.err = ParseHeader(w.wire, w.side, false)
			})
			sc.Step(`^parsing succeeds and consumes (\d+) header bytes$`, func(raw string) error {
				want, _ := strconv.Atoi(raw)
				if w.err != nil || w.consumed != want {
					return fmt.Errorf("consumed=%d err=%v", w.consumed, w.err)
				}
				return nil
			})
			sc.Step(`^the parsed payload length is (\d+)$`, func(raw string) error {
				want, _ := strconv.ParseInt(raw, 10, 64)
				if w.header.Length != want {
					return fmt.Errorf("length=%d", w.header.Length)
				}
				return nil
			})
			sc.Step(`^the wire header bytes "([0-9a-f]+)"$`, func(raw string) error { var err error; w.wire, err = hex.DecodeString(raw); return err })
			sc.Step(`^a server parses the header$`, func() { _, _, w.err = ParseHeader(w.wire, ServerSide, false) })
			sc.Step(`^parsing fails with "([^"]+)"$`, func(part string) error {
				if w.err == nil || !strings.Contains(w.err.Error(), part) {
					return fmt.Errorf("err=%v", w.err)
				}
				return nil
			})
			sc.Step(`^payload "([^"]*)" and mask "([0-9a-f]{8})"$`, func(payload, key string) error {
				w.payload = []byte(payload)
				w.original = append([]byte(nil), w.payload...)
				raw, _ := hex.DecodeString(key)
				copy(w.mask[:], raw)
				return nil
			})
			sc.Step(`^I apply the mask twice$`, func() { Mask(w.payload, w.mask, 0); Mask(w.payload, w.mask, 0) })
			sc.Step(`^the payload is unchanged$`, func() error {
				if string(w.payload) != string(w.original) {
					return fmt.Errorf("payload=%x", w.payload)
				}
				return nil
			})
			sc.Step(`^an open message assembler$`, func() { w.assembler = Assembler{MaxMessage: 1 << 20} })
			sc.Step(`^I feed non-final text "([^"]*)"$`, func(s string) error {
				_, err := w.assembler.Feed(Header{Opcode: OpText, Length: int64(len(s))}, []byte(s))
				return err
			})
			sc.Step(`^I feed a ping "([^"]*)"$`, func(s string) error {
				m, err := w.assembler.Feed(Header{FIN: true, Opcode: OpPing, Length: int64(len(s))}, []byte(s))
				if err == nil {
					if _, ok := m.(PingMessage); !ok {
						return fmt.Errorf("message=%T", m)
					}
				}
				return err
			})
			sc.Step(`^I feed final continuation "([^"]*)"$`, func(s string) error {
				var err error
				w.message, err = w.assembler.Feed(Header{FIN: true, Opcode: OpContinuation, Length: int64(len(s))}, []byte(s))
				return err
			})
			sc.Step(`^the completed text is "([^"]*)"$`, func(want string) error {
				m, ok := w.message.(TextMessage)
				if !ok || string(m.Payload) != want {
					return fmt.Errorf("message=%#v", w.message)
				}
				return nil
			})
			sc.Step(`^I feed final text bytes "([0-9a-f]+)"$`, func(raw string) {
				b, _ := hex.DecodeString(raw)
				_, w.err = w.assembler.Feed(Header{FIN: true, Opcode: OpText, Length: int64(len(b))}, b)
			})
			sc.Step(`^assembly fails with "([^"]+)"$`, func(part string) error {
				if w.err == nil || !strings.Contains(w.err.Error(), part) {
					return fmt.Errorf("err=%v", w.err)
				}
				return nil
			})
			sc.Step(`^I parse a close payload with code (\d+)$`, func(raw string) {
				n, _ := strconv.Atoi(raw)
				var p [2]byte
				binary.BigEndian.PutUint16(p[:], uint16(n))
				_, _, w.err = ParseClosePayload(p[:])
			})
			sc.Step(`^close parsing fails$`, func() error {
				if !errors.Is(w.err, ErrInvalidCloseCode) {
					return fmt.Errorf("err=%v", w.err)
				}
				return nil
			})
			sc.Step(`^Autobahn category (.+)$`, func(category string) { w.category = category })
			sc.Step(`^the conformance manifest marks it required$`, func() error {
				body, err := os.ReadFile("autobahn/conformance-manifest.json")
				if err != nil {
					return err
				}
				var required map[string]bool
				if err = json.Unmarshal(body, &required); err != nil {
					return err
				}
				if !required[w.category] {
					return fmt.Errorf("category %q is not required", w.category)
				}
				return nil
			})
			sc.Step(`^compression options with client window (\d+) and server window (\d+)$`, func(client, server int) {
				w.compression = CompressionOptions{ClientMaxWindowBits: client, ServerMaxWindowBits: server}
			})
			sc.Step(`^I build the permessage-deflate offer$`, func() {
				w.extension, w.err = compressionOffer(w.compression)
			})
			sc.Step(`^the extension offer is "([^"]*)"$`, func(want string) error {
				if w.err != nil || w.extension != want {
					return fmt.Errorf("offer=%q error=%v", w.extension, w.err)
				}
				return nil
			})
			sc.Step(`^I negotiate offer "([^"]*)"$`, func(offer string) {
				w.extension, w.settings, w.err = negotiateCompression(offer, w.compression)
			})
			sc.Step(`^the server write window is (\d+)$`, func(want int) error {
				if w.err != nil || w.settings.writeWindow != want {
					return fmt.Errorf("write window=%d error=%v", w.settings.writeWindow, w.err)
				}
				return nil
			})
			sc.Step(`^the server read window is (\d+)$`, func(want int) error {
				if w.settings.readWindow != want {
					return fmt.Errorf("read window=%d", w.settings.readWindow)
				}
				return nil
			})
			sc.Step(`^I validate response "([^"]*)"$`, func(response string) {
				_, w.settings, w.err = acceptCompressionResponse(response, w.compression)
			})
			sc.Step(`^extension negotiation fails$`, func() error {
				if !errors.Is(w.err, ErrInvalidExtension) {
					return fmt.Errorf("error=%v", w.err)
				}
				return nil
			})
			sc.Step(`^(\d+) repeated bytes compressed with window (\d+)$`, func(size, bits int) {
				w.payload, w.err = deflateMessage(bytes.Repeat([]byte{'x'}, size), bits)
			})
			sc.Step(`^I decompress with a (\d+) byte limit$`, func(limit int) {
				if w.err == nil {
					_, w.err = inflateMessage(w.payload, int64(limit))
				}
			})
			sc.Step(`^decompression fails with message too large$`, func() error {
				if !errors.Is(w.err, ErrMessageTooLarge) {
					return fmt.Errorf("error=%v", w.err)
				}
				return nil
			})
		},
		Options: &godog.Options{Format: "pretty", Paths: []string{"features"}, Tags: "~@benchmark", TestingT: t},
	}
	if status := suite.Run(); status != 0 {
		t.Fatalf("feature suite exited %d", status)
	}
}
