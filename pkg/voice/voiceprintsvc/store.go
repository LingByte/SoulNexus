package voiceprintsvc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Store persists enrolled WAV samples on disk for the embedded HTTP microservice.
type Store struct {
	root string
	mu   sync.RWMutex
}

func NewStore(root string) (*Store, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "./data/voiceprints"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create voiceprint data dir: %w", err)
	}
	return &Store{root: root}, nil
}

func (s *Store) wavPath(agentID, speakerID string) (string, error) {
	agentID = sanitizePathPart(agentID)
	speakerID = sanitizePathPart(speakerID)
	if agentID == "" || speakerID == "" {
		return "", fmt.Errorf("agent_id and speaker_id are required")
	}
	return filepath.Join(s.root, agentID, speakerID+".wav"), nil
}

func (s *Store) Save(agentID, speakerID string, wav []byte) error {
	path, err := s.wavPath(agentID, speakerID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, wav, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) Load(agentID, speakerID string) ([]byte, error) {
	path, err := s.wavPath(agentID, speakerID)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return os.ReadFile(path)
}

func (s *Store) Delete(agentID, speakerID string) error {
	path, err := s.wavPath(agentID, speakerID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Store) CountAll() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := 0
	err := filepath.WalkDir(s.root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".wav" {
			return nil
		}
		total++
		return nil
	})
	return total, err
}

func sanitizePathPart(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "._")
	if out == "" || out == "." || out == ".." {
		return ""
	}
	return out
}
