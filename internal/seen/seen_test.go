package seen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenMissingFile(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "seen.json"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !s.IsEmpty() {
		t.Error("expected empty store")
	}
}

func TestSaveAndReopen(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "seen.json")
	s, err := Open(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	s.Add("a")
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.Has("a") || s2.Has("b") {
		t.Error("reopened store has wrong contents")
	}
}

func TestPrunesExpired(t *testing.T) {
	p := filepath.Join(t.TempDir(), "seen.json")
	data, _ := json.Marshal(map[string]time.Time{
		"old": time.Now().UTC().Add(-2 * time.Hour),
		"new": time.Now().UTC(),
	})
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := Open(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if s.Has("old") {
		t.Error("expired entry should be pruned")
	}
	if !s.Has("new") {
		t.Error("fresh entry should be kept")
	}
}

func TestOpenCorruptFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "seen.json")
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p, time.Hour); err == nil {
		t.Error("expected error for corrupt file")
	}
}
