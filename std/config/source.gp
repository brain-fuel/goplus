// Capability-scoped source loading and watch/reload for config.
//
// The resolution core (config.gp) takes pre-materialized Layers; this file adds
// the missing half — LOADING a Layer from a backing (file, environment,
// directory) under an explicit effect permission, and detecting when a source
// has CHANGED so the snapshot can be reloaded. The effect of touching a source
// lives entirely behind Probe/Load: a caller stays pure until it presents a
// granting Capability.
package config

// Capability authorizes reading a source — the effect permission that gates a
// Load. Granted permits the read; Denied carries the reason and makes Load skip
// the source. This is where a consumer's access policy lives: a file-based
// loader grants when the file exists and is readable; a per-directory loader
// grants when the directory's config has been allowed and denies when blocked.
type Capability enum {
	Granted()
	Denied(Reason string)
}

// IsGranted reports whether a capability permits the load.
func IsGranted(c Capability) bool {
	match c {
	case Granted():
		return true
	case Denied(_):
		return false
	}
}

// Fingerprint is a source's change-detection token: an opaque content/mtime
// token plus whether the source currently exists. Reload compares fingerprints;
// equal fingerprints mean the resolved snapshot is still current for that
// source.
type Fingerprint struct {
	Token  string
	Exists bool
}

// FingerprintEqual reports whether two fingerprints denote no change.
func FingerprintEqual(a, b Fingerprint) bool {
	return a.Exists == b.Exists && a.Token == b.Token
}

// Loaded is the outcome of a Load: the source either contributed a Layer or was
// Skipped (denied capability, or absent), always paired with the fingerprint
// observed at load time so the caller can seed a watch.
type Loaded enum {
	Skipped(Reason string, Fingerprint Fingerprint)
	Applied(Layer Layer, Fingerprint Fingerprint)
}

// Loader reads exactly one configuration source. Provenance fixes the Source
// (and thus the resolution precedence and each Entry's provenance). Probe
// returns the current fingerprint without materializing values (for watch);
// Load reads the values into a Layer, gated by a Capability.
type Loader interface {
	Provenance() Source
	Probe() (Fingerprint, error)
	Load(cap Capability) (Loaded, error)
}

// sourceName is the lowercase provenance label used in load-error field paths.
func sourceName(s Source) string {
	match s {
	case DefaultSource():
		return "default"
	case RemoteSource():
		return "remote"
	case FileSource():
		return "file"
	case EnvironmentSource():
		return "environment"
	case FlagSource():
		return "flag"
	case OverrideSource():
		return "override"
	}
}

// LoadAll loads every loader in caller order, gating each with grant(source).
// It returns the produced layers (pass to Resolve), each source's fingerprint
// (seed a watch), and per-source load errors (path-tagged by provenance). A
// denied or absent source contributes no layer, so resolution falls back to
// lower-precedence values — capability-scoped loading is failure-tolerant by
// construction.
func LoadAll(grant func(Source) Capability, loaders ...Loader) (layers []Layer, prints map[Source]Fingerprint, errs []error) {
	prints = make(map[Source]Fingerprint)
	for _, loader := range loaders {
		source := loader.Provenance()
		loaded, err := loader.Load(grant(source))
		if err != nil {
			errs = append(errs, At(sourceName(source), err))
		}
		match loaded {
		case Applied(layer, fp):
			layers = append(layers, layer)
			prints[source] = fp
		case Skipped(_, fp):
			prints[source] = fp
		}
	}
	return
}

// Reload re-probes every loader and reports which sources changed since prev.
// An empty changed set means the resolved snapshot is still valid; otherwise the
// caller re-runs LoadAll + Resolve for a fresh snapshot. This is the single
// watch/reload law shared by a filesystem notifier and an mtime ledger: a source
// reloads exactly when its fingerprint differs from the one last observed.
func Reload(prev map[Source]Fingerprint, loaders ...Loader) (changed []Source, now map[Source]Fingerprint, errs []error) {
	now = make(map[Source]Fingerprint)
	for _, loader := range loaders {
		source := loader.Provenance()
		fp, err := loader.Probe()
		if err != nil {
			errs = append(errs, At(sourceName(source), err))
			continue
		}
		now[source] = fp
		old, seen := prev[source]
		if !seen || !FingerprintEqual(old, fp) {
			changed = append(changed, source)
		}
	}
	return
}
