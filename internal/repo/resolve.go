package repo

import (
	"fmt"
	"path/filepath"
	"strings"
)

type AmbiguousNameError struct {
	Name    string
	Matches []RegistryEntry
}

func (e AmbiguousNameError) Error() string {
	return fmt.Sprintf("registry name %q is ambiguous across %d repositories", e.Name, len(e.Matches))
}

func ResolveEntry(entries []RegistryEntry, query string) (RegistryEntry, error) {
	if filepath.IsAbs(query) {
		resolved := absClean(query)
		for _, entry := range entries {
			if SamePath(entry.Path, resolved) {
				return entry, nil
			}
		}
		return RegistryEntry{}, fmt.Errorf("repository path %q is not registered", query)
	}

	var matches []RegistryEntry
	for _, entry := range entries {
		if strings.EqualFold(entry.Name, query) {
			matches = append(matches, entry)
		}
	}
	switch len(matches) {
	case 0:
		return RegistryEntry{}, fmt.Errorf("repository %q is not registered", query)
	case 1:
		return matches[0], nil
	default:
		return RegistryEntry{}, AmbiguousNameError{Name: query, Matches: matches}
	}
}
