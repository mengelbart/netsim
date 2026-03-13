package netsim

import (
	"encoding/json"
	"fmt"
	"time"
)

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
		qn, ok := n.(*QueueNode)
		if !ok {
			continue
		}
		if dq, ok := qn.queue.(*DelayQueue); ok {
			tc.Delay = dq.delay.Microseconds()
		}
		if rq, ok := qn.queue.(*RateQueue); ok {
			tc.Bandwidth = formatBitrateTC(float64(rq.limiter.Limit()) * 8)
			tc.Burst = rq.limiter.Burst()
			tc.Limit = rq.queueSize
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
