package websocket

import (
	"net/http"
	"testing"
)

func validServerRequest() *http.Request {
	return &http.Request{
		Method:     http.MethodGet,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Host:       "example.test",
		Header: http.Header{
			"Connection":            {"keep-alive", "Upgrade"},
			"Upgrade":               {"websocket"},
			"Sec-Websocket-Version": {"13"},
			"Sec-Websocket-Key":     {"dGhlIHNhbXBsZSBub25jZQ=="},
		},
	}
}

func TestValidateServerRequest(t *testing.T) {
	request := validServerRequest()
	if key, err := ValidateServerRequest(request); err != nil || key != "dGhlIHNhbXBsZSBub25jZQ==" {
		t.Fatalf("key=%q error=%v", key, err)
	}
}

func TestValidateServerRequestRejectsAmbiguity(t *testing.T) {
	tests := map[string]func(*http.Request){
		"missing host": func(r *http.Request) { r.Host = "" },
		"duplicate key": func(r *http.Request) {
			r.Header["Sec-Websocket-Key"] = append(r.Header["Sec-Websocket-Key"], "dGhlIHNhbXBsZSBub25jZQ==")
		},
		"combined key":      func(r *http.Request) { r.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==, other") },
		"duplicate version": func(r *http.Request) { r.Header["Sec-Websocket-Version"] = []string{"13", "13"} },
		"invalid protocol":  func(r *http.Request) { r.Header.Set("Sec-WebSocket-Protocol", "good, bad protocol") },
		"wrong method":      func(r *http.Request) { r.Method = http.MethodPost },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := validServerRequest()
			mutate(request)
			if _, err := ValidateServerRequest(request); err != ErrHandshake {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestHeaderTokensAcrossRepeatedFields(t *testing.T) {
	header := http.Header{"Connection": {"keep-alive", "Upgrade"}}
	if !hasToken(joinedHeader(header, "Connection"), "upgrade") {
		t.Fatal("upgrade token not found")
	}
}
