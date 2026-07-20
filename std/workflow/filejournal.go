package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"goforge.dev/goplus/std/fsatomic"
)

// FileJournal stores one crash-safe JSON record per workflow. Names are
// derived from IDs only after rejecting path separators.
type FileJournal struct {
	Dir  string
	Perm os.FileMode
}

func (j FileJournal) path(id string) (string, error) {
	if id == "" || filepath.Base(id) != id || id == "." || id == ".." {
		return "", errors.New("workflow: invalid journal ID")
	}
	return filepath.Join(j.Dir, id+".json"), nil
}

func (j FileJournal) Load(_ context.Context, id string) (Record, bool, error) {
	p, err := j.path(id)
	if err != nil {
		return Record{}, false, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	var r Record
	if err := json.Unmarshal(b, &r); err != nil {
		return Record{}, false, err
	}
	return r, true, nil
}

func (j FileJournal) Save(_ context.Context, r Record) error {
	p, err := j.path(r.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(j.Dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	perm := j.Perm
	if perm == 0 {
		perm = 0o600
	}
	return fsatomic.WriteFile(p, b, perm)
}

func (j FileJournal) Delete(_ context.Context, id string) error {
	p, err := j.path(id)
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
