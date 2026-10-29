package update

import (
	"fmt"
	"sync"
	"time"
)

// JobRegistry tracks in-flight update jobs: at most one non-terminal job per (component, instance).
type JobRegistry struct {
	mu   sync.Mutex
	jobs map[string]*UpdateJob // key = component + "\x00" + instance
}

func NewJobRegistry() *JobRegistry {
	return &JobRegistry{jobs: make(map[string]*UpdateJob)}
}

func jobKey(c ComponentID, id InstanceID) string {
	return string(c) + "\x00" + string(id)
}

func isTerminal(s JobState) bool {
	switch s {
	case StateSucceeded, StateFailed, StateRolledBack, StateCancelled, StateUnsupported:
		return true
	default:
		return false
	}
}

// Begin registers a new job or returns the existing active one when idempotencyKey matches.
func (r *JobRegistry) Begin(j *UpdateJob) (*UpdateJob, error) {
	if j == nil {
		return nil, fmt.Errorf("update: nil job")
	}
	if !ValidComponent(j.Component) {
		return nil, fmt.Errorf("update: bad component")
	}
	if j.InstanceID == "" {
		return nil, fmt.Errorf("update: empty instance_id")
	}
	if j.JobID == "" {
		return nil, fmt.Errorf("update: empty job_id")
	}
	now := time.Now().Unix()
	r.mu.Lock()
	defer r.mu.Unlock()
	k := jobKey(j.Component, j.InstanceID)
	if cur, ok := r.jobs[k]; ok && !isTerminal(cur.State) {
		if j.IdempotencyKey != "" && j.IdempotencyKey == cur.IdempotencyKey {
			return cur, nil
		}
		return cur, fmt.Errorf("update: job already active for %s/%s state=%s job=%s", j.Component, j.InstanceID, cur.State, cur.JobID)
	}
	j.CreatedAtUnix = now
	j.UpdatedAtUnix = now
	if j.State == "" {
		j.State = StateQueued
	}
	cp := *j
	r.jobs[k] = &cp
	return r.jobs[k], nil
}

// Transition moves the active job for (component, instance) if legal.
func (r *JobRegistry) Transition(c ComponentID, id InstanceID, to JobState, errMsg string) (*UpdateJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := jobKey(c, id)
	cur, ok := r.jobs[k]
	if !ok {
		return nil, fmt.Errorf("update: no job for %s/%s", c, id)
	}
	if err := ValidateTransition(cur.State, to); err != nil {
		return cur, err
	}
	cur.State = to
	cur.UpdatedAtUnix = time.Now().Unix()
	if errMsg != "" {
		cur.LastError = errMsg
	}
	return cur, nil
}

// Get returns a copy of the job for (component, instance), if any.
func (r *JobRegistry) Get(c ComponentID, id InstanceID) (*UpdateJob, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.jobs[jobKey(c, id)]
	if !ok {
		return nil, false
	}
	cp := *cur
	return &cp, true
}
