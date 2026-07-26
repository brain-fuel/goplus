package config_test

import (
	"testing"

	"goforge.dev/goplus/std/config"
)

// stubLoader is a Loader backed by in-memory values + a settable fingerprint —
// enough to exercise the capability + watch laws without touching a filesystem.
type stubLoader struct {
	src    config.Source
	values map[string]any
	fp     config.Fingerprint
}

func (l *stubLoader) Provenance() config.Source        { return l.src }
func (l *stubLoader) Probe() (config.Fingerprint, error) { return l.fp, nil }
func (l *stubLoader) Load(cap config.Capability) (config.Loaded, error) {
	if !config.IsGranted(cap) {
		return config.Skipped{Reason: "denied", Fingerprint: l.fp}, nil
	}
	return config.Applied{
		Layer:       config.Layer{Source: l.src, Values: l.values},
		Fingerprint: l.fp,
	}, nil
}

const schema = 1

func grantAll(config.Source) config.Capability { return config.Granted{} }

func TestLoadAllResolvesByPrecedence(t *testing.T) {
	file := &stubLoader{src: config.FileSource{}, values: map[string]any{"port": 80}, fp: config.Fingerprint{Token: "f1", Exists: true}}
	env := &stubLoader{src: config.EnvironmentSource{}, values: map[string]any{"port": 90}, fp: config.Fingerprint{Token: "e1", Exists: true}}

	layers, prints, errs := config.LoadAll(grantAll, file, env)
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if len(prints) != 2 {
		t.Fatalf("want 2 fingerprints, got %d", len(prints))
	}
	snap := config.Resolve(schema, layers...)
	key := config.NewKey[int](schema, "port", func(v any) (int, bool) { i, ok := v.(int); return i, ok })
	// environment outranks file → 90 wins.
	if got := config.Get(snap, key); mustFound(t, got) != 90 {
		t.Fatalf("precedence: got %d, want 90", mustFound(t, got))
	}
}

func TestDeniedCapabilitySkipsSource(t *testing.T) {
	file := &stubLoader{src: config.FileSource{}, values: map[string]any{"port": 80}, fp: config.Fingerprint{Token: "f1", Exists: true}}
	// deny the file → it contributes no layer; resolution falls back to default.
	deny := func(s config.Source) config.Capability {
		if config.SourceEqual(s, config.FileSource{}) {
			return config.Denied{Reason: "blocked"}
		}
		return config.Granted{}
	}
	def := &stubLoader{src: config.DefaultSource{}, values: map[string]any{"port": 8080}, fp: config.Fingerprint{Token: "d", Exists: true}}
	layers, _, _ := config.LoadAll(deny, def, file)
	if len(layers) != 1 {
		t.Fatalf("denied file must not contribute a layer; got %d layers", len(layers))
	}
	snap := config.Resolve(schema, layers...)
	key := config.NewKey[int](schema, "port", func(v any) (int, bool) { i, ok := v.(int); return i, ok })
	if got := mustFound(t, config.Get(snap, key)); got != 8080 {
		t.Fatalf("denied fallback: got %d, want 8080 (default)", got)
	}
}

func TestReloadDetectsChange(t *testing.T) {
	file := &stubLoader{src: config.FileSource{}, fp: config.Fingerprint{Token: "v1", Exists: true}}
	_, prints, _ := config.LoadAll(grantAll, file)
	// unchanged → no reload
	changed, now, _ := config.Reload(prints, file)
	if len(changed) != 0 {
		t.Fatalf("unchanged source must not reload; got %v", changed)
	}
	// change the fingerprint → reload of exactly that source
	file.fp = config.Fingerprint{Token: "v2", Exists: true}
	changed2, _, _ := config.Reload(now, file)
	if len(changed2) != 1 || !config.SourceEqual(changed2[0], config.FileSource{}) {
		t.Fatalf("changed source must reload; got %v", changed2)
	}
}

func TestFingerprintEqual(t *testing.T) {
	a := config.Fingerprint{Token: "x", Exists: true}
	if !config.FingerprintEqual(a, a) {
		t.Fatal("reflexive")
	}
	if config.FingerprintEqual(a, config.Fingerprint{Token: "x", Exists: false}) {
		t.Fatal("existence matters")
	}
	if config.FingerprintEqual(a, config.Fingerprint{Token: "y", Exists: true}) {
		t.Fatal("token matters")
	}
}

func mustFound(t *testing.T, l config.Lookup[int]) int {
	t.Helper()
	f, ok := l.(config.Found[int])
	if !ok {
		t.Fatalf("expected Found, got %#v", l)
	}
	return f.Value
}
