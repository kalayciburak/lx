package lookup

import "testing"

func TestGetStatus(t *testing.T) {
	tests := []struct {
		code     int
		wantName string
		wantOK   bool
	}{
		{200, "OK", true},
		{404, "Not Found", true},
		{503, "Service Unavailable", true},
		{999, "", false},
	}

	for _, tt := range tests {
		info, ok := GetStatus(tt.code)
		if ok != tt.wantOK {
			t.Errorf("GetStatus(%d) ok = %v, want %v", tt.code, ok, tt.wantOK)
		}
		if ok && info.Name != tt.wantName {
			t.Errorf("GetStatus(%d) name = %q, want %q", tt.code, info.Name, tt.wantName)
		}
	}
}

func TestSearchExactCode(t *testing.T) {
	results := Search("503", 10)
	if len(results) != 1 {
		t.Errorf("Search('503') got %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Code != 503 {
		t.Errorf("Search('503') got code %d, want 503", results[0].Code)
	}
}

func TestSearchPrefix(t *testing.T) {
	tests := []struct {
		query    string
		minCount int
	}{
		{"5", 5},
		{"50", 5},
		{"4", 10},
	}

	for _, tt := range tests {
		results := Search(tt.query, 20)
		if len(results) < tt.minCount {
			t.Errorf("Search(%q) got %d results, want >= %d", tt.query, len(results), tt.minCount)
		}
	}
}

func TestSearchText(t *testing.T) {
	tests := []struct {
		query       string
		expectCodes []int
	}{
		{"gateway", []int{502, 504}},
		{"unauthorized", []int{401}},
		{"timeout", []int{408, 504}},
	}

	for _, tt := range tests {
		results := Search(tt.query, 10)
		if len(results) < len(tt.expectCodes) {
			t.Errorf("Search(%q) got %d results, want >= %d", tt.query, len(results), len(tt.expectCodes))
		}

		for _, code := range tt.expectCodes {
			found := false
			for _, r := range results {
				if r.Code == code {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Search(%q) missing expected code %d", tt.query, code)
			}
		}
	}
}

func TestSearchEmpty(t *testing.T) {
	results := Search("", 10)
	if len(results) != 0 {
		t.Errorf("Search('') should return empty, got %d", len(results))
	}
}

func TestSearchNoMatch(t *testing.T) {
	results := Search("xyz123", 10)
	if len(results) != 0 {
		t.Errorf("Search('xyz123') should return empty, got %d", len(results))
	}
}

func TestExtractHTTPCode(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{"HTTP 503 Service Unavailable", 503},
		{"status: 404 not found", 404},
		{"error 500 occurred", 500},
		{"got 502 from upstream", 502},
		{"no code here", 0},
		{"invalid 999 code", 0},
	}

	for _, tt := range tests {
		got := ExtractHTTPCode(tt.text)
		if got != tt.want {
			t.Errorf("ExtractHTTPCode(%q) = %d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestFormatResult(t *testing.T) {
	info := StatusInfo{503, "Service Unavailable", "Server temporarily overloaded", "Maintenance mode → 503 + Retry-After"}
	result := FormatResult(info)
	expected := "503 Service Unavailable — Server temporarily overloaded"
	if result != expected {
		t.Errorf("FormatResult = %q, want %q", result, expected)
	}
}
