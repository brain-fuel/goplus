// Package artifactio safely materializes compiler artifact sets.
package artifactio

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"goforge.dev/goplus/compiler"
)

// Options controls filesystem synchronization.
type Options struct {
	Check                bool
	Stage                bool
	IgnoreRuntimeInCheck bool
}

// Result reports paths relative to root.
type Result struct {
	Written []string
	Stale   []string
	Orphans []string
}

// Sync writes one artifact set and removes only outputs owned by its previous
// manifest. Every output must remain beneath root.
func Sync(root string, set compiler.ArtifactSet, opts Options) (Result, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return Result{}, fmt.Errorf("resolving artifact root: %w", err)
	}
	outputs := map[string][]byte{}
	roles := map[string]compiler.ArtifactRole{}
	manifestPath := ""
	for _, artifact := range set.Artifacts {
		path, err := resolve(abs, artifact.Path)
		if err != nil {
			return Result{}, err
		}
		if _, duplicate := outputs[path]; duplicate {
			return Result{}, fmt.Errorf("artifact path %q is listed twice", artifact.Path)
		}
		outputs[path] = artifact.Data
		roles[path] = artifact.Role
		if artifact.Role == compiler.ArtifactManifest {
			manifestPath = path
		}
	}
	owned, err := previousOwned(abs, manifestPath)
	if err != nil {
		return Result{}, err
	}
	var result Result
	var touched []string
	paths := sortedPaths(outputs)
	for _, path := range paths {
		if opts.Check && opts.IgnoreRuntimeInCheck && roles[path] == compiler.ArtifactRuntime {
			continue
		}
		existing, readErr := os.ReadFile(path)
		stale := readErr != nil || string(existing) != string(outputs[path])
		if !stale {
			continue
		}
		rel, _ := filepath.Rel(abs, path)
		rel = filepath.ToSlash(rel)
		if opts.Check {
			result.Stale = append(result.Stale, rel)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return Result{}, err
		}
		if err := os.WriteFile(path, outputs[path], 0o644); err != nil {
			return Result{}, err
		}
		result.Written = append(result.Written, rel)
		if roles[path] != compiler.ArtifactRuntime {
			touched = append(touched, path)
		}
	}
	for _, path := range owned {
		if _, retained := outputs[path]; retained {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return Result{}, err
		}
		rel, _ := filepath.Rel(abs, path)
		rel = filepath.ToSlash(rel)
		result.Orphans = append(result.Orphans, rel)
		if opts.Check {
			result.Stale = append(result.Stale, rel)
			continue
		}
		if err := os.Remove(path); err != nil {
			return Result{}, err
		}
		result.Written = append(result.Written, rel)
		if opts.Stage && tracked(abs, path) {
			touched = append(touched, path)
		}
	}
	sort.Strings(result.Written)
	sort.Strings(result.Stale)
	sort.Strings(result.Orphans)
	if opts.Stage && len(touched) > 0 {
		args := append([]string{"-C", abs, "add", "--"}, touched...)
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return Result{}, fmt.Errorf("git add: %v\n%s", err, out)
		}
	}
	return result, nil
}

func tracked(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return exec.Command("git", "-C", root, "ls-files", "--error-unmatch", "--", rel).Run() == nil
}

func previousOwned(root, manifestPath string) ([]string, error) {
	if manifestPath == "" {
		return nil, nil
	}
	data, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Artifacts []string `json:"artifacts"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("reading previous artifact manifest %s: %w", manifestPath, err)
	}
	var paths []string
	for _, rel := range manifest.Artifacts {
		path, err := resolve(root, rel)
		if err != nil {
			return nil, fmt.Errorf("previous artifact manifest: %w", err)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func resolve(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("artifact path %q must be relative", rel)
	}
	path := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("artifact path %q escapes module root", rel)
	}
	if err := rejectSymlinkComponents(root, path); err != nil {
		return "", fmt.Errorf("artifact path %q: %w", rel, err)
	}
	return path, nil
}

func rejectSymlinkComponents(root, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuses symbolic-link component %s", current)
		}
	}
	return nil
}

func sortedPaths(values map[string][]byte) []string {
	paths := make([]string, 0, len(values))
	for path := range values {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
