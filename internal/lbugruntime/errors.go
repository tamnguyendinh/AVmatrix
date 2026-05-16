package lbugruntime

import (
	"regexp"
	"strings"
)

var missingColumnOrTablePattern = regexp.MustCompile(`(?i)(table|column|property).*not found`)

var walCorruptionPatterns = []string{
	"corrupted wal file",
	"invalid wal record type",
}

func IsMissingColumnOrTableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") || missingColumnOrTablePattern.MatchString(msg)
}

func IsBusyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "busy") ||
		strings.Contains(msg, "lock") ||
		strings.Contains(msg, "already in use") ||
		strings.Contains(msg, "could not set lock")
}

func IsWALCorruptionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, pattern := range walCorruptionPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

func IsAlreadyLoadedOrInstalledError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already loaded") ||
		strings.Contains(msg, "already installed") ||
		strings.Contains(msg, "already exists")
}
