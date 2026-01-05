package logx

import (
	"encoding/json"
	"regexp"
	"strings"
)

var timestampPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?`),
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?`),
	regexp.MustCompile(`^\d{2}/\w{3}/\d{4}:\d{2}:\d{2}:\d{2}`),
	regexp.MustCompile(`^\w{3}\s+\d{1,2} \d{2}:\d{2}:\d{2}`),
}

var stackPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\s*at\s+[\w.$]+\(`),
	regexp.MustCompile(`^\s*\tat\s+`),
	regexp.MustCompile(`goroutine\s+\d+`),
	regexp.MustCompile(`\.go:\d+`),
	regexp.MustCompile(`^\s*File\s+"[^"]+\.py",\s+line\s+\d+`),
	regexp.MustCompile(`^\s+at\s+`),
}

var messageFields = []string{"msg", "message", "log", "error"}

var levelFields = []string{"level", "severity", "lvl", "log_level"}

func ParseLine(raw string, index int) Entry {
	entry := Entry{
		Index: index,
		Raw:   raw,
	}

	trimmed := strings.TrimSpace(raw)

	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		var fields map[string]any
		if err := json.Unmarshal([]byte(trimmed), &fields); err == nil {
			entry.IsJSON = true
			entry.Fields = fields
			entry.Message = extractMessage(fields, trimmed)
			entry.Level = extractLevelJSON(fields)
			entry.Timestamp = extractTimestampJSON(fields)

			if entry.Level == LevelUnknown {
				entry.Level = detectLevelText(entry.Message)
			}
			return entry
		}
	}

	entry.Message = trimmed
	entry.Level = detectLevelText(trimmed)
	entry.Timestamp = extractTimestampText(trimmed)
	entry.IsStack = isStackTrace(trimmed)

	return entry
}

func ParseLines(lines []string) []Entry {
	entries := make([]Entry, 0, len(lines))
	for i, line := range lines {
		if line == "" {
			continue
		}
		entries = append(entries, ParseLine(line, i))
	}
	return entries
}

func extractMessage(fields map[string]any, raw string) string {
	for _, key := range messageFields {
		if val, ok := fields[key]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}
	compact, err := json.Marshal(fields)
	if err != nil {
		return raw
	}
	return string(compact)
}

func extractLevelJSON(fields map[string]any) Level {
	for _, key := range levelFields {
		if val, ok := fields[key]; ok {
			return normalizeLevel(val)
		}
	}
	return LevelUnknown
}

func normalizeLevel(val any) Level {
	var str string
	switch v := val.(type) {
	case string:
		str = strings.ToLower(v)
	case float64:
		switch int(v) {
		case 10:
			return LevelTrace
		case 20:
			return LevelDebug
		case 30:
			return LevelInfo
		case 40:
			return LevelWarn
		case 50, 60:
			return LevelError
		}
		return LevelUnknown
	default:
		return LevelUnknown
	}

	switch str {
	case "error", "err", "fatal", "panic", "critical", "crit":
		return LevelError
	case "warn", "warning":
		return LevelWarn
	case "info", "information":
		return LevelInfo
	case "debug":
		return LevelDebug
	case "trace", "verbose":
		return LevelTrace
	default:
		return LevelUnknown
	}
}

func detectLevelText(text string) Level {
	upper := strings.ToUpper(text)

	if containsAny(upper, "ERROR", "ERR]", "[ERR", "FATAL", "PANIC", "CRITICAL") {
		return LevelError
	}
	if containsAny(upper, "WARN", "WARNING") {
		return LevelWarn
	}
	if containsAny(upper, "[INFO", "INFO]", " INFO ") {
		return LevelInfo
	}
	if containsAny(upper, "DEBUG", "[DBG", "DBG]") {
		return LevelDebug
	}
	if containsAny(upper, "TRACE", "VERBOSE") {
		return LevelTrace
	}
	return LevelUnknown
}

func containsAny(text string, substrs ...string) bool {
	for _, s := range substrs {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}

func extractTimestampJSON(fields map[string]any) string {
	timestampKeys := []string{"timestamp", "time", "ts", "@timestamp", "datetime", "date"}
	for _, key := range timestampKeys {
		if val, ok := fields[key]; ok {
			if v, ok := val.(string); ok {
				return v
			}
		}
	}
	return ""
}

func extractTimestampText(text string) string {
	for _, pattern := range timestampPatterns {
		if match := pattern.FindString(text); match != "" {
			return match
		}
	}
	return ""
}

func isStackTrace(text string) bool {
	for _, pattern := range stackPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}
