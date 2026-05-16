package session

import (
	"context"
	"sync"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/repo"
)

type ChatContext struct {
	Repo ResolvedRepo
}

type Adapter interface {
	Provider() Provider
	ExecutionMode() ExecutionMode
	RuntimeEnvironment() RuntimeEnvironment
	GetStatus(ctx context.Context) (Status, error)
	RunChat(ctx context.Context, job *Job, request ChatRequest, chatContext ChatContext) error
}

type Controller struct {
	adapter  Adapter
	resolver RepoResolver

	mu                 sync.Mutex
	jobs               map[string]*Job
	activeRepoSessions map[string]string
}

func NewController(adapter Adapter, store repo.Store) *Controller {
	if adapter == nil {
		adapter = NewCodexAdapter(CodexAdapterOptions{})
	}
	return &Controller{
		adapter:            adapter,
		resolver:           NewStoreResolver(store),
		jobs:               map[string]*Job{},
		activeRepoSessions: map[string]string{},
	}
}

func NewControllerWithResolver(adapter Adapter, resolver RepoResolver) *Controller {
	return &Controller{
		adapter:            adapter,
		resolver:           resolver,
		jobs:               map[string]*Job{},
		activeRepoSessions: map[string]string{},
	}
}

func (c *Controller) GetStatus(ctx context.Context, binding RepoBinding) (Status, error) {
	status, err := c.adapter.GetStatus(ctx)
	if err != nil {
		return Status{}, err
	}
	if binding.RepoName == "" && binding.RepoPath == "" {
		return status, nil
	}

	resolved, err := c.resolver.Resolve(binding)
	if err == nil {
		state := "index_required"
		if resolved.Indexed {
			state = "indexed"
		}
		status.Repo = &RepoResolution{
			RepoName:         binding.RepoName,
			RepoPath:         binding.RepoPath,
			State:            state,
			ResolvedRepoName: resolved.RepoName,
			ResolvedRepoPath: resolved.RepoPath,
		}
		return status, nil
	}

	if runtimeErr, ok := err.(*RuntimeError); ok {
		state := "invalid"
		switch runtimeErr.Code {
		case ErrorRepoNotFound:
			state = "not_found"
		case ErrorIndexRequired:
			state = "index_required"
		}
		status.Repo = &RepoResolution{
			RepoName: binding.RepoName,
			RepoPath: binding.RepoPath,
			State:    state,
			Message:  runtimeErr.Message,
		}
		return status, nil
	}
	return Status{}, err
}

func (c *Controller) StartChat(ctx context.Context, request ChatRequest) (*Job, ResolvedRepo, error) {
	resolved, err := c.resolver.Resolve(request.Binding())
	if err != nil {
		return nil, ResolvedRepo{}, err
	}

	status, err := c.adapter.GetStatus(ctx)
	if err != nil {
		return nil, ResolvedRepo{}, err
	}
	if !resolved.Indexed {
		return nil, ResolvedRepo{}, NewRuntimeError(
			ErrorIndexRequired,
			"Repository \""+resolved.RepoPath+"\" is not indexed yet. Run analyze first.",
			409,
			map[string]any{"repoName": resolved.RepoName, "repoPath": resolved.RepoPath},
		)
	}

	c.mu.Lock()
	if existingID := c.activeRepoSessions[resolved.RepoPath]; existingID != "" {
		if existing := c.jobs[existingID]; existing != nil {
			existing.Cancel("Superseded by a newer chat on the same repository")
		}
	}
	job := NewJob(c.adapter.Provider(), resolved.RepoName, resolved.RepoPath)
	c.jobs[job.ID] = job
	c.activeRepoSessions[resolved.RepoPath] = job.ID
	c.mu.Unlock()

	job.Emit(Event{
		Type:               "session_started",
		RuntimeEnvironment: status.RuntimeEnvironment,
		ExecutionMode:      c.adapter.ExecutionMode(),
	})

	go func() {
		err := c.adapter.RunChat(job.Context(), job, request, ChatContext{Repo: resolved})
		if err != nil {
			job.mu.Lock()
			running := job.Status == JobRunning
			job.mu.Unlock()
			if running {
				runtimeErr := wrapStartError(err)
				job.Emit(Event{Type: "error", Code: runtimeErr.Code, Error: runtimeErr.Message})
			}
		}
		c.mu.Lock()
		if c.activeRepoSessions[resolved.RepoPath] == job.ID {
			delete(c.activeRepoSessions, resolved.RepoPath)
		}
		c.mu.Unlock()
	}()

	return job, resolved, nil
}

func (c *Controller) CancelSession(sessionID string, reason string) bool {
	c.mu.Lock()
	job := c.jobs[sessionID]
	c.mu.Unlock()
	if job == nil {
		return false
	}
	job.mu.Lock()
	running := job.Status == JobRunning
	job.mu.Unlock()
	if !running {
		return false
	}
	job.Cancel(reason)
	return true
}

func (c *Controller) GetSession(sessionID string) *Job {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.jobs[sessionID]
}

func (c *Controller) Dispose() {
	c.mu.Lock()
	jobs := make([]*Job, 0, len(c.jobs))
	for _, job := range c.jobs {
		jobs = append(jobs, job)
	}
	c.jobs = map[string]*Job{}
	c.activeRepoSessions = map[string]string{}
	c.mu.Unlock()

	for _, job := range jobs {
		job.Cancel("Runtime shutting down")
	}
	time.Sleep(0)
}
