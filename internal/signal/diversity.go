package signal

import (
	"github.com/kalayciburak/lx/internal/logx"
)

func Diversity(entries []logx.Entry) *SignalResult {
	uniqueMessages := make(map[string]struct{})
	totalErrors := 0

	for _, e := range entries {
		if e.Deleted {
			continue
		}
		if e.Level == logx.LevelError {
			totalErrors++
			uniqueMessages[e.Message] = struct{}{}
		}
	}

	uniqueCount := len(uniqueMessages)

	var ratio float64
	if totalErrors > 0 {
		ratio = float64(uniqueCount) / float64(totalErrors)
	}

	quality, reason := classifyDiversity(totalErrors, uniqueCount, ratio)

	return &SignalResult{
		Type:  SignalDiversity,
		Title: "Error Diversity",
		Diversity: &DiversityResult{
			TotalErrors:   totalErrors,
			UniqueErrors:  uniqueCount,
			Ratio:         ratio,
			Quality:       quality,
			QualityReason: reason,
		},
	}
}

func classifyDiversity(total, unique int, ratio float64) (quality, reason string) {
	if total == 0 {
		return "HIGH", "No errors - clean logs!"
	}

	if unique == 1 {
		return "HIGH", "Single error type - easy to focus"
	}

	if ratio <= 0.1 {
		return "HIGH", "Repetitive errors - clear pattern"
	}

	if ratio <= 0.3 {
		return "MEDIUM", "Moderate variety - some patterns visible"
	}

	if ratio <= 0.5 {
		return "MEDIUM", "Mixed errors - multiple issues"
	}

	return "LOW", "High diversity - many different errors"
}
