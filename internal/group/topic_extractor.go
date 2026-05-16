package group

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var topicSourceExtensions = map[string]bool{
	".ts":   true,
	".tsx":  true,
	".js":   true,
	".jsx":  true,
	".go":   true,
	".java": true,
	".py":   true,
}

func ExtractTopicContracts(repoPath string) ([]StoredContract, error) {
	files, err := walkGroupSourceFiles(repoPath, topicSourceExtensions, true)
	if err != nil {
		return nil, err
	}
	out := make([]StoredContract, 0)
	for _, rel := range files {
		content := readGroupSourceFile(repoPath, rel)
		if content == "" {
			continue
		}
		out = append(out, scanTopicFile(rel, content)...)
	}
	return dedupeTopicContracts(out), nil
}

func scanTopicFile(rel string, content string) []StoredContract {
	ext := strings.ToLower(filepath.Ext(rel))
	out := make([]StoredContract, 0)
	switch ext {
	case ".java":
		out = append(out, topicMatches(rel, content, `@KafkaListener\([^)]*topics\s*=\s*"([^"]+)"`, "consumer", "kafka", "@KafkaListener", 0.8)...)
		out = append(out, topicMatches(rel, content, `kafkaTemplate\.send\(\s*"([^"]+)"`, "provider", "kafka", "kafkaTemplate.send", 0.75)...)
		out = append(out, topicMatches(rel, content, `@RabbitListener\([^)]*queues\s*=\s*"([^"]+)"`, "consumer", "rabbitmq", "@RabbitListener", 0.8)...)
		out = append(out, topicMatches(rel, content, `rabbitTemplate\.convertAndSend\(\s*"([^"]+)"`, "provider", "rabbitmq", "rabbitTemplate.convertAndSend", 0.75)...)
	case ".ts", ".tsx", ".js", ".jsx":
		out = append(out, topicMatches(rel, content, `consumer\.subscribe\(\s*\{[^}]*topic\s*:\s*['"]([^'"]+)['"]`, "consumer", "kafka", "consumer.subscribe", 0.75)...)
		out = append(out, topicMatches(rel, content, `producer\.send\(\s*\{[^}]*topic\s*:\s*['"]([^'"]+)['"]`, "provider", "kafka", "producer.send", 0.75)...)
		out = append(out, topicMatches(rel, content, `channel\.consume\(\s*["']([^"']+)["']`, "consumer", "rabbitmq", "channel.consume", 0.75)...)
		out = append(out, topicMatches(rel, content, `channel\.publish\(\s*["']([^"']+)["']`, "provider", "rabbitmq", "channel.publish", 0.75)...)
		out = append(out, topicMatches(rel, content, `channel\.sendToQueue\(\s*["']([^"']+)["']`, "provider", "rabbitmq", "channel.sendToQueue", 0.75)...)
		out = append(out, topicMatches(rel, content, `nc\.subscribe\(\s*["']([^"']+)["']`, "consumer", "nats", "nc.subscribe", 0.75)...)
		out = append(out, topicMatches(rel, content, `nc\.publish\(\s*["']([^"']+)["']`, "provider", "nats", "nc.publish", 0.75)...)
	case ".go":
		out = append(out, topicMatches(rel, content, `js\.Publish\(\s*"([^"]+)"`, "provider", "nats", "js.Publish", 0.75)...)
		out = append(out, topicMatches(rel, content, `js\.Subscribe\(\s*"([^"]+)"`, "consumer", "nats", "js.Subscribe", 0.75)...)
		out = append(out, topicMatches(rel, content, `nc\.Subscribe\(\s*"([^"]+)"`, "consumer", "nats", "nc.Subscribe", 0.75)...)
		out = append(out, topicMatches(rel, content, `nc\.Publish\(\s*"([^"]+)"`, "provider", "nats", "nc.Publish", 0.75)...)
		out = append(out, topicMatches(rel, content, `ConsumePartition\(\s*"([^"]+)"`, "consumer", "kafka", "ConsumePartition", 0.75)...)
		out = append(out, topicMatches(rel, content, `ProducerMessage\s*\{[^}]*Topic\s*:\s*"([^"]+)"`, "provider", "kafka", "ProducerMessage", 0.75)...)
		out = append(out, topicMatches(rel, content, `kafka\.Writer\s*\{[^}]*Topic\s*:\s*"([^"]+)"`, "provider", "kafka", "kafka.Writer", 0.75)...)
		out = append(out, topicMatches(rel, content, `kafka\.ReaderConfig\s*\{[^}]*Topic\s*:\s*"([^"]+)"`, "consumer", "kafka", "kafka.Reader", 0.75)...)
	case ".py":
		out = append(out, topicMatches(rel, content, `KafkaConsumer\(\s*['"]([^'"]+)['"]`, "consumer", "kafka", "KafkaConsumer", 0.75)...)
		out = append(out, topicMatches(rel, content, `producer\.send\(\s*['"]([^'"]+)['"]`, "provider", "kafka", "producer.send", 0.75)...)
		out = append(out, topicMatches(rel, content, `nc\.subscribe\(\s*["']([^"']+)["']`, "consumer", "nats", "nc.subscribe", 0.75)...)
		out = append(out, topicMatches(rel, content, `nc\.publish\(\s*["']([^"']+)["']`, "provider", "nats", "nc.publish", 0.75)...)
	}
	return out
}

func topicMatches(rel string, content string, pattern string, role string, broker string, symbolName string, confidence float64) []StoredContract {
	re := regexp.MustCompile("(?is)" + pattern)
	out := make([]StoredContract, 0)
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		topic := strings.TrimSpace(match[1])
		if topic == "" {
			continue
		}
		out = append(out, StoredContract{
			ContractID: "topic::" + topic,
			Type:       "topic",
			Role:       role,
			SymbolUID:  "",
			SymbolRef:  SymbolRef{FilePath: rel, Name: symbolName},
			SymbolName: symbolName,
			Confidence: confidence,
			Meta: map[string]any{
				"broker":             broker,
				"topicName":          topic,
				"extractionStrategy": "source_scan",
			},
		})
	}
	return out
}

func dedupeTopicContracts(items []StoredContract) []StoredContract {
	seen := make(map[string]bool, len(items))
	out := make([]StoredContract, 0, len(items))
	for _, item := range items {
		key := item.ContractID + "\x00" + item.Role + "\x00" + item.SymbolRef.FilePath
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ContractID != out[j].ContractID {
			return out[i].ContractID < out[j].ContractID
		}
		if out[i].Role != out[j].Role {
			return out[i].Role < out[j].Role
		}
		return out[i].SymbolRef.FilePath < out[j].SymbolRef.FilePath
	})
	return out
}
