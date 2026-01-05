package app

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/kalayciburak/lx/internal/logx"
)

func ExportLogs(entries []logx.Entry) string {
	var lines []string
	for _, e := range entries {
		lines = append(lines, e.Raw)
	}
	return strings.Join(lines, "\n")
}

func ExportLogsWithNotes(entries []logx.Entry, notes map[int]string, indices []int) string {
	var b strings.Builder

	if len(notes) > 0 {
		b.WriteString("=== NOTES (lx) ===\n")

		var lineNums []int
		for lineNum := range notes {
			lineNums = append(lineNums, lineNum)
		}
		sort.Ints(lineNums)

		for _, lineNum := range lineNums {
			note := notes[lineNum]
			if strings.TrimSpace(note) != "" {
				b.WriteString("• [line ")
				b.WriteString(itoa(lineNum + 1))
				b.WriteString("] ")
				b.WriteString(note)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("=== LOGS (filtered) ===\n")
	for i, e := range entries {
		lineNum := i + 1
		if i < len(indices) {
			lineNum = indices[i] + 1
		}
		b.WriteString("line ")
		b.WriteString(itoa(lineNum))
		b.WriteString(": ")
		b.WriteString(e.Raw)
		b.WriteString("\n")
	}

	return b.String()
}

func ExportNotes(notes string) string {
	return notes
}

func ExportCombined(entries []logx.Entry, notes string) string {
	var b strings.Builder

	b.WriteString("=== LOGS ===\n")
	for _, e := range entries {
		b.WriteString(e.Raw)
		b.WriteString("\n")
	}

	if strings.TrimSpace(notes) != "" {
		b.WriteString("\n=== NOTES ===\n")
		b.WriteString(notes)
		b.WriteString("\n")
	}

	return b.String()
}

func ExportEntry(entry *logx.Entry) string {
	if entry == nil {
		return ""
	}

	if entry.IsJSON && entry.Fields != nil {
		pretty, err := json.MarshalIndent(entry.Fields, "", "  ")
		if err == nil {
			return string(pretty)
		}
	}

	return entry.Raw
}

func ExportEntryRaw(entry *logx.Entry) string {
	if entry == nil {
		return ""
	}
	return entry.Raw
}

func CountExport(count int, what string) string {
	if count == 1 {
		return "Copied 1 " + what
	}
	return "Copied " + itoa(count) + " " + what + "s"
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
