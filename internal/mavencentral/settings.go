package mavencentral

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Credentials are Central Portal user-token credentials.
type Credentials struct{ Username, Password, Source string }

type mavenSettings struct {
	Servers []mavenServer `xml:"servers>server"`
}
type mavenServer struct {
	ID       string `xml:"id"`
	Username string `xml:"username"`
	Password string `xml:"password"`
}

// LoadCredentials discovers credentials without passing secrets on the command
// line. Environment variables take precedence, followed by a project-root
// maven_settings.xml and the conventional user Maven settings file.
func LoadCredentials(root string) (Credentials, error) {
	if username, password := strings.TrimSpace(os.Getenv("MAVEN_CENTRAL_USERNAME")), strings.TrimSpace(os.Getenv("MAVEN_CENTRAL_PASSWORD")); username != "" || password != "" {
		if username == "" || password == "" {
			return Credentials{}, fmt.Errorf("both MAVEN_CENTRAL_USERNAME and MAVEN_CENTRAL_PASSWORD must be set")
		}
		return Credentials{Username: username, Password: password, Source: "environment"}, nil
	}
	var candidates []string
	if explicit := strings.TrimSpace(os.Getenv("MAVEN_SETTINGS")); explicit != "" {
		candidates = append(candidates, explicit)
	} else {
		candidates = append(candidates, filepath.Join(root, "maven_settings.xml"))
		if parent := filepath.Dir(root); parent != root {
			candidates = append(candidates, filepath.Join(parent, "maven_settings.xml"))
		}
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, ".m2", "settings.xml"))
		}
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return Credentials{}, err
		}
		var settings mavenSettings
		if err := xml.Unmarshal(data, &settings); err != nil {
			return Credentials{}, fmt.Errorf("reading Maven settings %s: %w", path, err)
		}
		server, ok := selectServer(settings.Servers)
		if !ok {
			return Credentials{}, fmt.Errorf("Maven settings %s has no Central server credentials", path)
		}
		username, err := expandSetting(server.Username)
		if err != nil {
			return Credentials{}, err
		}
		password, err := expandSetting(server.Password)
		if err != nil {
			return Credentials{}, err
		}
		if username == "" || password == "" {
			return Credentials{}, fmt.Errorf("Maven settings %s has empty Central credentials", path)
		}
		return Credentials{Username: username, Password: password, Source: path}, nil
	}
	return Credentials{}, fmt.Errorf("Central credentials not found; set MAVEN_CENTRAL_USERNAME/PASSWORD or provide maven_settings.xml")
}

func selectServer(servers []mavenServer) (mavenServer, bool) {
	for _, server := range servers {
		if server.ID == "central" || server.ID == "maven-central" {
			return server, true
		}
	}
	if len(servers) == 1 {
		return servers[0], true
	}
	return mavenServer{}, false
}

func expandSetting(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "${env.") && strings.HasSuffix(value, "}") {
		name := strings.TrimSuffix(strings.TrimPrefix(value, "${env."), "}")
		if expanded := os.Getenv(name); expanded != "" {
			return expanded, nil
		}
		return "", fmt.Errorf("Maven settings references unset environment variable %s", name)
	}
	return value, nil
}
