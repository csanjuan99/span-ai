package start

import (
	"time"
)

// Health is the answer to GET /health and span start health.
type Health struct {
	Status    string `json:"status"`
	Version   string `json:"version,omitempty"`
	LatencyMs int64  `json:"latencyMs"`
}

// Probe reports that the server is listening. Latency is the time since started.
func Probe(version string, started time.Time) Health {
	ms := time.Since(started).Milliseconds()
	if ms < 0 {
		ms = 0
	}
	return Health{Status: "connected", Version: version, LatencyMs: ms}
}

// RefusedHealth answers when the access token was wrong.
func RefusedHealth() Health {
	return Health{Status: "refused", LatencyMs: 0}
}
