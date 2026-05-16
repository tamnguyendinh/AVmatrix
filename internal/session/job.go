package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type JobStatus string

const (
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
)

type Job struct {
	ID          string
	Provider    Provider
	RepoName    string
	RepoPath    string
	StartedAt   int64
	CompletedAt int64
	Status      JobStatus
	Error       string

	ctx    context.Context
	cancel context.CancelCauseFunc

	mu      sync.Mutex
	history []Event
	subs    map[chan Event]struct{}
}

func NewJob(provider Provider, repoName string, repoPath string) *Job {
	ctx, cancel := context.WithCancelCause(context.Background())
	return &Job{
		ID:        newID(),
		Provider:  provider,
		RepoName:  repoName,
		RepoPath:  repoPath,
		StartedAt: time.Now().UnixMilli(),
		Status:    JobRunning,
		ctx:       ctx,
		cancel:    cancel,
		subs:      map[chan Event]struct{}{},
	}
}

func (j *Job) Context() context.Context {
	return j.ctx
}

func (j *Job) Emit(event Event) {
	j.mu.Lock()
	if event.SessionID == "" {
		event.SessionID = j.ID
	}
	if event.Provider == "" {
		event.Provider = j.Provider
	}
	if event.RepoName == "" {
		event.RepoName = j.RepoName
	}
	if event.RepoPath == "" {
		event.RepoPath = j.RepoPath
	}
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}

	j.history = append(j.history, event)
	terminal := false
	switch event.Type {
	case "done":
		j.Status = JobCompleted
		j.CompletedAt = time.Now().UnixMilli()
		terminal = true
	case "error":
		j.Status = JobFailed
		j.Error = event.Error
		j.CompletedAt = time.Now().UnixMilli()
		terminal = true
	case "cancelled":
		j.Status = JobCancelled
		j.Error = event.Reason
		j.CompletedAt = time.Now().UnixMilli()
		terminal = true
	}

	for sub := range j.subs {
		sub <- event
		if terminal {
			close(sub)
			delete(j.subs, sub)
		}
	}
	j.mu.Unlock()
}

func (j *Job) Subscribe(replay bool) (<-chan Event, func()) {
	ch := make(chan Event, 256)

	j.mu.Lock()
	if replay {
		for _, event := range j.history {
			ch <- event
		}
	}
	if j.Status == JobRunning {
		j.subs[ch] = struct{}{}
	} else {
		close(ch)
	}
	j.mu.Unlock()

	unsubscribe := func() {
		j.mu.Lock()
		if _, ok := j.subs[ch]; ok {
			delete(j.subs, ch)
			close(ch)
		}
		j.mu.Unlock()
	}
	return ch, unsubscribe
}

func (j *Job) Cancel(reason string) {
	if reason == "" {
		reason = "Cancelled by user"
	}
	j.mu.Lock()
	running := j.Status == JobRunning
	j.mu.Unlock()
	if running {
		j.cancel(cancelReason(reason))
	}
}

type cancelReason string

func (r cancelReason) Error() string {
	return string(r)
}

func newID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
}
