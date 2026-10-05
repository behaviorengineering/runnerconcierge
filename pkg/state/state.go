package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const fileName = "state.json"

// CheckpointStageDone marks a finished setup (verify passed, runner online).
const CheckpointStageDone = "done"

// Checkpoint is redacted wizard progress (no tokens or passwords).
type Checkpoint struct {
	Version     int       `json:"version"`
	Fingerprint string    `json:"fingerprint"`
	Stage       string    `json:"stage"`
	Completed   []string  `json:"completed"`
	RunnerID    int       `json:"runner_id"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Executor    string    `json:"executor"`
	GitLabURL   string    `json:"gitlab_url"`
	RepoPath    string    `json:"repo_path"`
	GroupPath   string    `json:"group_path,omitempty"`
	ConfigPath  string    `json:"config_path"`
	BinaryPath  string    `json:"binary_path"`
	ServiceKind string    `json:"service_kind"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Store persists checkpoints under a config directory.
type Store struct {
	Dir string
}

// Path returns the checkpoint file path.
func (s *Store) Path() string {
	return filepath.Join(s.Dir, fileName)
}

// Load reads a checkpoint if present.
func (s *Store) Load() (*Checkpoint, bool, error) {
	if s == nil || s.Dir == "" {
		return nil, false, nil
	}
	path := s.Path()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, false, err
	}
	return &cp, true, nil
}

// Save writes the checkpoint atomically.
func (s *Store) Save(cp *Checkpoint) error {
	if s == nil || s.Dir == "" {
		return nil
	}
	if cp == nil {
		return nil
	}
	cp.UpdatedAt = time.Now().UTC()
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path())
}

// Archive moves state.json to state.json.bak.
func (s *Store) Archive() error {
	if s == nil {
		return nil
	}
	src := s.Path()
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.Rename(src, src+".bak")
}
