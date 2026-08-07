package mavencentral

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPersistentOpenPGPSigningIsDeterministic(t *testing.T) {
	epoch := time.Unix(1_700_000_000, 0).UTC()
	path := filepath.Join(t.TempDir(), "publisher.asc")
	key, err := EnsureSigningKey(path, "GoForge", "opensource@goforge.dev", epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !key.Created || len(key.Fingerprint) != 40 {
		t.Fatalf("key=%+v", key)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("private key mode=%v error=%v", info.Mode().Perm(), err)
	}
	message := filepath.Join(t.TempDir(), "artifact.jar")
	if err := os.WriteFile(message, []byte("artifact"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := key.Signer.Sign(context.Background(), message)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := EnsureSigningKey(path, "ignored", "ignored@example.com", epoch)
	if err != nil {
		t.Fatal(err)
	}
	second, err := reloaded.Signer.Sign(context.Background(), message)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || reloaded.Created || reloaded.Fingerprint != key.Fingerprint {
		t.Fatal("persistent signing was not deterministic")
	}
	if err := reloaded.Signer.Verify([]byte("artifact"), second); err != nil {
		t.Fatal(err)
	}
}

func TestPublishPublicKey(t *testing.T) {
	var submitted string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			io.WriteString(w, "-----BEGIN PGP PUBLIC KEY BLOCK-----")
			return
		}
		body, _ := io.ReadAll(r.Body)
		submitted = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	if err := PublishPublicKey(context.Background(), server.Client(), server.URL, "ABC123", []byte("PUBLIC KEY")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(submitted, "keytext=PUBLIC+KEY") {
		t.Fatalf("form=%q", submitted)
	}
}
