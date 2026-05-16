package parser

import "time"

type Metrics struct {
	Total          int           `json:"total"`
	Succeeded      int           `json:"succeeded"`
	Failed         int           `json:"failed"`
	Unsupported    int           `json:"unsupported"`
	TimedOut       int           `json:"timedOut"`
	CreatedParsers int           `json:"createdParsers"`
	TotalBytes     int64         `json:"totalBytes"`
	TotalDuration  time.Duration `json:"totalDuration"`
}
