package seen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Seen struct {
	path string
	ttl  time.Duration

	mu  sync.Mutex
	ids map[string]time.Time
}

func Open(path string, ttl time.Duration) (*Seen, error) {
	s := &Seen{
		path: path,
		ttl:  ttl,
		ids:  make(map[string]time.Time),
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read seen store: %w", err)
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s.ids); err != nil {
		return nil, fmt.Errorf("parse seen store: %w", err)
	}

	s.prune()
	return s, nil
}

func (s *Seen) IsEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.ids) == 0
}

func (s *Seen) Has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.ids[id]
	return ok
}

func (s *Seen) Add(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids[id] = time.Now().UTC()
}

func (s *Seen) AddAll(ids []string) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		s.ids[id] = now
	}
}

func (s *Seen) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.ids)
}

func (s *Seen) Save() error {
	s.mu.Lock()
	s.pruneLocked()
	data, err := json.Marshal(s.ids)
	s.mu.Unlock()

	if err != nil {
		return fmt.Errorf("encode seen store: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create store dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".seen-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp store: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp store: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp store: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace store: %w", err)
	}
	return nil
}

func (s *Seen) prune() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked()
}

func (s *Seen) pruneLocked() {
	if s.ttl <= 0 {
		return
	}
	cutoff := time.Now().UTC().Add(-s.ttl)
	for id, at := range s.ids {
		if at.Before(cutoff) {
			delete(s.ids, id)
		}
	}
}
