package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/tamnguyendinh/avmatrix-go/internal/ignore"
)

const MaxFileSize int64 = 512 * 1024
const readConcurrency = 32

type File struct {
	Path     string   `json:"path"`
	Size     int64    `json:"size"`
	Hash     string   `json:"hash"`
	Language Language `json:"language,omitempty"`
}

type Metrics struct {
	Visited        int `json:"visited"`
	Included       int `json:"included"`
	SkippedIgnored int `json:"skippedIgnored"`
	SkippedLarge   int `json:"skippedLarge"`
	SkippedErrored int `json:"skippedErrored"`
}

type Options struct {
	NoGitignore bool
	Include     []string
	Exclude     []string
}

type ProgressFunc func(current int, total int, filePath string)

func WalkRepositoryPaths(repoPath string, options Options, onProgress ProgressFunc) ([]File, Metrics, error) {
	matcher, err := ignore.Load(repoPath, ignore.Options{NoGitignore: options.NoGitignore})
	if err != nil {
		return nil, Metrics{}, err
	}
	progress := func(current int, total int, filePath string) {
		if onProgress != nil {
			onProgress(current, total, filePath)
		}
	}

	var candidates []string
	metrics := Metrics{}
	err = filepath.WalkDir(repoPath, func(fullPath string, dirEntry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			metrics.SkippedErrored++
			if dirEntry != nil && dirEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if fullPath == repoPath {
			return nil
		}

		rel, err := filepath.Rel(repoPath, fullPath)
		if err != nil {
			metrics.SkippedErrored++
			return nil
		}
		rel = ignore.NormalizePath(rel)
		if rel == "" {
			return nil
		}

		if dirEntry.IsDir() {
			if matcher.Ignored(rel, true) {
				metrics.SkippedIgnored++
				return filepath.SkipDir
			}
			return nil
		}

		metrics.Visited++
		if matcher.Ignored(rel, false) || !matchesSelection(rel, options) {
			metrics.SkippedIgnored++
			return nil
		}
		candidates = append(candidates, rel)
		return nil
	})
	if err != nil {
		return nil, metrics, err
	}
	sort.Strings(candidates)

	files := make([]File, 0, len(candidates))
	workerCount := minInt(readConcurrency, runtime.GOMAXPROCS(0)*4, len(candidates))
	if workerCount == 0 {
		return files, metrics, nil
	}

	jobs := make(chan int)
	results := make(chan scanResult, len(candidates))
	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				results <- scanCandidate(repoPath, candidates[index], index)
			}
		}()
	}
	go func() {
		for index := range candidates {
			jobs <- index
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	processed := 0
	for result := range results {
		processed++
		switch {
		case result.err != nil:
			metrics.SkippedErrored++
		case result.large:
			metrics.SkippedLarge++
		default:
			files = append(files, result.file)
			metrics.Included++
		}
		progress(processed, len(candidates), result.rel)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	return files, metrics, nil
}

type scanResult struct {
	index int
	rel   string
	file  File
	large bool
	err   error
}

func scanCandidate(repoPath string, rel string, index int) scanResult {
	fullPath := filepath.Join(repoPath, filepath.FromSlash(rel))
	info, err := os.Stat(fullPath)
	if err != nil {
		return scanResult{index: index, rel: rel, err: err}
	}
	if info.Size() > MaxFileSize {
		return scanResult{index: index, rel: rel, large: true}
	}

	hash, err := hashFile(fullPath)
	if err != nil {
		return scanResult{index: index, rel: rel, err: err}
	}

	file := File{Path: rel, Size: info.Size(), Hash: hash}
	if lang, ok := DetectLanguage(rel); ok {
		file.Language = lang
	}
	return scanResult{index: index, rel: rel, file: file}
}

func minInt(first int, rest ...int) int {
	minimum := first
	for _, value := range rest {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}

func ReadFileContents(repoPath string, relativePaths []string) (map[string]string, error) {
	contents := make(map[string]string, len(relativePaths))
	for _, rel := range relativePaths {
		normalized := ignore.NormalizePath(rel)
		raw, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(normalized)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return contents, err
		}
		contents[normalized] = string(raw)
	}
	return contents, nil
}

func hashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
