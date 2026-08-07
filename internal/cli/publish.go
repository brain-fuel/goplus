package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	automatic := fs.Bool("automatic", false, "publish automatically after Central validation")
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
	result, err := mavencentral.BuildBundle(context.Background(), mavencentral.BundleOptions{Root: cfg.Root, Jar: cfg.Java.Jar, SourceDir: cfg.Java.SourceDir, RuntimeSourceDir: ".goplus/build/java/runtime-src", Output: output, Signer: mavencentral.GPGSigner{Key: m.GPGKey}, Metadata: mavencentral.Metadata{GroupID: m.GroupID, ArtifactID: m.ArtifactID, Version: m.Version, Name: m.Name, Description: m.Description, URL: m.URL, LicenseName: m.LicenseName, LicenseURL: m.LicenseURL, DeveloperID: m.DeveloperID, DeveloperName: m.DeveloperName, DeveloperEmail: m.DeveloperEmail, DeveloperURL: m.DeveloperURL, SCMURL: m.SCMURL, SCMConnection: m.SCMConnection, SCMDeveloperConnection: m.SCMDeveloperConnection}})
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
	username, password := strings.TrimSpace(os.Getenv("MAVEN_CENTRAL_USERNAME")), strings.TrimSpace(os.Getenv("MAVEN_CENTRAL_PASSWORD"))
	if username == "" || password == "" {
		fmt.Fprintln(stderr, "goplus publish: set MAVEN_CENTRAL_USERNAME and MAVEN_CENTRAL_PASSWORD to a Central Portal user token")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	client := mavencentral.Client{Username: username, Password: password}
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
