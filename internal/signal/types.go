package signal

type SignalType int

const (
	SignalFrequency SignalType = iota
	SignalLifetime
	SignalBurst
	SignalDiversity
)

type FrequencyResult struct {
	Message string
	Count   int
}

type LifetimeResult struct {
	Message     string
	FirstSeen   string
	LastSeen    string
	Occurrences int
	IsSingle    bool
}

type BurstResult struct {
	Message     string
	Detected    bool
	Count       int
	WindowSecs  int
	Description string
}

type DiversityResult struct {
	TotalErrors   int
	UniqueErrors  int
	Ratio         float64
	Quality       string
	QualityReason string
}

type SignalResult struct {
	Type       SignalType
	Title      string
	Frequency  []FrequencyResult
	Lifetime   *LifetimeResult
	Burst      *BurstResult
	Diversity  *DiversityResult
}

func (r *SignalResult) FormatForClipboard() string {
	switch r.Type {
	case SignalFrequency:
		return formatFrequencyClipboard(r.Frequency)
	case SignalLifetime:
		return formatLifetimeClipboard(r.Lifetime)
	case SignalBurst:
		return formatBurstClipboard(r.Burst)
	case SignalDiversity:
		return formatDiversityClipboard(r.Diversity)
	}
	return ""
}

func formatFrequencyClipboard(results []FrequencyResult) string {
	if len(results) == 0 {
		return "No errors found"
	}
	var s string
	s = "TOP ERROR SIGNALS\n\n"
	for _, r := range results {
		s += r.Message + " x" + itoa(r.Count) + "\n"
	}
	return s
}

func formatLifetimeClipboard(r *LifetimeResult) string {
	if r == nil {
		return ""
	}
	s := "SIGNAL LIFETIME\n\n"
	s += "Message: " + r.Message + "\n"
	if r.IsSingle {
		s += "Single occurrence\n"
	} else {
		s += "First seen: " + r.FirstSeen + "\n"
		s += "Last seen:  " + r.LastSeen + "\n"
	}
	s += "Occurrences: " + itoa(r.Occurrences) + "\n"
	return s
}

func formatBurstClipboard(r *BurstResult) string {
	if r == nil {
		return ""
	}
	s := "BURST ANALYSIS\n\n"
	s += "Message: " + r.Message + "\n"
	if r.Detected {
		s += "BURST DETECTED\n"
		s += r.Description + "\n"
	} else {
		s += "No abnormal burst detected\n"
	}
	return s
}

func formatDiversityClipboard(r *DiversityResult) string {
	if r == nil {
		return ""
	}
	s := "ERROR DIVERSITY\n\n"
	s += "Total ERROR lines:     " + itoa(r.TotalErrors) + "\n"
	s += "Unique ERROR messages: " + itoa(r.UniqueErrors) + "\n"
	s += "\nSignal quality: " + r.Quality + "\n"
	if r.QualityReason != "" {
		s += r.QualityReason + "\n"
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
