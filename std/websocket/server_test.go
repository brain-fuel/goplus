package websocket

import (
	"net/http/httptest"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	request := httptest.NewRequest("GET", "https://example.test/socket", nil)
	if !SameOrigin(request) {
		t.Fatal("missing Origin should be accepted")
	}
	request.Header.Set("Origin", "https://EXAMPLE.test")
	if !SameOrigin(request) {
		t.Fatal("same origin should be accepted case-insensitively")
	}
	request.Header.Set("Origin", "https://example.test:443")
	if !SameOrigin(request) {
		t.Fatal("explicit default port should be accepted")
	}
	for _, origin := range []string{"https://attacker.test", "null", "://bad"} {
		request.Header.Set("Origin", origin)
		if SameOrigin(request) {
			t.Fatalf("accepted Origin %q", origin)
		}
	}
}

func TestSelectProtocolUsesServerPreference(t *testing.T) {
	if got := selectProtocol("chat, assay.v2", []string{"assay.v2", "chat"}); got != "assay.v2" {
		t.Fatalf("selected=%q", got)
	}
}
