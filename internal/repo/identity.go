package repo

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

func SamePath(left string, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func RuntimeID(repoPath string) string {
	normalized := filepath.Clean(repoPath)
	if runtime.GOOS == "windows" {
		normalized = strings.ToLower(normalized)
	}
	sum := sha256.Sum256([]byte(normalized))
	return base64.RawURLEncoding.EncodeToString(sum[:])[:16]
}

func DisplayLabel(entry RegistryEntry, entries []RegistryEntry) string {
	duplicate := false
	for _, candidate := range entries {
		if candidate.Path != entry.Path && strings.EqualFold(candidate.Name, entry.Name) {
			duplicate = true
			break
		}
	}
	if !duplicate {
		return entry.Name
	}
	return fmt.Sprintf("%s (%s)", entry.Name, entry.Path)
}
