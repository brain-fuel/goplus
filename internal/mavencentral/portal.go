package mavencentral

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const PortalURL = "https://central.sonatype.com"

type Client struct {
	BaseURL, Username, Password string
	HTTP                        *http.Client
}
type Deployment struct {
	DeploymentID    string   `json:"deploymentId"`
	DeploymentName  string   `json:"deploymentName"`
	DeploymentState string   `json:"deploymentState"`
	PURLs           []string `json:"purls"`
	Errors          any      `json:"errors"`
}

func (c Client) Upload(ctx context.Context, bundle, name string, automatic bool) (string, error) {
	if err := c.validateCredentials(); err != nil {
		return "", err
	}
	data, err := os.ReadFile(bundle)
	if err != nil {
		return "", err
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("bundle", filepath.Base(bundle))
	if err != nil {
		return "", err
	}
	if _, err = part.Write(data); err != nil {
		return "", err
	}
	if err = w.Close(); err != nil {
		return "", err
	}
	typeName := "USER_MANAGED"
	if automatic {
		typeName = "AUTOMATIC"
	}
	endpoint := c.base() + "/api/v1/publisher/upload?publishingType=" + typeName + "&name=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.auth(req)
	resp, err := c.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("Central upload: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
func (c Client) Status(ctx context.Context, id string) (Deployment, error) {
	if err := c.validateCredentials(); err != nil {
		return Deployment{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base()+"/api/v1/publisher/status?id="+url.QueryEscape(id), nil)
	if err != nil {
		return Deployment{}, err
	}
	c.auth(req)
	resp, err := c.client().Do(req)
	if err != nil {
		return Deployment{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return Deployment{}, fmt.Errorf("Central status: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var d Deployment
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return Deployment{}, err
	}
	return d, nil
}
func (c Client) Wait(ctx context.Context, id string, interval time.Duration, acceptValidated bool) (Deployment, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	for {
		d, err := c.Status(ctx, id)
		if err != nil {
			return d, err
		}
		switch d.DeploymentState {
		case "PUBLISHED":
			return d, nil
		case "VALIDATED":
			if acceptValidated {
				return d, nil
			}
		case "FAILED":
			return d, fmt.Errorf("Central deployment failed: %v", d.Errors)
		}
		select {
		case <-ctx.Done():
			return d, ctx.Err()
		case <-time.After(interval):
		}
	}
}
func (c Client) validateCredentials() error {
	if strings.TrimSpace(c.Username) == "" || strings.TrimSpace(c.Password) == "" {
		return fmt.Errorf("Maven Central username and password must not be empty")
	}
	return nil
}
func (c Client) base() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return PortalURL
}
func (c Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 2 * time.Minute}
}
func (c Client) auth(r *http.Request) {
	token := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Password))
	r.Header.Set("Authorization", "Bearer "+token)
}
