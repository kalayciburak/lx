package signal

import (
	"github.com/kalayciburak/lx/internal/logx"
)

func Lifetime(entries []logx.Entry, targetMsg string) *SignalResult {
	if targetMsg == "" {
		return &SignalResult{
			Type:  SignalLifetime,
			Title: "Signal Lifetime",
			Lifetime: &LifetimeResult{
				Message:     "",
				Occurrences: 0,
				IsSingle:    true,
			},
		}
	}

	var timestamps []string
	count := 0

	for _, e := range entries {
		if e.Deleted {
			continue
		}
		if e.Message == targetMsg {
			count++
			if e.Timestamp != "" {
				timestamps = append(timestamps, e.Timestamp)
			}
		}
	}

	result := &LifetimeResult{
		Message:     targetMsg,
		Occurrences: count,
	}

	if len(timestamps) == 0 {
		result.IsSingle = true
		result.FirstSeen = "no timestamp"
		result.LastSeen = "no timestamp"
	} else if len(timestamps) == 1 {
		result.IsSingle = true
		result.FirstSeen = timestamps[0]
		result.LastSeen = timestamps[0]
	} else {
		result.IsSingle = false
		result.FirstSeen = timestamps[0]
		result.LastSeen = timestamps[len(timestamps)-1]
	}

	return &SignalResult{
		Type:     SignalLifetime,
		Title:    "Signal Lifetime",
		Lifetime: result,
	}
}
