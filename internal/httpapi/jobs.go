package httpapi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobAnalyzing JobStatus = "analyzing"
	JobLoading   JobStatus = "loading"
	JobComplete  JobStatus = "complete"
	JobFailed    JobStatus = "failed"
)

type JobProgress struct {
	Phase   string `json:"phase"`
	Percent int    `json:"percent"`
	Message string `json:"message"`
}

type Job struct {
	ID          string      `json:"id"`
	Status      JobStatus   `json:"status"`
	RepoPath    string      `json:"repoPath,omitempty"`
	RepoName    string      `json:"repoName,omitempty"`
	Progress    JobProgress `json:"progress"`
	Error       string      `json:"error,omitempty"`
	StartedAt   int64       `json:"startedAt"`
	CompletedAt int64       `json:"completedAt,omitempty"`
}

const jobTTL = time.Hour

type JobManager struct {
	mu      sync.Mutex
	nextID  int64
	jobs    map[string]*Job
	cancels map[string]context.CancelFunc
	now     func() time.Time
}

func NewJobManager() *JobManager {
	return &JobManager{
		jobs:    map[string]*Job{},
		cancels: map[string]context.CancelFunc{},
		now:     time.Now,
	}
}

func (m *JobManager) Create(repoPath string, repoName string) (Job, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	m.cleanupLocked(now)

	for _, job := range m.jobs {
		if !isTerminalJobStatus(job.Status) && repo.SamePath(job.RepoPath, repoPath) {
			return *job, false, nil
		}
	}

	for _, job := range m.jobs {
		if !isTerminalJobStatus(job.Status) {
			return Job{}, false, fmt.Errorf("Analysis already in progress (job %s)", job.ID)
		}
	}

	m.nextID++
	job := &Job{
		ID:        fmt.Sprintf("job-%d", m.nextID),
		Status:    JobQueued,
		RepoPath:  repoPath,
		RepoName:  repoName,
		StartedAt: unixMilli(now),
		Progress:  JobProgress{Phase: "queued", Percent: 0, Message: "Waiting to start..."},
	}
	m.jobs[job.ID] = job
	return *job, true, nil
}

func (m *JobManager) RegisterCancel(id string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel == nil {
		delete(m.cancels, id)
		return
	}
	m.cancels[id] = cancel
}

func (m *JobManager) ClearCancel(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cancels, id)
}

func (m *JobManager) Get(id string) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(m.now())
	job, ok := m.jobs[id]
	if !ok {
		return Job{}, false
	}
	return *job, true
}

func (m *JobManager) List() []Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(m.now())
	jobs := make([]Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, *job)
	}
	return jobs
}

func (m *JobManager) UpdateProgress(id string, progress JobProgress) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok || isTerminalJobStatus(job.Status) {
		return
	}
	job.Progress = progress
	if progress.Phase == string(JobLoading) {
		job.Status = JobLoading
	} else {
		job.Status = JobAnalyzing
	}
}

func (m *JobManager) Complete(id string) {
	m.CompleteWithResult(id, "", "")
}

func (m *JobManager) CompleteWithResult(id string, repoPath string, repoName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok || isTerminalJobStatus(job.Status) {
		return
	}
	if repoPath != "" {
		job.RepoPath = repoPath
	}
	if repoName != "" {
		job.RepoName = repoName
	}
	job.Status = JobComplete
	job.Progress = JobProgress{Phase: string(JobComplete), Percent: 100, Message: "Complete"}
	job.CompletedAt = unixMilli(m.now())
}

func (m *JobManager) Fail(id string, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok || isTerminalJobStatus(job.Status) {
		return
	}
	job.Status = JobFailed
	job.Error = message
	job.Progress = JobProgress{Phase: string(JobFailed), Percent: job.Progress.Percent, Message: message}
	job.CompletedAt = unixMilli(m.now())
}

func (m *JobManager) Cancel(id string, reason string) bool {
	m.mu.Lock()
	cancel := m.cancels[id]
	job, ok := m.jobs[id]
	if !ok || isTerminalJobStatus(job.Status) {
		m.mu.Unlock()
		return false
	}
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if reason == "" {
		reason = "Cancelled by user"
	}
	m.Fail(id, reason)
	return true
}

func (m *JobManager) cleanupLocked(now time.Time) {
	for id, job := range m.jobs {
		if !isTerminalJobStatus(job.Status) || job.CompletedAt == 0 {
			continue
		}
		completedAt := time.Unix(0, job.CompletedAt*int64(time.Millisecond))
		if now.Sub(completedAt) > jobTTL {
			delete(m.jobs, id)
		}
	}
}

func isTerminalJobStatus(status JobStatus) bool {
	return status == JobComplete || status == JobFailed
}

func unixMilli(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
