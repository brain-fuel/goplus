package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"goforge.dev/goplus/internal/artifactio"
	"goforge.dev/goplus/internal/javatool"
	"goforge.dev/goplus/internal/mavencentral"
	"goforge.dev/goplus/internal/projectconfig"
)

func runPublish(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("goplus publish", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "java", "publication target (java only)")
	bundleOnly := fs.Bool("bundle-only", false, "build and inspect the signed Central bundle without uploading")
	automatic := fs.Bool("automatic", true, "publish automatically after Central validation")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *target != "java" {
		fmt.Fprintln(stderr, "goplus publish: Maven Central publication requires --target java")
		return 2
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 2
	}
	cfg, err := projectconfig.Load(cwd)
	if err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 2
	}
	m := cfg.Java.Maven
	if m.GroupID == "" {
		fmt.Fprintln(stderr, "goplus publish: configure [targets.java.maven] in goplus.toml")
		return 2
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	set, code := compileJava(cwd, cfg, patterns, false, stderr)
	if code != 0 {
		return code
	}
	if _, err := artifactio.Sync(cfg.Root, set, artifactio.Options{}); err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 2
	}
	tool, err := javatool.Resolve(context.Background(), cfg.Java.Release)
	if err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 1
	}
	if _, err = javatool.Build(context.Background(), tool, javatool.Config{Root: cfg.Root, Release: cfg.Java.Release, SourceDir: cfg.Java.SourceDir, RuntimeSourceDir: ".goplus/build/java/runtime-src", ClassDir: cfg.Java.ClassDir, Jar: cfg.Java.Jar, RuntimeJar: cfg.Java.RuntimeJar, ModuleName: set.ModuleName, ClasspathFiles: cfg.Java.ClasspathFiles, ModulepathFiles: cfg.Java.ModulepathFiles, StrongModule: cfg.Java.StrongModule, Bundle: cfg.Java.Bundle}, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 1
	}
	output := m.Bundle
	if output == "" {
		output = filepath.ToSlash(filepath.Join("dist", "central", m.ArtifactID+"-"+m.Version+"-bundle.zip"))
	}
	epoch, err := publicationEpoch(cfg.Root)
	if err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 1
	}
	keyPath := m.SigningKey
	if keyPath == "" {
		keyPath = strings.TrimSpace(os.Getenv("GOPLUS_MAVEN_SIGNING_KEY"))
	}
	key, err := mavencentral.EnsureSigningKey(keyPath, m.DeveloperName, m.DeveloperEmail, epoch)
	if err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "OpenPGP signing key: %s (%s)\n", key.Fingerprint, key.Path)
	result, err := mavencentral.BuildBundle(context.Background(), mavencentral.BundleOptions{Root: cfg.Root, Jar: cfg.Java.Jar, SourceDir: cfg.Java.SourceDir, RuntimeSourceDir: ".goplus/build/java/runtime-src", Output: output, Signer: key.Signer, Metadata: mavencentral.Metadata{GroupID: m.GroupID, ArtifactID: m.ArtifactID, Version: m.Version, Name: m.Name, Description: m.Description, URL: m.URL, LicenseName: m.LicenseName, LicenseURL: m.LicenseURL, DeveloperID: m.DeveloperID, DeveloperName: m.DeveloperName, DeveloperEmail: m.DeveloperEmail, DeveloperURL: m.DeveloperURL, SCMURL: m.SCMURL, SCMConnection: m.SCMConnection, SCMDeveloperConnection: m.SCMDeveloperConnection}})
	if err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 1
	}
	if err := mavencentral.InspectBundle(result.Path); err != nil {
		fmt.Fprintf(stderr, "goplus publish: invalid Central bundle: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Maven Central bundle: %s\n", result.Path)
	if *bundleOnly {
		return 0
	}
	credentials, err := mavencentral.LoadCredentials(cfg.Root)
	if err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 2
	}
	keyContext, keyCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if err := mavencentral.PublishPublicKey(keyContext, nil, "", key.Fingerprint, key.PublicArmor); err != nil {
		keyCancel()
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 1
	}
	keyCancel()
	fmt.Fprintln(stdout, "OpenPGP public key: published")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	client := mavencentral.Client{Username: credentials.Username, Password: credentials.Password}
	id, err := client.Upload(ctx, result.Path, m.GroupID+":"+m.ArtifactID+":"+m.Version, *automatic)
	if err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Central deployment: %s\n", id)
	deployment, err := client.Wait(ctx, id, 2*time.Second, !*automatic)
	if err != nil {
		fmt.Fprintf(stderr, "goplus publish: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Central state: %s\n", deployment.DeploymentState)
	for _, purl := range deployment.PURLs {
		fmt.Fprintln(stdout, purl)
	}
	if deployment.DeploymentState == "VALIDATED" {
		fmt.Fprintln(stdout, "validated; publish it in Central Portal or rerun with --automatic")
	}
	return 0
}

func publicationEpoch(root string) (time.Time, error) {
	if raw := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); raw != "" {
		seconds, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || seconds < 0 {
			return time.Time{}, fmt.Errorf("SOURCE_DATE_EPOCH must be a non-negative Unix timestamp")
		}
		return time.Unix(seconds, 0).UTC(), nil
	}
	command := exec.Command("git", "-C", root, "log", "-1", "--format=%ct")
	output, err := command.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("deterministic publication requires SOURCE_DATE_EPOCH or a Git commit: %w", err)
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("reading Git commit timestamp: %w", err)
	}
	return time.Unix(seconds, 0).UTC(), nil
}
