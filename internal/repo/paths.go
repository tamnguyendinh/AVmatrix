package repo

import (
	"os"
	"path/filepath"
)

const (
	StorageDirName = ".avmatrix"
	HomeEnvName    = "AVMATRIX_HOME"
)

type StoragePaths struct {
	StoragePath     string
	LbugPath        string
	GraphPath       string
	MetaPath        string
	AnalyzeLockPath string
	AnalyzeTempPath string
}

func StoragePath(repoPath string) string {
	return filepath.Join(absClean(repoPath), StorageDirName)
}

func Paths(repoPath string) StoragePaths {
	storagePath := StoragePath(repoPath)
	return StoragePaths{
		StoragePath:     storagePath,
		LbugPath:        filepath.Join(storagePath, "lbug"),
		GraphPath:       filepath.Join(storagePath, "graph.json"),
		MetaPath:        filepath.Join(storagePath, "meta.json"),
		AnalyzeLockPath: filepath.Join(storagePath, "analyze.lock"),
		AnalyzeTempPath: filepath.Join(storagePath, "analyze.tmp"),
	}
}

func GlobalDir() string {
	if home := os.Getenv(HomeEnvName); home != "" {
		return home
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return StorageDirName
	}
	return filepath.Join(userHome, StorageDirName)
}

func GlobalRegistryPath() string {
	return filepath.Join(GlobalDir(), "registry.json")
}

func absClean(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(absolute)
}
