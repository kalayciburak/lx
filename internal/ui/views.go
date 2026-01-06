package ui

import (
	"encoding/json"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kalayciburak/lx/internal/app"
	"github.com/kalayciburak/lx/internal/logx"
	"github.com/kalayciburak/lx/internal/lookup"
	"github.com/kalayciburak/lx/internal/signal"
)

func RenderTitleBar(s *app.State, width int) string {
	var parts []string

	parts = append(parts, StyleBarHighlight.Render("lx by kalayciburak"))

	sep := StyleBarDim.Render(" │ ")

	if s.FileName != "" {
		parts = append(parts, StyleBarHighlight.Render(Truncate(s.FileName, 25)))
	}

	counts := StyleBarAccent.Render(Itoa(len(s.Filtered))) +
		StyleBarDim.Render("/"+Itoa(len(s.Entries)))
	parts = append(parts, counts)

	if s.TotalNotes() > 0 {
		parts = append(parts, StyleBarAccent.Render("≡")+StyleBarText.Render(" "+Itoa(s.TotalNotes())))
	}

	left := strings.Join(parts, sep)

	var right string
	if s.LevelFilter != app.LevelFilterAll {
		right = StyleBarAccent.Render("◉ ") + StyleBarHighlight.Render(s.LevelFilter.String())
	}
	if s.FilterQuery != "" {
		right += StyleBarAccent.Render("⚡") + StyleBarText.Render(s.FilterQuery)
	}
	if s.StatusMsg != "" {
		if right != "" {
			right += StyleBarDim.Render("  │  ")
		}
		right += StyleBarAccent.Render("✓ ") + StyleBarHighlight.Render(s.StatusMsg)
	}

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	padding := width - leftW - rightW
	if padding < 1 {
		padding = 1
	}

	content := left + StyleBar.Render(strings.Repeat(" ", padding)) + right
	contentW := lipgloss.Width(content)
	if contentW < width {
		content += StyleBar.Render(strings.Repeat(" ", width-contentW))
	}

	return content
}

