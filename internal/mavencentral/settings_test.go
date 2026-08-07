package mavencentral

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCredentialsFromWorkspaceSettings(t *testing.T) {
	workspace := t.TempDir()
	project := filepath.Join(workspace, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := `<settings><servers><server><id>${server}</id><username>token-user</username><password>token-password</password></server></servers></settings>`
	if err := os.WriteFile(filepath.Join(workspace, "maven_settings.xml"), []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}
	credentials, err := LoadCredentials(project)
	if err != nil {
		t.Fatal(err)
	}
	if credentials.Username != "token-user" || credentials.Password != "token-password" {
		t.Fatalf("credentials were not loaded")
	}
}

func TestLoadCredentialsExpandsEnvironment(t *testing.T) {
	t.Setenv("TEST_CENTRAL_USER", "user")
	t.Setenv("TEST_CENTRAL_PASSWORD", "password")
	path := filepath.Join(t.TempDir(), "settings.xml")
	settings := `<settings><servers><server><id>central</id><username>${env.TEST_CENTRAL_USER}</username><password>${env.TEST_CENTRAL_PASSWORD}</password></server></servers></settings>`
	if err := os.WriteFile(path, []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MAVEN_SETTINGS", path)
	credentials, err := LoadCredentials(t.TempDir())
	if err != nil || credentials.Username != "user" || credentials.Password != "password" {
		t.Fatalf("credentials=%+v error=%v", credentials, err)
	}
}
