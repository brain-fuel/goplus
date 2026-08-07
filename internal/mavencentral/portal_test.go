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

func TestPortalUploadAndWait(t *testing.T) {
	var statusCalls int
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer dXNlcjpwYXNz" {
			http.Error(w, "auth", 401)
			return
		}
		switch r.URL.Path {
		case "/api/v1/publisher/upload":
			if r.URL.Query().Get("publishingType") != "AUTOMATIC" {
				t.Errorf("publishingType=%s", r.URL.Query().Get("publishingType"))
			}
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
			}
			f, _, err := r.FormFile("bundle")
			if err != nil {
				t.Error(err)
			} else {
				b, _ := io.ReadAll(f)
				if string(b) != "zip" {
					t.Errorf("bundle=%q", b)
				}
			}
			w.WriteHeader(201)
			io.WriteString(w, "deployment-id")
		case "/api/v1/publisher/status":
			statusCalls++
			w.Header().Set("Content-Type", "application/json")
			state := "VALIDATING"
			if statusCalls > 1 {
				state = "PUBLISHED"
			}
			io.WriteString(w, `{"deploymentId":"deployment-id","deploymentState":"`+state+`","purls":["pkg:maven/dev.goforge/demo@1"]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()
	path := filepath.Join(t.TempDir(), "bundle.zip")
	if err := os.WriteFile(path, []byte("zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := Client{BaseURL: s.URL, Username: "user", Password: "pass", HTTP: s.Client()}
	id, err := c.Upload(context.Background(), path, "demo", true)
	if err != nil {
		t.Fatal(err)
	}
	if id != "deployment-id" {
		t.Fatal(id)
	}
	d, err := c.Wait(context.Background(), id, time.Millisecond, false)
	if err != nil {
		t.Fatal(err)
	}
	if d.DeploymentState != "PUBLISHED" || len(d.PURLs) != 1 {
		t.Fatalf("%+v", d)
	}
}

func TestPortalUserManagedStopsAtValidated(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"deploymentId":"deployment-id","deploymentState":"VALIDATED"}`)
	}))
	defer s.Close()
	d, err := (Client{BaseURL: s.URL, Username: "user", Password: "pass", HTTP: s.Client()}).Wait(context.Background(), "deployment-id", time.Millisecond, true)
	if err != nil || d.DeploymentState != "VALIDATED" {
		t.Fatalf("deployment=%+v error=%v", d, err)
	}
}

func TestPortalRequiresCredentials(t *testing.T) {
	_, err := (Client{}).Status(context.Background(), "deployment-id")
	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("error=%v", err)
	}
}

func TestPortalRejectsHTTPError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no namespace", 403) }))
	defer s.Close()
	path := filepath.Join(t.TempDir(), "x")
	os.WriteFile(path, []byte("x"), 0o644)
	_, err := Client{BaseURL: s.URL, Username: "u", Password: "p", HTTP: s.Client()}.Upload(context.Background(), path, "x", false)
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("error=%v", err)
	}
}
