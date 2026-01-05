package signal

import (
	"time"

	"github.com/kalayciburak/lx/internal/logx"
)

var timeFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05.000Z",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.000",
	"2006-01-02 15:04:05",
	"02/Jan/2006:15:04:05",
	"Jan 02 15:04:05",
	"15:04:05.000",
	"15:04:05",
}

func DetectBurst(entries []logx.Entry, targetMsg string) *SignalResult {
	if targetMsg == "" {
		return &SignalResult{
			Type:  SignalBurst,
			Title: "Burst Detector",
			Burst: &BurstResult{
				Message:     "",
				Detected:    false,
				Description: "No message selected",
			},
		}
	}

	var times []time.Time
	for _, e := range entries {
		if e.Deleted || e.Message != targetMsg {
			continue
		}
		if t := parseTimestamp(e.Timestamp); !t.IsZero() {
			times = append(times, t)
		}
	}

	if len(times) < 2 {
		return &SignalResult{
			Type:  SignalBurst,
			Title: "Burst Detector",
			Burst: &BurstResult{
				Message:     truncateMsg(targetMsg, 50),
				Detected:    false,
				Count:       len(times),
				Description: "Not enough data points with timestamps",
			},
		}
	}

	type windowConfig struct {
		seconds   int
		threshold int
	}
	windows := []windowConfig{
		{10, 5},
		{30, 8},
		{60, 15},
	}

	for _, w := range windows {
		maxCount, _ := findMaxInWindow(times, time.Duration(w.seconds)*time.Second)
		if maxCount >= w.threshold {
			return &SignalResult{
				Type:  SignalBurst,
				Title: "Burst Detector",
				Burst: &BurstResult{
					Message:     truncateMsg(targetMsg, 50),
					Detected:    true,
					Count:       maxCount,
					WindowSecs:  w.seconds,
					Description: itoa(maxCount) + " occurrences in " + itoa(w.seconds) + "s window",
				},
			}
		}
	}

	return &SignalResult{
		Type:  SignalBurst,
		Title: "Burst Detector",
		Burst: &BurstResult{
			Message:     truncateMsg(targetMsg, 50),
			Detected:    false,
			Count:       len(times),
			Description: "No abnormal burst pattern detected",
		},
	}
}

func findMaxInWindow(times []time.Time, window time.Duration) (maxCount int, windowStart time.Time) {
	if len(times) == 0 {
		return 0, time.Time{}
	}

	maxCount = 1
	windowStart = times[0]

	for i := 0; i < len(times); i++ {
		count := 1
		windowEnd := times[i].Add(window)
		for j := i + 1; j < len(times); j++ {
			if times[j].Before(windowEnd) || times[j].Equal(windowEnd) {
				count++
			}
		}
		if count > maxCount {
			maxCount = count
			windowStart = times[i]
		}
	}

	return maxCount, windowStart
}

func parseTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}
	for _, format := range timeFormats {
		if t, err := time.Parse(format, ts); err == nil {
			return t
		}
	}
	return time.Time{}
}

func truncateMsg(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen-1] + "…"
}
