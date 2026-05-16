package httpapi

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJobManagerDeduplicatesActiveSameRepoAndRejectsOtherActiveRepo(t *testing.T) {
	manager := NewJobManager()
	current := time.Unix(1_700_000_000, 0)
	manager.now = func() time.Time { return current }

	repoPath := filepath.Join(t.TempDir(), "repo")
	first, created, err := manager.Create(repoPath, "alpha")
	if err != nil {
		t.Fatalf("Create first job error = %v", err)
	}
	if !created {
		t.Fatal("first job was not marked created")
	}

	second, created, err := manager.Create(filepath.Join(repoPath, "."), "ignored")
	if err != nil {
		t.Fatalf("Create same-repo job error = %v", err)
	}
	if created {
		t.Fatal("same-repo job should return the existing active job")
	}
	if second.ID != first.ID || second.RepoName != first.RepoName {
		t.Fatalf("same-repo job = %#v, want existing %#v", second, first)
	}

	_, _, err = manager.Create(t.TempDir(), "beta")
	if err == nil || !strings.Contains(err.Error(), "already in progress") {
		t.Fatalf("different active repo error = %v, want already in progress", err)
	}
}

func TestJobManagerCleansExpiredTerminalJobs(t *testing.T) {
	manager := NewJobManager()
	current := time.Unix(1_700_000_000, 0)
	manager.now = func() time.Time { return current }

	job, created, err := manager.Create(t.TempDir(), "alpha")
	if err != nil {
		t.Fatalf("Create job error = %v", err)
	}
	if !created {
		t.Fatal("job was not marked created")
	}
	manager.Complete(job.ID)

	current = current.Add(jobTTL + time.Millisecond)
	if _, ok := manager.Get(job.ID); ok {
		t.Fatal("expired terminal job was not cleaned")
	}
	if jobs := manager.List(); len(jobs) != 0 {
		t.Fatalf("List returned %d jobs after cleanup, want 0", len(jobs))
	}
}
