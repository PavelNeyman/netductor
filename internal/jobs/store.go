// Package jobs is the shared thin-UI job store (operator serve + primary API).
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	StatusQueued      Status = "queued"
	StatusRunning     Status = "running"
	StatusSucceeded   Status = "succeeded"
	StatusFailed      Status = "failed"
	StatusCancelled   Status = "cancelled"
	StatusInterrupted Status = "interrupted"
)

// Step is a stable progress marker (same idea as operator.Reporter).
type Step struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Done    bool   `json:"done"`
	Err     string `json:"error,omitempty"`
}

// Job is the durable record shown to all thin UIs.
type Job struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Status    Status          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Steps     []Step          `json:"steps"`
	Log       string          `json:"log"` // ring-trimmed
	Result    json.RawMessage `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
	// Params kept server-side for workers; redacted on public GET if needed.
	Params map[string]any `json:"-"`

	cancel context.CancelFunc
	mu     sync.Mutex
}

// Public copies without internal fields.
func (j *Job) Snapshot() map[string]any {
	j.mu.Lock()
	defer j.mu.Unlock()
	steps := append([]Step(nil), j.Steps...)
	log := j.Log
	if len(log) > 64<<10 {
		log = log[len(log)-(64<<10):]
	}
	return map[string]any{
		"id":         j.ID,
		"type":       j.Type,
		"status":     j.Status,
		"created_at": j.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at": j.UpdatedAt.UTC().Format(time.RFC3339),
		"steps":      steps,
		"log":        log,
		"result":     j.Result,
		"error":      j.Error,
	}
}

type Store struct {
	mu      sync.Mutex
	jobs    map[string]*Job
	dir     string
	maxLog  int
	runners map[string]Runner // type → runner
}

// Runner executes a job. It must respect ctx cancel.
type Runner func(ctx context.Context, j *Job, appendLog func(string), setStep func(Step)) (result any, err error)

func NewStore(dir string) *Store {
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".cache", "netductor", "jobs")
	}
	_ = os.MkdirAll(dir, 0o700)
	s := &Store{
		jobs:    map[string]*Job{},
		dir:     dir,
		maxLog:  256 << 10,
		runners: map[string]Runner{},
	}
	s.loadDisk()
	return s
}

func (s *Store) Register(typ string, r Runner) {
	s.mu.Lock()
	s.runners[typ] = r
	s.mu.Unlock()
}

func (s *Store) Dir() string { return s.dir }

func newID() string {
	return fmt.Sprintf("j_%d", time.Now().UnixNano())
}

// Start creates a queued job and runs it in a goroutine.
func (s *Store) Start(typ string, params map[string]any) (*Job, error) {
	s.mu.Lock()
	r, ok := s.runners[typ]
	s.mu.Unlock()
	if !ok || r == nil {
		return nil, fmt.Errorf("unknown job type %q", typ)
	}
	ctx, cancel := context.WithCancel(context.Background())
	j := &Job{
		ID:        newID(),
		Type:      typ,
		Status:    StatusQueued,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Params:    params,
		cancel:    cancel,
	}
	s.mu.Lock()
	s.jobs[j.ID] = j
	s.mu.Unlock()
	s.persist(j)

	go s.run(ctx, j, r)
	return j, nil
}

func (s *Store) run(ctx context.Context, j *Job, r Runner) {
	j.mu.Lock()
	j.Status = StatusRunning
	j.UpdatedAt = time.Now().UTC()
	j.mu.Unlock()
	s.persist(j)

	appendLog := func(line string) {
		j.mu.Lock()
		if j.Log != "" && !strings.HasSuffix(j.Log, "\n") {
			j.Log += "\n"
		}
		j.Log += line
		if !strings.HasSuffix(line, "\n") {
			j.Log += "\n"
		}
		if len(j.Log) > s.maxLog {
			j.Log = j.Log[len(j.Log)-s.maxLog:]
		}
		j.UpdatedAt = time.Now().UTC()
		j.mu.Unlock()
		s.persist(j)
	}
	setStep := func(st Step) {
		j.mu.Lock()
		found := false
		for i := range j.Steps {
			if j.Steps[i].ID == st.ID {
				j.Steps[i] = st
				found = true
				break
			}
		}
		if !found {
			j.Steps = append(j.Steps, st)
		}
		j.UpdatedAt = time.Now().UTC()
		j.mu.Unlock()
		s.persist(j)
		if st.Message != "" {
			appendLog(fmt.Sprintf("step %s: %s", st.ID, st.Message))
		}
	}

	result, err := r(ctx, j, appendLog, setStep)

	if ctx.Err() != nil {
		j.mu.Lock()
		j.Status = StatusCancelled
		j.Error = "cancelled"
		j.UpdatedAt = time.Now().UTC()
		j.mu.Unlock()
		appendLog("cancelled")
		s.persist(j)
		return
	}
	if err != nil {
		appendLog("ERROR: " + err.Error())
		j.mu.Lock()
		j.Status = StatusFailed
		j.Error = err.Error()
		j.UpdatedAt = time.Now().UTC()
		j.mu.Unlock()
		s.persist(j)
		return
	}
	j.mu.Lock()
	j.Status = StatusSucceeded
	j.UpdatedAt = time.Now().UTC()
	if result != nil {
		b, _ := json.Marshal(result)
		j.Result = b
	}
	j.mu.Unlock()
	s.persist(j)
}

func (s *Store) Get(id string) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[id]
}

func (s *Store) Cancel(id string) error {
	s.mu.Lock()
	j := s.jobs[id]
	s.mu.Unlock()
	if j == nil {
		return fmt.Errorf("job not found")
	}
	j.mu.Lock()
	if j.Status == StatusSucceeded || j.Status == StatusFailed || j.Status == StatusCancelled {
		j.mu.Unlock()
		return fmt.Errorf("job already finished")
	}
	j.Status = StatusCancelled
	c := j.cancel
	j.mu.Unlock()
	if c != nil {
		c()
	}
	s.persist(j)
	return nil
}

// Counts returns running and failed job totals (for metrics).
func (s *Store) Counts() (running, failed int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		j.mu.Lock()
		st := j.Status
		j.mu.Unlock()
		switch st {
		case StatusRunning, StatusQueued:
			running++
		case StatusFailed:
			failed++
		}
	}
	return
}

func (s *Store) List(status Status, typ string) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.jobs))
	for id := range s.jobs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return s.jobs[ids[i]].CreatedAt.After(s.jobs[ids[j]].CreatedAt)
	})
	var out []map[string]any
	for _, id := range ids {
		j := s.jobs[id]
		j.mu.Lock()
		st, ty := j.Status, j.Type
		j.mu.Unlock()
		if status != "" && st != status {
			continue
		}
		if typ != "" && ty != typ {
			continue
		}
		out = append(out, j.Snapshot())
	}
	return out
}

func (s *Store) persist(j *Job) {
	if s.dir == "" {
		return
	}
	snap := j.Snapshot()
	b, _ := json.MarshalIndent(snap, "", "  ")
	_ = os.WriteFile(filepath.Join(s.dir, j.ID+".json"), b, 0o600)
}

func (s *Store) loadDisk() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var m map[string]any
		if json.Unmarshal(b, &m) != nil {
			continue
		}
		id, _ := m["id"].(string)
		if id == "" {
			continue
		}
		st, _ := m["status"].(string)
		// Mark previously running as interrupted
		if st == string(StatusRunning) || st == string(StatusQueued) {
			st = string(StatusInterrupted)
			m["status"] = st
			m["error"] = "interrupted (process restart)"
		}
		j := &Job{
			ID:     id,
			Type:   fmt.Sprint(m["type"]),
			Status: Status(st),
			Error:  fmt.Sprint(m["error"]),
		}
		if t, err := time.Parse(time.RFC3339, fmt.Sprint(m["created_at"])); err == nil {
			j.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, fmt.Sprint(m["updated_at"])); err == nil {
			j.UpdatedAt = t
		}
		if log, ok := m["log"].(string); ok {
			j.Log = log
		}
		s.jobs[id] = j
	}
}