func RenderList(s *app.State, height, width int) string {
	if len(s.Filtered) == 0 {
		return RenderEmpty(height, width)
	}

	var lines []string

	maxNum := len(s.Entries)
	lineNumW := len(Itoa(maxNum)) + 1
	if lineNumW < 4 {
		lineNumW = 4
	}

	start := 0
	if s.Cursor >= height {
		start = s.Cursor - height + 1
	}
	end := start + height
	if end > len(s.Filtered) {
		end = len(s.Filtered)
	}

	for i := start; i < end && len(lines) < height; i++ {
		entryIdx := s.Filtered[i]
		entry := s.Entries[entryIdx]
		isSelected := i == s.Cursor
		hasNote := s.HasNote(entryIdx)

		if s.IsNoteShowing(entryIdx) && hasNote && len(lines) < height-1 {
			note := s.GetNote(entryIdx)
			noteBox := RenderInlineNoteBox(note, width)
			noteLines := strings.Split(noteBox, "\n")
			for _, noteLine := range noteLines {
				if len(lines) < height-1 {
					lines = append(lines, noteLine)
				}
			}
		}

		if len(lines) < height {
			line := RenderListLine(&entry, entryIdx+1, lineNumW, width, isSelected, hasNote)
			lines = append(lines, line)
		}
	}

	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func RenderListLine(entry *logx.Entry, lineNum, lineNumW, width int, selected, hasNote bool) string {
	var parts []string

	bg := lipgloss.NewStyle()
	if selected {
		bg = bg.Background(ColorBgSelect)
	}

	noteIndicator := bg.Render("  ")
	if hasNote {
		noteIndicator = StyleNoteIndicator.Copy().Background(ColorBgSelect).Render("≡") + bg.Render(" ")
		if !selected {
			noteIndicator = StyleNoteIndicator.Render("≡") + " "
		}
	}

	numStr := PadLeft(Itoa(lineNum), lineNumW-1)
	if selected {
		parts = append(parts, noteIndicator+StyleLineNumSelected.Copy().Background(ColorBgSelect).Render(numStr))
	} else {
		parts = append(parts, noteIndicator+StyleLineNum.Render(numStr))
	}

	if selected {
		parts = append(parts, StyleCursorIndicator.Copy().Background(ColorBgSelect).Render(">"))
	} else {
		parts = append(parts, " ")
	}

	levelStr := PadCenter(entry.Level.String(), 7)
	parts = append(parts, LevelStyle(entry.Level).Render(levelStr))

	if entry.Timestamp != "" {
		if selected {
			parts = append(parts, StyleTimestamp.Copy().Background(ColorBgSelect).Render(Truncate(entry.Timestamp, 19)))
		} else {
			parts = append(parts, StyleTimestamp.Render(Truncate(entry.Timestamp, 19)))
		}
	}

	usedW := 0
	for _, p := range parts {
		usedW += lipgloss.Width(p) + 1
	}
	msgW := width - usedW - 2
	if msgW < 10 {
		msgW = 10
	}

	msg := Truncate(entry.Message, msgW)
	if selected {
		styledMsg := StyleMessage.Copy().Background(ColorBgSelect).Render(msg)
		parts = append(parts, styledMsg)
	} else {
		styledMsg := RenderLxFormat(msg, entry.IsStack)
		parts = append(parts, styledMsg)
	}

	line := strings.Join(parts, bg.Render(" "))

	if selected {
		lineW := lipgloss.Width(line)
		if lineW < width {
			line += bg.Render(strings.Repeat(" ", width-lineW))
		}
	}

	return line
}

func RenderLxFormat(msg string, isStack bool) string {
	if strings.Contains(msg, "=== NOTE") || strings.HasPrefix(msg, "=== NOTE") {
		return StyleNotesHeader.Render(msg)
	}
	if strings.Contains(msg, "=== LOG") || strings.HasPrefix(msg, "=== LOG") {
		return StyleDetailHeader.Render(msg)
	}
	if strings.HasPrefix(msg, "• [line") {
		if idx := strings.Index(msg, "] "); idx != -1 {
			prefix := msg[:idx+1]
			content := msg[idx+1:]
			return StyleNoteIndicator.Render(prefix) + StyleMessage.Render(content)
		}
		return StyleNoteIndicator.Render(msg)
	}

	if isStack {
		return StyleStack.Render(msg)
	}
	return StyleMessage.Render(msg)
}

func RenderInlineNoteBox(note string, width int) string {
	noteVisualLen := lipgloss.Width(note)
	boxW := noteVisualLen + 6

	minW := 20
	maxW := width - 8
	if maxW > 70 {
		maxW = 70
	}
	if boxW < minW {
		boxW = minW
	}
	if boxW > maxW {
		boxW = maxW
	}

	contentW := boxW - 4

	noteRunes := []rune(note)
	var noteLines []string
	if len(noteRunes) <= contentW {
		noteLines = []string{note}
	} else {
		for len(noteRunes) > 0 {
			if len(noteRunes) <= contentW {
				noteLines = append(noteLines, string(noteRunes))
				break
			}
			breakAt := contentW
			for i := contentW; i > contentW/2; i-- {
				if noteRunes[i] == ' ' {
					breakAt = i
					break
				}
			}
			noteLines = append(noteLines, string(noteRunes[:breakAt]))
			noteRunes = []rune(strings.TrimLeft(string(noteRunes[breakAt:]), " "))
		}
	}

	if len(noteLines) > 4 {
		noteLines = noteLines[:4]
		lastRunes := []rune(noteLines[3])
		if len(lastRunes) > 3 {
			noteLines[3] = string(lastRunes[:len(lastRunes)-3]) + "..."
		}
	}

	headerText := " NOT "
	leftPad := 1
	rightPad := boxW - 2 - len(headerText) - leftPad
	if rightPad < 0 {
		rightPad = 0
	}

	var result []string

	topLine := "   " + StyleNoteBoxBorder.Render("╭"+strings.Repeat("─", leftPad)) +
		StyleNotesHeader.Render(headerText) +
		StyleNoteBoxBorder.Render(strings.Repeat("─", rightPad)+"╮")
	result = append(result, topLine)

	for _, line := range noteLines {
		lineVisualW := lipgloss.Width(line)
		pad := contentW - lineVisualW
		if pad < 0 {
			pad = 0
		}
		contentLine := "   " + StyleNoteBoxBorder.Render("│") +
			StyleNoteBox.Render(" "+line+strings.Repeat(" ", pad)+" ") +
			StyleNoteBoxBorder.Render("│")
		result = append(result, contentLine)
	}

	bottomLine := "   " + StyleNoteBoxBorder.Render("╰"+strings.Repeat("─", boxW-2)+"╯")
	result = append(result, bottomLine)

	return strings.Join(result, "\n")
}

func RenderEmpty(height, width int) string {
	modalW := 44

	makeLine := func(content string) string {
		contentW := lipgloss.Width(content)
		pad := modalW - 2 - contentW
		if pad < 0 {
			pad = 0
		}
		return StyleFrameBorder.Render("│") + content + strings.Repeat(" ", pad) + StyleFrameBorder.Render("│")
	}

	var lines []string

	lines = append(lines, StyleFrameBorder.Render("╭"+strings.Repeat("─", modalW-2)+"╮"))

	lines = append(lines, makeLine(""))

	logoText := "★ LX"
	logoPad := (modalW - 2 - lipgloss.Width(logoText)) / 2
	lines = append(lines, makeLine(strings.Repeat(" ", logoPad)+StyleDetailHeader.Render(logoText)))

	subtitle := "Log X-Ray Viewer"
	subPad := (modalW - 2 - len(subtitle)) / 2
	lines = append(lines, makeLine(strings.Repeat(" ", subPad)+StyleHelpDesc.Render(subtitle)))

	lines = append(lines, makeLine(""))

	lines = append(lines, StyleFrameBorder.Render("├"+strings.Repeat("─", modalW-2)+"┤"))

	lines = append(lines, makeLine(""))

	helpItems := []struct {
		key  string
		desc string
	}{
		{"p", "paste from clipboard"},
		{"lx <file>", "open log file"},
		{"cmd | lx", "pipe from command"},
	}

	for _, item := range helpItems {
		keyPart := StyleHelpKey.Render(PadRight(item.key, 12))
		descPart := StyleHelpDesc.Render(item.desc)
		line := " " + keyPart + descPart
		lines = append(lines, makeLine(line))
	}

	lines = append(lines, makeLine(""))

	lines = append(lines, StyleFrameBorder.Render("╰"+strings.Repeat("─", modalW-2)+"╯"))

	modalH := len(lines)
	padTop := (height - modalH) / 2
	if padTop < 0 {
		padTop = 0
	}
	padLeft := (width - modalW) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	var result []string

	for i := 0; i < padTop; i++ {
		result = append(result, "")
	}
	for _, line := range lines {
		result = append(result, strings.Repeat(" ", padLeft)+line)
	}
	for len(result) < height {
		result = append(result, "")
	}

	return strings.Join(result, "\n")
}

func CountDetailLines(entry *logx.Entry, width int) int {
	if entry == nil {
		return 0
	}

	lines := 2

	if entry.IsJSON && entry.Fields != nil {
		keyFields := []string{"msg", "message", "level", "error", "timestamp"}
		for _, k := range keyFields {
			if _, ok := entry.Fields[k]; ok {
				lines++
			}
		}
		remaining := len(entry.Fields) - lines + 2
		if remaining > 0 {
			lines += Min(remaining*2, 6)
		}
	} else {
		if entry.Timestamp != "" {
			lines++
		}
		if entry.Level != logx.LevelUnknown {
			lines++
		}
		rawLines := (len(entry.Raw) / (width - 4)) + 1
		lines += Min(rawLines, 8)
	}

	return lines
}

func RenderDetail(entry *logx.Entry, height, width int) string {
	var lines []string

	if entry == nil {
		for i := 0; i < height; i++ {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	headerText := "─── DETAIL ───"
	headerPad := (width - len(headerText)) / 2
	if headerPad < 0 {
		headerPad = 0
	}
	lines = append(lines, strings.Repeat(" ", headerPad)+StyleDetailHeader.Render(headerText))

	contentH := height - 1
	var contentLines []string
	if entry.IsJSON && entry.Fields != nil {
		contentLines = renderJSONDetailLines(entry, width-2, contentH)
	} else {
		contentLines = renderTextDetailLines(entry, width-2, contentH)
	}
	lines = append(lines, contentLines...)

	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

func renderJSONDetailLines(entry *logx.Entry, width, maxLines int) []string {
	var lines []string

	keyFields := []string{"msg", "message", "level", "severity", "error", "timestamp", "time"}
	shown := make(map[string]bool)

	for _, key := range keyFields {
		if val, ok := entry.Fields[key]; ok {
			line := " " + StyleDetailLabel.Render(key+":") + " " + StyleDetailValue.Render(formatValue(val))
			lines = append(lines, Truncate(line, width))
			shown[key] = true
			if len(lines) >= maxLines-1 {
				break
			}
		}
	}

	if len(lines) < maxLines-1 {
		remaining := make(map[string]any)
		for k, v := range entry.Fields {
			if !shown[k] {
				remaining[k] = v
			}
		}

		if len(remaining) > 0 {
			lines = append(lines, " "+StyleDetailDim.Render("───"))

			pretty, _ := json.MarshalIndent(remaining, " ", "  ")
			highlighted := HighlightJSON(string(pretty))
			jsonLines := strings.Split(highlighted, "\n")

			for _, line := range jsonLines {
				if len(lines) >= maxLines-1 {
					lines = append(lines, " "+StyleDetailDim.Render("..."))
					break
				}
				if lipgloss.Width(line) > width {
					line = line[:width-3] + "..."
				}
				lines = append(lines, line)
			}
		}
	}

	return lines
}

func renderTextDetailLines(entry *logx.Entry, width, maxLines int) []string {
	var lines []string

	if entry.Timestamp != "" {
		lines = append(lines, " "+StyleDetailLabel.Render("Time:")+" "+StyleDetailValue.Render(entry.Timestamp))
	}
	if entry.Level != logx.LevelUnknown {
		lines = append(lines, " "+StyleDetailLabel.Render("Level:")+" "+LevelStyle(entry.Level).Render(" "+entry.Level.String()+" "))
	}

	if len(lines) < maxLines-1 {
		lines = append(lines, " "+StyleDetailDim.Render("───"))

		wrapped := WordWrap(entry.Raw, width-2)
		rawLines := strings.Split(wrapped, "\n")

		for _, line := range rawLines {
			if len(lines) >= maxLines-1 {
				lines = append(lines, " "+StyleDetailDim.Render("..."))
				break
			}
			lines = append(lines, " "+StyleDetailValue.Render(line))
		}
	}

	return lines
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int(val)) {
			return Itoa(int(val))
		}
		data, _ := json.Marshal(val)
		return string(data)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		data, _ := json.Marshal(val)
		return string(data)
	}
}

func HighlightJSON(s string) string {
	var b strings.Builder
	inString := false
	stringStart := -1

	for i := 0; i < len(s); i++ {
		c := s[i]

		switch {
		case c == '"':
			if i > 0 && s[i-1] == '\\' {
				continue
			}

			if !inString {
				inString = true
				stringStart = i
			} else {
				inString = false
				content := s[stringStart : i+1]

				isKey := false
				for j := i + 1; j < len(s); j++ {
					if s[j] == ':' {
						isKey = true
						break
					}
					if s[j] != ' ' && s[j] != '\t' && s[j] != '\n' {
						break
					}
				}

				if isKey {
					b.WriteString(StyleJSONKey.Render(content))
				} else {
					b.WriteString(StyleJSONString.Render(content))
				}
				stringStart = -1
				continue
			}

		case inString:
			continue

		case c >= '0' && c <= '9' || (c == '-' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9'):
			j := i
			for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == '.' || s[j] == '-' || s[j] == 'e' || s[j] == 'E' || s[j] == '+') {
				j++
			}
			b.WriteString(StyleJSONNumber.Render(s[i:j]))
			i = j - 1

		case c == 't' && i+4 <= len(s) && s[i:i+4] == "true":
			b.WriteString(StyleJSONBool.Render("true"))
			i += 3

		case c == 'f' && i+5 <= len(s) && s[i:i+5] == "false":
			b.WriteString(StyleJSONBool.Render("false"))
			i += 4

		case c == 'n' && i+4 <= len(s) && s[i:i+4] == "null":
			b.WriteString(StyleJSONNull.Render("null"))
			i += 3

		default:
			b.WriteByte(c)
		}
	}

	return b.String()
}

func RenderNotesModal(note string, lineNum, height, width int) string {
	modalW := 50
	if modalW > width-8 {
		modalW = width - 8
	}
	modalH := 7

	var content strings.Builder

	headerText := " NOTE FOR LINE " + Itoa(lineNum) + " "
	leftPad := (modalW - 2 - len(headerText)) / 2
	rightPad := modalW - 2 - len(headerText) - leftPad
	if leftPad < 0 {
		leftPad = 0
	}
	if rightPad < 0 {
		rightPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("╭" + strings.Repeat("─", leftPad)) + StyleNotesHeader.Render(headerText) + StyleFrameBorder.Render(strings.Repeat("─", rightPad) + "╮") + "\n")

	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")

	maxW := modalW - 7
	displayNote := note
	if len(note) > maxW {
		displayNote = "…" + note[len(note)-maxW+1:]
	}
	inputLine := displayNote + StyleCursorIndicator.Render("█")
	inputPad := modalW - 4 - lipgloss.Width(inputLine)
	if inputPad < 0 {
		inputPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + " " + StyleNotesInput.Render(inputLine) + strings.Repeat(" ", inputPad) + " " + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("├" + strings.Repeat("─", modalW-2) + "┤") + "\n")

	hints := StyleHelpKey.Render("Enter") + StyleFooter.Render(" save  ") +
		StyleHelpKey.Render("ESC") + StyleFooter.Render(" cancel")
	hintsW := lipgloss.Width(hints)
	hintsPad := (modalW - 2 - hintsW) / 2
	if hintsPad < 0 {
		hintsPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", hintsPad) + hints + strings.Repeat(" ", modalW-2-hintsPad-hintsW) + StyleFrameBorder.Render("│") + "\n")
	content.WriteString(StyleFrameBorder.Render("╰" + strings.Repeat("─", modalW-2) + "╯"))

	modalLines := strings.Split(content.String(), "\n")
	padTop := (height - modalH) / 2
	if padTop < 0 {
		padTop = 0
	}
	padLeft := (width - modalW) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	var result strings.Builder
	for i := 0; i < padTop; i++ {
		result.WriteString(strings.Repeat(" ", width) + "\n")
	}
	for _, line := range modalLines {
		result.WriteString(strings.Repeat(" ", padLeft) + line + "\n")
	}

	return result.String()
}

func RenderLookupModal(query string, results []lookup.StatusInfo, cursor, height, width int) string {
	modalW := 60
	if modalW > width-4 {
		modalW = width - 4
	}
	if modalW < 40 {
		modalW = 40
	}
	innerW := modalW - 4

	maxResults := 5
	if len(results) < maxResults {
		maxResults = len(results)
	}
	if maxResults < 1 {
		maxResults = 1
	}
	modalH := maxResults + 8

	var content strings.Builder

	headerText := " HTTP STATUS LOOKUP "
	headerPadTotal := modalW - 2 - len(headerText)
	leftPad := headerPadTotal / 2
	rightPad := headerPadTotal - leftPad
	if leftPad < 0 {
		leftPad = 0
	}
	if rightPad < 0 {
		rightPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("╭"+strings.Repeat("─", leftPad)) + StyleDetailHeader.Render(headerText) + StyleFrameBorder.Render(strings.Repeat("─", rightPad)+"╮") + "\n")

	inputText := "> " + query + "█"
	inputLine := StyleLookupInput.Render(inputText)
	inputW := lipgloss.Width(inputLine)
	inputPad := innerW - inputW
	if inputPad < 0 {
		inputPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + " " + inputLine + strings.Repeat(" ", inputPad) + " " + StyleFrameBorder.Render("│") + "\n")
	content.WriteString(StyleFrameBorder.Render("├"+strings.Repeat("─", modalW-2)+"┤") + "\n")

	if len(results) == 0 {
		hint := "Type code (503) or text (gateway)"
		if query != "" {
			hint = "No results found"
		}
		hintPad := (innerW - len(hint)) / 2
		if hintPad < 0 {
			hintPad = 0
		}
		rightHintPad := innerW - hintPad - len(hint)
		if rightHintPad < 0 {
			rightHintPad = 0
		}
		content.WriteString(StyleFrameBorder.Render("│") + " " + strings.Repeat(" ", hintPad) + StyleEmpty.Render(hint) + strings.Repeat(" ", rightHintPad) + " " + StyleFrameBorder.Render("│") + "\n")
	} else {
		start := 0
		if cursor >= maxResults {
			start = cursor - maxResults + 1
		}
		end := start + maxResults
		if end > len(results) {
			end = len(results)
		}

		for i := start; i < end; i++ {
			info := results[i]
			line := Itoa(info.Code) + " " + info.Name + " - " + info.Description
			if len(line) > innerW {
				line = line[:innerW-3] + "..."
			}
			linePad := innerW - len(line)
			if linePad < 0 {
				linePad = 0
			}
			if i == cursor {
				fullLine := line + strings.Repeat(" ", linePad)
				content.WriteString(StyleFrameBorder.Render("│") + " " + StyleLookupSelected.Render(fullLine) + " " + StyleFrameBorder.Render("│") + "\n")
			} else {
				content.WriteString(StyleFrameBorder.Render("│") + " " + StyleLookupResult.Render(line) + strings.Repeat(" ", linePad) + " " + StyleFrameBorder.Render("│") + "\n")
			}
		}

		if cursor >= 0 && cursor < len(results) {
			content.WriteString(StyleFrameBorder.Render("├"+strings.Repeat("─", modalW-2)+"┤") + "\n")
			example := results[cursor].Example
			if len(example) > innerW-2 {
				example = example[:innerW-5] + "..."
			}
			exLine := StyleDetailLabel.Render("→ ") + StyleDetailValue.Render(example)
			exLineW := lipgloss.Width(exLine)
			exPad := innerW - exLineW
			if exPad < 0 {
				exPad = 0
			}
			content.WriteString(StyleFrameBorder.Render("│") + " " + exLine + strings.Repeat(" ", exPad) + " " + StyleFrameBorder.Render("│") + "\n")
		}
	}

	content.WriteString(StyleFrameBorder.Render("╰"+strings.Repeat("─", modalW-2)+"╯"))

	modalLines := strings.Split(content.String(), "\n")
	padTop := (height - modalH) / 2
	if padTop < 0 {
		padTop = 0
	}
	padLeft := (width - modalW) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	var result strings.Builder
	for i := 0; i < padTop; i++ {
		result.WriteString(strings.Repeat(" ", width) + "\n")
	}
	for _, line := range modalLines {
		result.WriteString(strings.Repeat(" ", padLeft) + line + "\n")
	}

	return result.String()
}

func RenderFilterModal(query string, levelFilter int, height, width int) string {
	modalW := 50
	if modalW > width-8 {
		modalW = width - 8
	}
	if modalW < 40 {
		modalW = 40
	}
	innerW := modalW - 4

	pad := func(n int) string {
		if n <= 0 {
			return ""
		}
		return StyleModalInner.Render(strings.Repeat(" ", n))
	}

	var content strings.Builder

	headerText := " FILTER "
	headerPadTotal := modalW - 2 - len(headerText)
	leftPad := headerPadTotal / 2
	rightPad := headerPadTotal - leftPad
	if leftPad < 0 {
		leftPad = 0
	}
	if rightPad < 0 {
		rightPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("╭"+strings.Repeat("─", leftPad)) + StyleDetailHeader.Render(headerText) + StyleFrameBorder.Render(strings.Repeat("─", rightPad)+"╮") + "\n")

	content.WriteString(StyleFrameBorder.Render("│") + pad(modalW-2) + StyleFrameBorder.Render("│") + "\n")

	levelFilters := []string{"ALL", "ERROR", "WARN", "INFO", "DEBUG"}
	var levelParts []string
	for i, lf := range levelFilters {
		if i == levelFilter {
			levelParts = append(levelParts, StyleModalHighlight.Render("["+lf+"]"))
		} else {
			levelParts = append(levelParts, StyleModalDim.Render(" "+lf+" "))
		}
	}
	levelLine := strings.Join(levelParts, "")
	levelLineW := lipgloss.Width(levelLine)
	levelPadW := (innerW - levelLineW) / 2
	if levelPadW < 0 {
		levelPadW = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + pad(1) + pad(levelPadW) + levelLine + pad(innerW-levelPadW-levelLineW) + pad(1) + StyleFrameBorder.Render("│") + "\n")

	tabHint := StyleModalDim.Render("Tab: cycle levels")
	tabHintW := lipgloss.Width(tabHint)
	tabPadW := (innerW - tabHintW) / 2
	content.WriteString(StyleFrameBorder.Render("│") + pad(1) + pad(tabPadW) + tabHint + pad(innerW-tabPadW-tabHintW) + pad(1) + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("│") + pad(modalW-2) + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("├" + strings.Repeat("─", modalW-2) + "┤") + "\n")

	searchLabel := " Text Filter "
	content.WriteString(StyleFrameBorder.Render("│") + StyleModalText.Render(searchLabel) + pad(innerW-len(searchLabel)+1) + pad(1) + StyleFrameBorder.Render("│") + "\n")

	displayQuery := query
	maxQueryW := innerW - 4
	if len(query) > maxQueryW {
		displayQuery = "…" + query[len(query)-maxQueryW+1:]
	}
	inputLine := StyleModalAccent.Render("> ") + StyleModalHighlight.Render(displayQuery) + StyleCursorIndicator.Render("█")
	inputLineW := lipgloss.Width(inputLine)
	inputPadW := innerW - inputLineW
	if inputPadW < 0 {
		inputPadW = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + pad(1) + inputLine + pad(inputPadW) + pad(1) + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("│") + pad(modalW-2) + StyleFrameBorder.Render("│") + "\n")

	syntaxHints := []struct {
		example string
		desc    string
	}{
		{"error", "contains 'error'"},
		{"!debug", "exclude 'debug'"},
		{"api timeout", "both terms (AND)"},
	}
	for _, hint := range syntaxHints {
		hintLine := StyleModalHighlight.Render(PadRight(hint.example, 12)) + StyleModalText.Render(hint.desc)
		hintLineW := lipgloss.Width(hintLine)
		hintPadW := innerW - hintLineW
		if hintPadW < 0 {
			hintPadW = 0
		}
		content.WriteString(StyleFrameBorder.Render("│") + pad(1) + hintLine + pad(hintPadW) + pad(1) + StyleFrameBorder.Render("│") + "\n")
	}

	content.WriteString(StyleFrameBorder.Render("├" + strings.Repeat("─", modalW-2) + "┤") + "\n")

	hints := StyleModalHighlight.Render("Enter") + StyleModalDim.Render(" apply  ") +
		StyleModalHighlight.Render("ESC") + StyleModalDim.Render(" cancel")
	hintsW := lipgloss.Width(hints)
	hintsPadW := (modalW - 2 - hintsW) / 2
	if hintsPadW < 0 {
		hintsPadW = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + pad(hintsPadW) + hints + pad(modalW-2-hintsPadW-hintsW) + StyleFrameBorder.Render("│") + "\n")
	content.WriteString(StyleFrameBorder.Render("╰" + strings.Repeat("─", modalW-2) + "╯"))

	modalLines := strings.Split(content.String(), "\n")
	modalH := len(modalLines)
	padTop := (height - modalH) / 2
	if padTop < 0 {
		padTop = 0
	}
	padLeft := (width - modalW) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	var result strings.Builder
	for i := 0; i < padTop; i++ {
		result.WriteString(strings.Repeat(" ", width) + "\n")
	}
	for _, line := range modalLines {
		result.WriteString(strings.Repeat(" ", padLeft) + line + "\n")
	}

	return result.String()
}

func RenderHelp(height, width int) string {
	rows := [][]string{
		{"j/k", "navigate", "1", "frequency"},
		{"g/G", "top/bottom", "2", "lifetime"},
		{"/", "filter", "3", "burst"},
		{"Tab", "level filter", "4", "diversity"},
		{"ESC", "clear", "", ""},
		{"", "", "", ""},
		{"N", "write note", "c", "copy line"},
		{"n", "toggle note", "y", "copy all"},
		{"m", "show all", "^L", "HTTP lookup"},
		{"]/[", "jump notes", "?", "help"},
		{"", "", "", ""},
		{"Enter", " detail", "q", "quit"},
	}

	modalW := 46
	innerW := modalW - 4

	var content strings.Builder

	headerText := " HELP "
	headerPadTotal := modalW - 2 - len(headerText)
	leftPad := headerPadTotal / 2
	rightPad := headerPadTotal - leftPad
	content.WriteString(StyleFrameBorder.Render("╭"+strings.Repeat("─", leftPad)) + StyleDetailHeader.Render(headerText) + StyleFrameBorder.Render(strings.Repeat("─", rightPad)+"╮") + "\n")

	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")

	for _, row := range rows {
		if row[0] == "" && row[2] == "" {
			content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")
			continue
		}

		col1Key := StyleHelpKey.Render(PadRight(row[0], 5))
		col1Desc := StyleHelpDesc.Render(PadRight(row[1], 12))
		col2Key := StyleHelpKey.Render(PadRight(row[2], 4))
		col2Desc := StyleHelpDesc.Render(row[3])

		line := "  " + col1Key + col1Desc + "  " + col2Key + col2Desc
		lineW := lipgloss.Width(line)
		pad := innerW - lineW
		if pad < 0 {
			pad = 0
		}

		content.WriteString(StyleFrameBorder.Render("│") + " " + line + strings.Repeat(" ", pad) + " " + StyleFrameBorder.Render("│") + "\n")
	}

	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")

	footer := StyleHelpDesc.Render("lx by kalayciburak")
	footerW := lipgloss.Width(footer)
	footerPad := (innerW - footerW) / 2
	content.WriteString(StyleFrameBorder.Render("│") + " " + strings.Repeat(" ", footerPad) + footer + strings.Repeat(" ", innerW-footerPad-footerW) + " " + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("╰" + strings.Repeat("─", modalW-2) + "╯"))

	modalLines := strings.Split(content.String(), "\n")
	modalH := len(modalLines)
	padTop := (height - modalH) / 2
	if padTop < 0 {
		padTop = 0
	}
	padLeft := (width - modalW) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	var result []string

	for i := 0; i < padTop; i++ {
		result = append(result, "")
	}
	for _, line := range modalLines {
		result = append(result, strings.Repeat(" ", padLeft)+line)
	}
	for len(result) < height {
		result = append(result, "")
	}

	return strings.Join(result, "\n")
}

func RenderFooter(mode int, statusMsg string, width int) string {
	var hintParts []string

	switch app.Mode(mode) {
	case app.ModeFilter:
		hintParts = append(hintParts,
			StyleBarAccent.Render("ESC")+StyleBarText.Render(" cancel"),
			StyleBarAccent.Render("Enter")+StyleBarText.Render(" apply"))
	case app.ModeDetail:
		hintParts = append(hintParts,
			StyleBarAccent.Render("ESC")+StyleBarText.Render(" back"),
			StyleBarAccent.Render("c")+StyleBarText.Render(" copy"),
			StyleBarAccent.Render("N")+StyleBarText.Render(" note"),
			StyleBarAccent.Render("n")+StyleBarText.Render(" read"),
			StyleBarAccent.Render("m")+StyleBarText.Render(" all"))
	case app.ModeNotes:
		hintParts = append(hintParts,
			StyleBarAccent.Render("Enter")+StyleBarText.Render(" save"),
			StyleBarAccent.Render("ESC")+StyleBarText.Render(" cancel"))
	case app.ModeLookup:
		hintParts = append(hintParts,
			StyleBarAccent.Render("ESC")+StyleBarText.Render(" close"),
			StyleBarAccent.Render("j/k")+StyleBarText.Render(" select"),
			StyleBarAccent.Render("Enter")+StyleBarText.Render(" copy"))
	case app.ModeSignal:
		hintParts = append(hintParts,
			StyleBarAccent.Render("c")+StyleBarText.Render(" copy"),
			StyleBarAccent.Render("ESC")+StyleBarText.Render(" close"))
	default:
		hintParts = append(hintParts,
			StyleBarAccent.Render("j/k")+StyleBarText.Render(" nav"),
			StyleBarAccent.Render("/")+StyleBarText.Render(" filter"),
			StyleBarAccent.Render("y")+StyleBarText.Render(" yank"),
			StyleBarAccent.Render("c")+StyleBarText.Render(" copy"),
			StyleBarAccent.Render("N")+StyleBarText.Render(" note"),
			StyleBarAccent.Render("m")+StyleBarText.Render(" all"),
			StyleBarAccent.Render("?")+StyleBarText.Render(" help"))
	}

	sep := StyleBarDim.Render(" · ")
	hints := strings.Join(hintParts, sep)

	credit := StyleBarDim.Render("by ") + StyleBarText.Render("kalayciburak")

	leftW := lipgloss.Width(hints)
	rightW := lipgloss.Width(credit)
	padding := width - leftW - rightW
	if padding < 1 {
		padding = 1
	}

	content := hints + StyleBar.Render(strings.Repeat(" ", padding)) + credit
	contentW := lipgloss.Width(content)
	if contentW < width {
		content += StyleBar.Render(strings.Repeat(" ", width-contentW))
	}

	return content
}

func RenderSignalModal(result *signal.SignalResult, height, width int) string {
	if result == nil {
		return ""
	}

	modalW := 55
	if modalW > width-4 {
		modalW = width - 4
	}
	if modalW < 40 {
		modalW = 40
	}
	innerW := modalW - 4

	var content strings.Builder

	headerText := " SIGNAL BOOSTER — " + result.Title + " "
	headerPadTotal := modalW - 2 - len(headerText)
	leftPad := headerPadTotal / 2
	rightPad := headerPadTotal - leftPad
	if leftPad < 0 {
		leftPad = 0
	}
	if rightPad < 0 {
		rightPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("╭"+strings.Repeat("─", leftPad)) + StyleDetailHeader.Render(headerText) + StyleFrameBorder.Render(strings.Repeat("─", rightPad)+"╮") + "\n")

	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")

	var contentLines []string
	switch result.Type {
	case signal.SignalFrequency:
		contentLines = renderFrequencyContent(result.Frequency, innerW)
	case signal.SignalLifetime:
		contentLines = renderLifetimeContent(result.Lifetime, innerW)
	case signal.SignalBurst:
		contentLines = renderBurstContent(result.Burst, innerW)
	case signal.SignalDiversity:
		contentLines = renderDiversityContent(result.Diversity, innerW)
	}

	for _, line := range contentLines {
		lineW := lipgloss.Width(line)
		pad := innerW - lineW
		if pad < 0 {
			pad = 0
		}
		content.WriteString(StyleFrameBorder.Render("│") + " " + line + strings.Repeat(" ", pad) + " " + StyleFrameBorder.Render("│") + "\n")
	}

	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("├" + strings.Repeat("─", modalW-2) + "┤") + "\n")

	hints := StyleHelpKey.Render("c") + StyleFooter.Render(" copy  ") +
		StyleHelpKey.Render("ESC") + StyleFooter.Render(" close")
	hintsW := lipgloss.Width(hints)
	hintsPad := (modalW - 2 - hintsW) / 2
	if hintsPad < 0 {
		hintsPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", hintsPad) + hints + strings.Repeat(" ", modalW-2-hintsPad-hintsW) + StyleFrameBorder.Render("│") + "\n")
	content.WriteString(StyleFrameBorder.Render("╰" + strings.Repeat("─", modalW-2) + "╯"))

	modalLines := strings.Split(content.String(), "\n")
	modalH := len(modalLines)
	padTop := (height - modalH) / 2
	if padTop < 0 {
		padTop = 0
	}
	padLeft := (width - modalW) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	var resultStr strings.Builder
	for i := 0; i < padTop; i++ {
		resultStr.WriteString(strings.Repeat(" ", width) + "\n")
	}
	for _, line := range modalLines {
		resultStr.WriteString(strings.Repeat(" ", padLeft) + line + "\n")
	}

	return resultStr.String()
}

func renderFrequencyContent(results []signal.FrequencyResult, maxW int) []string {
	var lines []string

	if len(results) == 0 {
		lines = append(lines, StyleEmpty.Render("No ERROR entries found"))
		return lines
	}

	lines = append(lines, StyleDetailLabel.Render("TOP ERROR SIGNALS"))
	lines = append(lines, "")

	for i, r := range results {
		if i >= 10 {
			break
		}
		msg := r.Message
		countStr := " x" + Itoa(r.Count)
		maxMsgW := maxW - len(countStr) - 2
		if len(msg) > maxMsgW {
			msg = msg[:maxMsgW-1] + "…"
		}
		line := StyleMessage.Render(msg) + StyleBarAccent.Render(countStr)
		lines = append(lines, line)
	}

	return lines
}

func renderLifetimeContent(r *signal.LifetimeResult, maxW int) []string {
	var lines []string

	if r == nil || r.Message == "" {
		lines = append(lines, StyleEmpty.Render("Select a line first"))
		return lines
	}

	lines = append(lines, StyleDetailLabel.Render("SIGNAL LIFETIME"))
	lines = append(lines, "")

	msg := r.Message
	if len(msg) > maxW-2 {
		msg = msg[:maxW-3] + "…"
	}
	lines = append(lines, StyleMessage.Render(msg))
	lines = append(lines, "")

	if r.IsSingle {
		lines = append(lines, StyleDetailDim.Render("Single occurrence"))
	} else {
		lines = append(lines, StyleDetailLabel.Render("First seen: ")+StyleDetailValue.Render(r.FirstSeen))
		lines = append(lines, StyleDetailLabel.Render("Last seen:  ")+StyleDetailValue.Render(r.LastSeen))
	}

	lines = append(lines, "")
	lines = append(lines, StyleDetailLabel.Render("Occurrences: ")+StyleBarAccent.Render(Itoa(r.Occurrences)))

	return lines
}

func renderBurstContent(r *signal.BurstResult, maxW int) []string {
	var lines []string

	if r == nil || r.Message == "" {
		lines = append(lines, StyleEmpty.Render("Select a line first"))
		return lines
	}

	lines = append(lines, StyleDetailLabel.Render("BURST ANALYSIS"))
	lines = append(lines, "")

	msg := r.Message
	if len(msg) > maxW-2 {
		msg = msg[:maxW-3] + "…"
	}
	lines = append(lines, StyleMessage.Render(msg))
	lines = append(lines, "")

	if r.Detected {
		lines = append(lines, StyleLevelError.Render(" BURST DETECTED "))
		lines = append(lines, "")
		lines = append(lines, StyleDetailValue.Render(r.Description))
	} else {
		lines = append(lines, StyleStatus.Render("✓ No abnormal burst detected"))
		if r.Count > 0 {
			lines = append(lines, "")
			lines = append(lines, StyleDetailDim.Render("Analyzed "+Itoa(r.Count)+" timestamps"))
		}
	}

	return lines
}

func renderDiversityContent(r *signal.DiversityResult, maxW int) []string {
	var lines []string

	if r == nil {
		lines = append(lines, StyleEmpty.Render("No data"))
		return lines
	}

	lines = append(lines, StyleDetailLabel.Render("ERROR DIVERSITY"))
	lines = append(lines, "")

	lines = append(lines, StyleDetailLabel.Render("Total ERROR lines:     ")+StyleDetailValue.Render(Itoa(r.TotalErrors)))
	lines = append(lines, StyleDetailLabel.Render("Unique ERROR messages: ")+StyleDetailValue.Render(Itoa(r.UniqueErrors)))
	lines = append(lines, "")

	qualityLabel := "Signal quality: "
	var qualityValue string
	switch r.Quality {
	case "HIGH":
		qualityValue = StyleStatus.Render("● HIGH")
	case "MEDIUM":
		qualityValue = StyleLevelWarn.Render(" MEDIUM ")
	case "LOW":
		qualityValue = StyleLevelError.Render(" LOW ")
	default:
		qualityValue = StyleDetailValue.Render(r.Quality)
	}
	lines = append(lines, StyleDetailLabel.Render(qualityLabel)+qualityValue)

	if r.QualityReason != "" {
		lines = append(lines, StyleDetailDim.Render(r.QualityReason))
	}

	return lines
}
