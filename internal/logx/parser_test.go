package logx

import "testing"

func TestParseLineJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantJSON  bool
		wantMsg   string
		wantLevel Level
	}{
		{
			name:      "simple json with msg",
			input:     `{"msg":"hello world","level":"info"}`,
			wantJSON:  true,
			wantMsg:   "hello world",
			wantLevel: LevelInfo,
		},
		{
			name:      "json with message field",
			input:     `{"message":"something happened","severity":"error"}`,
			wantJSON:  true,
			wantMsg:   "something happened",
			wantLevel: LevelError,
		},
		{
			name:      "json with warn level",
			input:     `{"msg":"warning message","level":"warn"}`,
			wantJSON:  true,
			wantMsg:   "warning message",
			wantLevel: LevelWarn,
		},
		{
			name:      "invalid json treated as text",
			input:     `{broken json`,
			wantJSON:  false,
			wantMsg:   `{broken json`,
			wantLevel: LevelUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ParseLine(tt.input, 0)
			if entry.IsJSON != tt.wantJSON {
				t.Errorf("IsJSON = %v, want %v", entry.IsJSON, tt.wantJSON)
			}
			if entry.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", entry.Level, tt.wantLevel)
			}
		})
	}
}

func TestParseLineText(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLevel Level
	}{
		{"error in text", "2024-01-01 ERROR something went wrong", LevelError},
		{"warn in brackets", "[WARN] disk space low", LevelWarn},
		{"info log", "2024-01-01 [INFO] server started", LevelInfo},
		{"debug log", "DEBUG: entering function foo", LevelDebug},
		{"fatal error", "FATAL: cannot connect to database", LevelError},
		{"no level", "just a plain message", LevelUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ParseLine(tt.input, 0)
			if entry.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", entry.Level, tt.wantLevel)
			}
		})
	}
}

func TestTimestampExtraction(t *testing.T) {
	tests := []struct {
		input     string
		wantEmpty bool
	}{
		{"2024-01-15T10:30:45Z ERROR something", false},
		{"2024-01-15 10:30:45 INFO message", false},
		{"just a message", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			entry := ParseLine(tt.input, 0)
			gotEmpty := entry.Timestamp == ""
			if gotEmpty != tt.wantEmpty {
				t.Errorf("Timestamp empty = %v, want %v", gotEmpty, tt.wantEmpty)
			}
		})
	}
}

func TestStackTraceDetection(t *testing.T) {
	tests := []struct {
		input     string
		wantStack bool
	}{
		{"\tat com.example.Foo.bar(Foo.java:123)", true},
		{"main.go:45 +0x123", true},
		{"goroutine 1 [running]:", true},
		{`  File "/app/main.py", line 42, in main`, true},
		{"regular log message", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			entry := ParseLine(tt.input, 0)
			if entry.IsStack != tt.wantStack {
				t.Errorf("IsStack = %v, want %v", entry.IsStack, tt.wantStack)
			}
		})
	}
}

func TestMalformedInput(t *testing.T) {
	inputs := []string{"", "   ", "{}", "{", "}", `{"":}`, string([]byte{0, 1, 2, 3})}

	for _, input := range inputs {
		t.Run("", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ParseLine panicked on input %q: %v", input, r)
				}
			}()
			entry := ParseLine(input, 0)
			if entry.Raw != input {
				t.Errorf("Raw not preserved")
			}
		})
	}
}

func TestParseLines(t *testing.T) {
	lines := []string{
		`{"msg":"first","level":"info"}`,
		"",
		"2024-01-01 ERROR second",
		"",
		"third line",
	}
	entries := ParseLines(lines)
	if len(entries) != 3 {
		t.Errorf("ParseLines returned %d entries, want 3", len(entries))
	}
}
