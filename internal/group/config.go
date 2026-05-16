package group

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseConfig(content string) (Config, error) {
	config := Config{
		Repos:    map[string]string{},
		Links:    []ManifestLink{},
		Packages: map[string]map[string]string{},
		Detect:   defaultDetectConfig(),
		Matching: defaultMatchingConfig(),
	}
	section := ""
	var currentLink *ManifestLink
	currentPackage := ""
	sawRepos := false
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			currentLink = nil
			currentPackage = ""
			key, value, ok := splitYAMLKeyValue(trimmed)
			if !ok {
				continue
			}
			if value == "" {
				section = key
				if key == "repos" {
					sawRepos = true
				}
				continue
			}
			section = ""
			switch key {
			case "version":
				version, err := strconv.Atoi(unquoteYAML(value))
				if err != nil {
					return Config{}, fmt.Errorf("version must be a number")
				}
				config.Version = version
			case "name":
				config.Name = unquoteYAML(value)
			case "description":
				config.Description = unquoteYAML(value)
			case "repos":
				sawRepos = true
				if unquoteYAML(value) != "{}" {
					return Config{}, fmt.Errorf("repos is required in group.yaml (must be a mapping)")
				}
			case "links":
				if unquoteYAML(value) != "[]" {
					section = "links"
				}
			}
			continue
		}

		switch section {
		case "repos":
			key, value, ok := splitYAMLKeyValue(trimmed)
			if ok {
				config.Repos[unquoteYAML(key)] = unquoteYAML(value)
			}
		case "links":
			if strings.HasPrefix(trimmed, "- ") {
				link := ManifestLink{}
				config.Links = append(config.Links, link)
				currentLink = &config.Links[len(config.Links)-1]
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				if trimmed == "" {
					continue
				}
			}
			if currentLink == nil {
				continue
			}
			key, value, ok := splitYAMLKeyValue(trimmed)
			if ok {
				assignLinkField(currentLink, key, unquoteYAML(value))
			}
		case "detect":
			key, value, ok := splitYAMLKeyValue(trimmed)
			if !ok {
				continue
			}
			if err := assignDetectField(&config.Detect, key, value); err != nil {
				return Config{}, err
			}
		case "matching":
			key, value, ok := splitYAMLKeyValue(trimmed)
			if !ok {
				continue
			}
			if err := assignMatchingField(&config.Matching, key, value); err != nil {
				return Config{}, err
			}
		case "packages":
			key, value, ok := splitYAMLKeyValue(trimmed)
			if !ok {
				continue
			}
			key = unquoteYAML(key)
			if value == "" {
				currentPackage = key
				if config.Packages[currentPackage] == nil {
					config.Packages[currentPackage] = map[string]string{}
				}
				continue
			}
			if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t\t") {
				if currentPackage == "" {
					continue
				}
				config.Packages[currentPackage][key] = unquoteYAML(value)
				continue
			}
			config.Packages[key] = map[string]string{"value": unquoteYAML(value)}
		}
	}
	if config.Version != 1 {
		return Config{}, fmt.Errorf("Unsupported group.yaml version: %d. Expected 1.", config.Version)
	}
	if strings.TrimSpace(config.Name) == "" {
		return Config{}, fmt.Errorf("name is required in group.yaml")
	}
	if !sawRepos || config.Repos == nil {
		return Config{}, fmt.Errorf("repos is required in group.yaml (must be a mapping)")
	}
	for index, link := range config.Links {
		if _, ok := config.Repos[link.From]; !ok {
			return Config{}, fmt.Errorf("links[%d].from %q does not match any repo path in group", index, link.From)
		}
		if _, ok := config.Repos[link.To]; !ok {
			return Config{}, fmt.Errorf("links[%d].to %q does not match any repo path in group", index, link.To)
		}
		if !validContractType(link.Type) {
			return Config{}, fmt.Errorf("links[%d].type %q is invalid. Expected: http, grpc, topic, lib, custom", index, link.Type)
		}
		if !validContractRole(link.Role) {
			return Config{}, fmt.Errorf("links[%d].role %q is invalid. Expected: provider | consumer", index, link.Role)
		}
		if strings.TrimSpace(link.Contract) == "" {
			return Config{}, fmt.Errorf("links[%d].contract is required", index)
		}
	}
	return config, nil
}

func defaultDetectConfig() DetectConfig {
	return DetectConfig{
		HTTP:              true,
		GRPC:              true,
		Topics:            true,
		SharedLibs:        true,
		EmbeddingFallback: true,
	}
}

func defaultMatchingConfig() MatchingConfig {
	return MatchingConfig{
		BM25Threshold:        0.7,
		EmbeddingThreshold:   0.65,
		MaxCandidatesPerStep: 3,
	}
}

func splitYAMLKeyValue(line string) (string, string, bool) {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(key), strings.TrimSpace(value), true
}

func unquoteYAML(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}

func assignLinkField(link *ManifestLink, key string, value string) {
	switch key {
	case "from":
		link.From = value
	case "to":
		link.To = value
	case "type":
		link.Type = value
	case "contract":
		link.Contract = value
	case "role":
		link.Role = value
	}
}

func assignDetectField(config *DetectConfig, key string, value string) error {
	parsed, err := parseYAMLBool(value)
	if err != nil {
		return fmt.Errorf("detect.%s must be a boolean", key)
	}
	switch key {
	case "http":
		config.HTTP = parsed
	case "grpc":
		config.GRPC = parsed
	case "topics":
		config.Topics = parsed
	case "shared_libs":
		config.SharedLibs = parsed
	case "embedding_fallback":
		config.EmbeddingFallback = parsed
	}
	return nil
}

func assignMatchingField(config *MatchingConfig, key string, value string) error {
	switch key {
	case "bm25_threshold":
		parsed, err := strconv.ParseFloat(unquoteYAML(value), 64)
		if err != nil {
			return fmt.Errorf("matching.%s must be a number", key)
		}
		config.BM25Threshold = parsed
	case "embedding_threshold":
		parsed, err := strconv.ParseFloat(unquoteYAML(value), 64)
		if err != nil {
			return fmt.Errorf("matching.%s must be a number", key)
		}
		config.EmbeddingThreshold = parsed
	case "max_candidates_per_step":
		parsed, err := strconv.Atoi(unquoteYAML(value))
		if err != nil {
			return fmt.Errorf("matching.%s must be a number", key)
		}
		config.MaxCandidatesPerStep = parsed
	}
	return nil
}

func parseYAMLBool(value string) (bool, error) {
	switch strings.ToLower(unquoteYAML(value)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean")
	}
}

func validContractType(value string) bool {
	switch value {
	case "http", "grpc", "topic", "lib", "custom":
		return true
	default:
		return false
	}
}

func validContractRole(value string) bool {
	return value == "provider" || value == "consumer"
}
