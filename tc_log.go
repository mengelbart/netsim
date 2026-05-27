package netsim

import (
	"encoding/json"
	"fmt"
	"time"
)

type TcLogData struct {
	Bandwidth float64
	Burst     int
	Limit     int
	Delay     int64
}

type TcLogProvider interface {
	TcLog() TcLogData
}

type tcLogEntry struct {
	TrafficControl bool   `json:"traffic_control"`
	Duration       int    `json:"duration"`
	Bandwidth      string `json:"bandwidth"`
	Burst          int    `json:"burst"`
	Limit          int    `json:"limit"`
	Delay          int64  `json:"delay"` // microseconds
	Time           string `json:"time"`
}

func newTcLogEntry(path *Path, start time.Time, duration time.Duration) tcLogEntry {
	tc := tcLogEntry{
		TrafficControl: true,
		Duration:       int(duration.Seconds()),
		Time:           start.Format(time.RFC3339Nano),
	}

	for _, n := range path.nodes {
		provider, ok := n.(TcLogProvider)
		if !ok {
			continue
		}
		data := provider.TcLog()
		if data.Bandwidth != 0 {
			tc.Bandwidth = formatBitrateTC(data.Bandwidth)
		}
		if data.Burst != 0 {
			tc.Burst = data.Burst
		}
		if data.Limit != 0 {
			tc.Limit = data.Limit
		}
		if data.Delay != 0 {
			tc.Delay = data.Delay
		}
	}

	return tc
}

func (t *tcLogEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(*t)
}

func formatBitrateTC(bitrate float64) string {
	switch {
	case bitrate >= 1_000_000 && int64(bitrate)%1_000_000 == 0:
		return fmt.Sprintf("%dmbit", int64(bitrate)/1_000_000)
	case bitrate >= 1_000 && int64(bitrate)%1_000 == 0:
		return fmt.Sprintf("%dkbit", int64(bitrate)/1_000)
	default:
		return fmt.Sprintf("%dbit", int64(bitrate))
	}
}
