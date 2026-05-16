package resolution

import (
	"errors"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type bindingEntry struct {
	Name string
	Type scopeir.TypeRef
}

type bindingAccumulator struct {
	allByFile       map[string][]bindingEntry
	fileScopeByFile map[string]map[string]scopeir.TypeRef
	totalBindings   int
	finalized       bool
	disposed        bool
}

func newBindingAccumulator() *bindingAccumulator {
	return &bindingAccumulator{
		allByFile:       make(map[string][]bindingEntry),
		fileScopeByFile: make(map[string]map[string]scopeir.TypeRef),
	}
}

func (a *bindingAccumulator) appendFile(filePath string, entries []bindingEntry) error {
	if a.disposed {
		return errors.New("binding accumulator use after dispose")
	}
	if a.finalized {
		return errors.New("binding accumulator append after finalize")
	}
	if filePath == "" || len(entries) == 0 {
		return nil
	}
	a.allByFile[filePath] = append(a.allByFile[filePath], entries...)
	fileScope := a.fileScopeByFile[filePath]
	if fileScope == nil {
		fileScope = make(map[string]scopeir.TypeRef)
		a.fileScopeByFile[filePath] = fileScope
	}
	for _, entry := range entries {
		if entry.Name == "" {
			continue
		}
		fileScope[entry.Name] = entry.Type
	}
	a.totalBindings += len(entries)
	return nil
}

func (a *bindingAccumulator) finalize() error {
	if a.finalized {
		return nil
	}
	for filePath, fileScope := range a.fileScopeByFile {
		all := a.allByFile[filePath]
		if len(all) == 0 {
			return errors.New("binding accumulator storage split drift")
		}
		seen := make(map[string]struct{}, len(all))
		for _, entry := range all {
			if entry.Name == "" {
				continue
			}
			seen[entry.Name] = struct{}{}
		}
		if len(seen) != len(fileScope) {
			return errors.New("binding accumulator file-scope projection drift")
		}
	}
	a.finalized = true
	return nil
}

func (a *bindingAccumulator) dispose() {
	a.allByFile = make(map[string][]bindingEntry)
	a.fileScopeByFile = make(map[string]map[string]scopeir.TypeRef)
	a.totalBindings = 0
	a.disposed = true
}

func (a *bindingAccumulator) fileScopeGet(filePath string, name string) (scopeir.TypeRef, bool) {
	fileScope := a.fileScopeByFile[filePath]
	if fileScope == nil {
		return scopeir.TypeRef{}, false
	}
	value, ok := fileScope[name]
	return value, ok
}

func (a *bindingAccumulator) fileCount() int {
	return len(a.allByFile)
}

func (a *bindingAccumulator) total() int {
	return a.totalBindings
}
