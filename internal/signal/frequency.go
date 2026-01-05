package signal

import (
	"sort"

	"github.com/kalayciburak/lx/internal/logx"
)

func ErrorFrequency(entries []logx.Entry, limit int) *SignalResult {
	if limit <= 0 {
		limit = 10
	}

	counts := make(map[string]int)
	for _, e := range entries {
		if e.Level == logx.LevelError && !e.Deleted {
			counts[e.Message]++
		}
	}

	var results []FrequencyResult
	for msg, count := range counts {
		results = append(results, FrequencyResult{
			Message: msg,
			Count:   count,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Count > results[j].Count
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return &SignalResult{
		Type:      SignalFrequency,
		Title:     "Error Frequency",
		Frequency: results,
	}
}
