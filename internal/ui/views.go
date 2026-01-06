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

func RenderTitleBar(s *app.State, workspaceIdx, workspaceCount, width int) string {
	var parts []string

	parts = append(parts, StyleBarHighlight.Render("lx"))

	sep := StyleBarDim.Render(" │ ")

	if workspaceCount > 1 {
		var wsIndicator string
		for i := 0; i < workspaceCount; i++ {
			if i == workspaceIdx {
				wsIndicator += StyleBarAccent.Render("[" + Itoa(i+1) + "]")
			} else {
				wsIndicator += StyleBarDim.Render(" " + Itoa(i+1) + " ")
			}
		}
		parts = append(parts, wsIndicator)
	}

	if s.FileName != "" {
		parts = append(parts, StyleBarHighlight.Render(Truncate(s.FileName, 25)))
	}

	if s.IsLive {
		liveText := "LIVE " + Itoa(len(s.Entries)) + " lines"
		parts = append(parts, StyleBarAccent.Render("● ")+StyleBarHighlight.Render(liveText))
	} else if s.IsLoading {
		loadingText := "Loading... " + Itoa(s.LoadingProgress) + " lines"
		parts = append(parts, StyleBarAccent.Render("⟳ ")+StyleBarHighlight.Render(loadingText))
	} else {
		counts := StyleBarAccent.Render(Itoa(len(s.Filtered))) +
			StyleBarDim.Render("/"+Itoa(len(s.Entries)))
		parts = append(parts, counts)
	}

	if s.TotalNotes() > 0 {
		parts = append(parts, StyleBarAccent.Render("≡")+StyleBarText.Render(" "+Itoa(s.TotalNotes())))
	}

	if s.SelectionCount() > 0 {
		parts = append(parts, StyleBarAccent.Render("●")+StyleBarText.Render(" "+Itoa(s.SelectionCount())+" sel"))
	}

	left := strings.Join(parts, sep)

	var right string
	if s.LevelFilter != app.LevelFilterAll {
		right = StyleBarAccent.Render("◉ ") + StyleBarHighlight.Render(s.LevelFilter.String())
	}
	if s.FilterQuery != "" {
		right += StyleBarAccent.Render("⚡") + StyleBarText.Render(s.FilterQuery)
	}
	if s.StatusMsg != "" && !s.IsLoading {
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

	maxNum := len(s.Entries)
	lineNumW := len(Itoa(maxNum)) + 1
	if lineNumW < 4 {
		lineNumW = 4
	}

	start := 0
	
	getItemHeight := func(idx int) int {
		h := 1
		if s.IsNoteShowing(idx) && s.HasNote(idx) {
			if note, ok := s.GetNoteObj(idx); ok {
				noteBox := RenderInlineNoteBox(note, width)
				h += strings.Count(noteBox, "\n") + 1
			}
		}
		return h
	}

	if s.Cursor >= len(s.Filtered) {
		s.Cursor = len(s.Filtered) - 1
	}
	
	matchedStart := 0
	
	if s.Cursor < height {
		testLines := 0
		fits := true
		for i := 0; i <= s.Cursor; i++ {
			testLines += getItemHeight(s.Filtered[i])
			if testLines > height {
				fits = false
				break
			}
		}
		if fits {
			matchedStart = 0
		} else {
			needed := 0
			for i := s.Cursor; i >= 0; i-- {
				h := getItemHeight(s.Filtered[i])
				if needed + h > height {
					matchedStart = i + 1
					break
				}
				needed += h
				matchedStart = i
			}
		}
	} else {
		needed := 0
		for i := s.Cursor; i >= 0; i-- {
			h := getItemHeight(s.Filtered[i])
			if needed + h > height {
				matchedStart = i + 1
				break
			}
			needed += h
			matchedStart = i
		}
	}
	
	start = matchedStart
	
	var lines []string
	end := len(s.Filtered)
	
	for i := start; i < end && len(lines) < height; i++ {
		entryIdx := s.Filtered[i]
		entry := s.Entries[entryIdx]
		isSelected := i == s.Cursor
		hasNote := s.HasNote(entryIdx)
		isChecked := s.IsSelected(entryIdx)

		itemLines := []string{}

		if s.IsNoteShowing(entryIdx) && hasNote {
			if note, ok := s.GetNoteObj(entryIdx); ok {
				noteBox := RenderInlineNoteBox(note, width)
				noteParts := strings.Split(noteBox, "\n")
				itemLines = append(itemLines, noteParts...)
			}
		}

		itemLines = append(itemLines, RenderListLine(&entry, entryIdx+1, lineNumW, width, isSelected, hasNote, isChecked))

		for _, l := range itemLines {
			if len(lines) < height {
				lines = append(lines, l)
			}
		}
	}

	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func RenderListLine(entry *logx.Entry, lineNum, lineNumW, width int, selected, hasNote, isChecked bool) string {
	var parts []string

	bg := lipgloss.NewStyle()
	if selected {
		bg = bg.Background(ColorBgSelect)
	}

	var checkbox string
	if isChecked {
		if selected {
			checkbox = StyleBarAccent.Copy().Background(ColorBgSelect).Render("●")
		} else {
			checkbox = StyleBarAccent.Render("●")
		}
	} else {
		checkbox = bg.Render(" ")
	}

	var noteIndicator string
	if hasNote {
		if selected {
			noteIndicator = StyleNoteIndicator.Copy().Background(ColorBgSelect).Render("≡")
		} else {
			noteIndicator = StyleNoteIndicator.Render("≡")
		}
	} else {
		noteIndicator = bg.Render(" ")
	}

	numStr := PadLeft(Itoa(lineNum), lineNumW-1)
	prefix := checkbox + noteIndicator + bg.Render(" ")
	if selected {
		parts = append(parts, prefix+StyleLineNumSelected.Copy().Background(ColorBgSelect).Render(numStr))
	} else {
		parts = append(parts, prefix+StyleLineNum.Render(numStr))
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
		endBracket := strings.Index(msg, "] ")
		if endBracket == -1 {
			return StyleNoteIndicator.Render(msg)
		}
		
		prefix := msg[:endBracket+2]
		rest := msg[endBracket+2:]

		if strings.HasPrefix(rest, "[") {
			endBlock := strings.Index(rest, "] ")
			if endBlock != -1 {
				blockContent := rest[1:endBlock]
				parts := strings.Fields(blockContent)
				
				var coloredBlock string
				if len(parts) > 1 {
					levelText := parts[0]
					timeText := parts[1]
					
					var levelStyle lipgloss.Style
					if levelText == "CRIT" {
						levelStyle = StyleLevelError
					} else if levelText == "UNSURE" {
						levelStyle = StyleLevelWarn
					} else {
						levelStyle = StyleLevelInfo
					}
					
					coloredBlock = "[" + levelStyle.Render(levelText) + " " + StyleTimestamp.Render(timeText) + "]"
				} else if len(parts) == 1 {
					timeText := parts[0]
					coloredBlock = "[" + StyleTimestamp.Render(timeText) + "]"
				} else {
					coloredBlock = "[" + blockContent + "]"
				}
				
				return StyleNoteIndicator.Render(prefix) + coloredBlock + StyleMessage.Render(rest[endBlock+1:])
			}
		}

		return StyleNoteIndicator.Render(prefix) + StyleMessage.Render(rest)
	}

	if isStack {
		return StyleStack.Render(msg)
	}
	return StyleMessage.Render(msg)
}

func RenderInlineNoteBox(note app.Note, width int) string {
	var headerText string
	var headerStyle lipgloss.Style
	switch note.Level {
	case app.NoteLevelCritical:
		headerText = "! CRIT"
		headerStyle = StyleLevelError
	case app.NoteLevelUnsure:
		headerText = "? UNSURE"
		headerStyle = StyleLevelWarn
	default:
		headerText = "NOT"
		headerStyle = StyleNotesHeader
	}

	boxW := 40
	maxW := width - 8
	if maxW < boxW {
		boxW = maxW
	}
	if boxW < 20 {
		boxW = 20
	}

	innerW := boxW - 4


	safeText := strings.ReplaceAll(note.Text, "\t", " ")
	safeText = strings.ReplaceAll(safeText, "\n", " ")
	safeText = strings.ReplaceAll(safeText, "\r", " ")
	
	noteRunes := []rune(safeText)
	var noteLines []string

	for len(noteRunes) > 0 {
		str := string(noteRunes)
		width := lipgloss.Width(str)
		if width <= innerW {
			noteLines = append(noteLines, str)
			break
		}
		
		breakAt := 0
		currentW := 0
		lastSpace := -1
		
		for i, r := range noteRunes {
			w := lipgloss.Width(string(r))
			if currentW+w > innerW {
				breakAt = i
				break
			}
			if r == ' ' {
				lastSpace = i
			}
			currentW += w
		}
		
		if breakAt == 0 {
			breakAt = 1
		}

		if lastSpace != -1 && lastSpace > innerW/2 {
			breakAt = lastSpace
		}
		
		noteLines = append(noteLines, string(noteRunes[:breakAt]))
		
		if breakAt < len(noteRunes) && noteRunes[breakAt] == ' ' {
			breakAt++
		}
		
		if breakAt >= len(noteRunes) {
			break
		}
		noteRunes = noteRunes[breakAt:]
	}

	if len(noteLines) > 3 {
		noteLines = noteLines[:3]
		runes := []rune(noteLines[2])
		if len(runes) > 3 {
			noteLines[2] = string(runes[:len(runes)-3]) + "..."
		}
	}

	var result []string
	
	headerRendered := headerStyle.Render(" "+headerText+" ")
	headerW := lipgloss.Width(headerRendered)
	
	rightDash := boxW - 3 - headerW
	if rightDash < 0 {
		rightDash = 0
	}
	topLine := "   " + StyleNoteBoxBorder.Render("╭─") +
		headerRendered +
		StyleNoteBoxBorder.Render(strings.Repeat("─", rightDash)+"╮")
	result = append(result, topLine)

	for _, line := range noteLines {
		lineWidth := lipgloss.Width(line)
		pad := innerW - lineWidth
		if pad < 0 {
			pad = 0
			line = Truncate(line, innerW) 
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
	modalW := 56
	innerW := modalW - 4

	var content strings.Builder

	headerText := " LX - LOG X-RAY "
	headerPadTotal := modalW - 2 - len(headerText)
	leftPad := headerPadTotal / 2
	rightPad := headerPadTotal - leftPad
	content.WriteString(StyleFrameBorder.Render("╭"+strings.Repeat("─", leftPad)) + StyleDetailHeader.Render(headerText) + StyleFrameBorder.Render(strings.Repeat("─", rightPad)+"╮") + "\n")

	emptyLine := func() {
		content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")
	}
	sectionHeader := func(title string) {
		styled := StyleBarAccent.Render("── " + title + " ")
		styledW := lipgloss.Width(styled)
		pad := innerW - styledW
		if pad < 0 {
			pad = 0
		}
		content.WriteString(StyleFrameBorder.Render("│") + " " + styled + StyleBarDim.Render(strings.Repeat("─", pad)) + " " + StyleFrameBorder.Render("│") + "\n")
	}
	row := func(key, desc string) {
		keyPart := StyleHelpKey.Render(PadRight(key, 20))
		descPart := StyleHelpDesc.Render(desc)
		line := " " + keyPart + descPart
		lineW := lipgloss.Width(line)
		pad := innerW - lineW
		if pad < 0 {
			pad = 0
		}
		content.WriteString(StyleFrameBorder.Render("│") + " " + line + strings.Repeat(" ", pad) + " " + StyleFrameBorder.Render("│") + "\n")
	}
	centerText := func(text string, style lipgloss.Style) {
		styled := style.Render(text)
		styledW := lipgloss.Width(styled)
		padL := (innerW - styledW) / 2
		padR := innerW - styledW - padL
		if padL < 0 {
			padL = 0
		}
		if padR < 0 {
			padR = 0
		}
		content.WriteString(StyleFrameBorder.Render("│") + " " + strings.Repeat(" ", padL) + styled + strings.Repeat(" ", padR) + " " + StyleFrameBorder.Render("│") + "\n")
	}

	emptyLine()
	centerText("No logs loaded", StyleHelpDesc)
	centerText("Start by loading some data", StyleDetailDim)
	emptyLine()

	sectionHeader("QUICK START")
	row("o", "open file")
	row("p", "paste from clipboard")
	row("cmd | lx", "pipe from command")

	emptyLine()

	sectionHeader("EXAMPLES")
	row("lx app.log", "view log file")
	row("tail -f x | lx", "stream logs")
	row("kubectl logs | lx", "kubernetes logs")

	emptyLine()

	content.WriteString(StyleFrameBorder.Render("├" + strings.Repeat("─", modalW-2) + "┤") + "\n")
	hints := StyleHelpKey.Render("o") + StyleFooter.Render(" open  ") +
		StyleHelpKey.Render("p") + StyleFooter.Render(" paste  ") +
		StyleHelpKey.Render("?") + StyleFooter.Render(" help  ") +
		StyleHelpKey.Render("q") + StyleFooter.Render(" quit")
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

func RenderDetail(entry *logx.Entry, height, width, offset int) string {
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
		contentLines = renderJSONDetailLines(entry, width-2)
	} else {
		contentLines = renderTextDetailLines(entry, width-2)
	}

	totalLines := len(contentLines)
	if offset > totalLines-contentH {
		offset = totalLines - contentH
	}
	if offset < 0 {
		offset = 0
	}

	visibleLines := contentLines[offset:]
	if len(visibleLines) > contentH {
		visibleLines = visibleLines[:contentH]
	}

	lines = append(lines, visibleLines...)

	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func renderJSONDetailLines(entry *logx.Entry, width int) []string {
	var lines []string

	keyFields := []string{"msg", "message", "level", "severity", "error", "timestamp", "time"}
	shown := make(map[string]bool)

	for _, key := range keyFields {
		if val, ok := entry.Fields[key]; ok {
			line := " " + StyleDetailLabel.Render(key+":") + " " + StyleDetailValue.Render(formatValue(val))
			if lipgloss.Width(line) > width {
				line = Truncate(line, width)
			}
			lines = append(lines, line)
			shown[key] = true
		}
	}

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
			wrapped := performWrapping(line, width, 2)
			for _, wLine := range strings.Split(wrapped, "\n") {
				lines = append(lines, wLine)
			}
		}
	}

	return lines
}

func renderTextDetailLines(entry *logx.Entry, width int) []string {
	var lines []string

	if entry.Timestamp != "" {
		lines = append(lines, " "+StyleDetailLabel.Render("Time:")+" "+StyleDetailValue.Render(entry.Timestamp))
	}
	if entry.Level != logx.LevelUnknown {
		lines = append(lines, " "+StyleDetailLabel.Render("Level:")+" "+LevelStyle(entry.Level).Render(" "+entry.Level.String()+" "))
	}

	lines = append(lines, " "+StyleDetailDim.Render("───"))

	style := lipgloss.NewStyle().Width(width - 2).Foreground(ColorTextPrimary)
	rendered := style.Render(entry.Raw)
	rawLines := strings.Split(rendered, "\n")

	for _, line := range rawLines {
		lines = append(lines, " "+line)
	}

	return lines
}

func performWrapping(text string, width int, indent int) string {
	if width <= 0 {
		return text
	}
	style := lipgloss.NewStyle().Width(width).PaddingLeft(indent)
	return style.Render(text)
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

func RenderNotesModal(note string, cursorPos, lineNum, height, width int) string {
	modalW := 50
	if modalW > width-8 {
		modalW = width - 8
	}
	modalH := 7

	var content strings.Builder

	level, _ := app.ParseNoteLevel(note)
	var headerText string
	var headerStyle lipgloss.Style
	switch level {
	case app.NoteLevelCritical:
		headerText = " ! CRIT - LINE " + Itoa(lineNum) + " "
		headerStyle = StyleLevelError
	case app.NoteLevelUnsure:
		headerText = " ? UNSURE - LINE " + Itoa(lineNum) + " "
		headerStyle = StyleLevelWarn
	default:
		headerText = " NOTE FOR LINE " + Itoa(lineNum) + " "
		headerStyle = StyleNotesHeader
	}

	headerRendered := headerStyle.Render(headerText)
	headerW := lipgloss.Width(headerRendered)
	
	leftPad := (modalW - 2 - headerW) / 2
	rightPad := modalW - 2 - headerW - leftPad
	if leftPad < 0 {
		leftPad = 0
	}
	if rightPad < 0 {
		rightPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("╭" + strings.Repeat("─", leftPad)) + headerRendered + StyleFrameBorder.Render(strings.Repeat("─", rightPad) + "╮") + "\n")

	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")

	maxW := modalW - 7
	if maxW < 10 {
		maxW = 10
	}

	noteRunes := []rune(note)
	if cursorPos > len(noteRunes) {
		cursorPos = len(noteRunes)
	}
	if cursorPos < 0 {
		cursorPos = 0
	}

	var displayStart, displayEnd int
	if len(noteRunes) <= maxW {
		displayStart = 0
		displayEnd = len(noteRunes)
	} else {
		if cursorPos <= maxW/2 {
			displayStart = 0
			displayEnd = maxW
		} else if cursorPos >= len(noteRunes)-maxW/2 {
			displayStart = len(noteRunes) - maxW
			displayEnd = len(noteRunes)
		} else {
			displayStart = cursorPos - maxW/2
			displayEnd = displayStart + maxW
		}
	}

	if displayStart < 0 { displayStart = 0 }
	if displayEnd > len(noteRunes) { displayEnd = len(noteRunes) }
	if displayEnd < displayStart { displayEnd = displayStart }

	displayRunes := noteRunes[displayStart:displayEnd]
	displayCursorPos := cursorPos - displayStart

	prefix := ""
	if displayStart > 0 {
		prefix = "…"
		if len(displayRunes) > 0 {
			displayRunes = displayRunes[1:]
		}
		displayCursorPos--
		if displayCursorPos < 0 {
			displayCursorPos = 0
		}
	}

	var inputLine string
	if displayCursorPos >= len(displayRunes) {
		inputLine = prefix + string(displayRunes) + StyleCursorIndicator.Render("█")
	} else {
		beforeCursor := string(displayRunes[:displayCursorPos])
		afterCursor := string(displayRunes[displayCursorPos:])
		inputLine = prefix + beforeCursor + StyleCursorIndicator.Render("█") + afterCursor
	}

	inputPad := modalW - 4 - lipgloss.Width(inputLine)
	if inputPad < 0 {
		inputPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + " " + StyleNotesInput.Render(inputLine) + strings.Repeat(" ", inputPad) + " " + StyleFrameBorder.Render("│") + "\n")

	charCount := Itoa(len(noteRunes)) + "/100"
	var counterStyle lipgloss.Style
	if len(noteRunes) >= 90 {
		counterStyle = StyleLevelError
	} else if len(noteRunes) >= 75 {
		counterStyle = StyleLevelWarn
	} else {
		counterStyle = StyleHelpDesc
	}
	counterText := counterStyle.Render(charCount)
	counterW := lipgloss.Width(counterText)
	counterPad := modalW - 3 - counterW
	if counterPad < 0 {
		counterPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", counterPad) + counterText + " " + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("├" + strings.Repeat("─", modalW-2) + "┤") + "\n")

	hints := StyleHelpKey.Render("Enter") + StyleFooter.Render(" save  ") +
		StyleHelpKey.Render("ESC") + StyleFooter.Render(" cancel  ") +
		StyleHelpKey.Render("←→") + StyleFooter.Render(" move")
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

func RenderFilterModal(query string, levelFilter int, height, width int, textFilterDisabled bool) string {
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
	if textFilterDisabled {
		headerText = " LEVEL FILTER "
	}
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

	levelFilters := []string{"ALL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
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

	if !textFilterDisabled {
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

func RenderOpenFileModal(path string, cursorPos int, suggestions []string, suggIdx, height, width int) string {
	modalW := 60
	if modalW > width-4 {
		modalW = width - 4
	}
	if modalW < 40 {
		modalW = 40
	}

	var content strings.Builder

	headerText := " OPEN FILE "
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

	maxW := modalW - 7
	pathRunes := []rune(path)
	if cursorPos > len(pathRunes) {
		cursorPos = len(pathRunes)
	}
	if cursorPos < 0 {
		cursorPos = 0
	}

	var displayStart, displayEnd int
	if len(pathRunes) <= maxW {
		displayStart = 0
		displayEnd = len(pathRunes)
	} else {
		if cursorPos <= maxW/2 {
			displayStart = 0
			displayEnd = maxW
		} else if cursorPos >= len(pathRunes)-maxW/2 {
			displayStart = len(pathRunes) - maxW
			displayEnd = len(pathRunes)
		} else {
			displayStart = cursorPos - maxW/2
			displayEnd = displayStart + maxW
		}
	}

	displayRunes := pathRunes[displayStart:displayEnd]
	displayCursorPos := cursorPos - displayStart

	prefix := ""
	if displayStart > 0 {
		prefix = "…"
		if len(displayRunes) > 0 {
			displayRunes = displayRunes[1:]
		}
		displayCursorPos--
		if displayCursorPos < 0 {
			displayCursorPos = 0
		}
	}

	var inputLine string
	if displayCursorPos >= len(displayRunes) {
		inputLine = prefix + string(displayRunes) + StyleCursorIndicator.Render("█")
	} else {
		beforeCursor := string(displayRunes[:displayCursorPos])
		afterCursor := string(displayRunes[displayCursorPos:])
		inputLine = prefix + beforeCursor + StyleCursorIndicator.Render("█") + afterCursor
	}

	inputPad := modalW - 4 - lipgloss.Width(inputLine)
	if inputPad < 0 {
		inputPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + " " + StyleNotesInput.Render(inputLine) + strings.Repeat(" ", inputPad) + " " + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")

	if len(suggestions) > 0 {
		maxSuggestions := 5
		if len(suggestions) < maxSuggestions {
			maxSuggestions = len(suggestions)
		}

		startIdx := 0
		if suggIdx >= maxSuggestions {
			startIdx = suggIdx - maxSuggestions + 1
		}
		endIdx := startIdx + maxSuggestions
		if endIdx > len(suggestions) {
			endIdx = len(suggestions)
			startIdx = endIdx - maxSuggestions
			if startIdx < 0 {
				startIdx = 0
			}
		}

		for i := startIdx; i < endIdx; i++ {
			sugg := suggestions[i]
			maxSuggW := modalW - 6
			suggRunes := []rune(sugg)
			displaySugg := sugg
			if len(suggRunes) > maxSuggW {
				displaySugg = "…" + string(suggRunes[len(suggRunes)-maxSuggW+1:])
			}

			var line string
			if i == suggIdx {
				line = StyleLookupSelected.Render(" " + displaySugg + " ")
			} else {
				line = StyleHelpDesc.Render(" " + displaySugg + " ")
			}
			lineW := lipgloss.Width(line)
			linePad := modalW - 4 - lineW
			if linePad < 0 {
				linePad = 0
			}
			content.WriteString(StyleFrameBorder.Render("│") + " " + line + strings.Repeat(" ", linePad) + " " + StyleFrameBorder.Render("│") + "\n")
		}

		if len(suggestions) > maxSuggestions {
			moreText := StyleHelpDesc.Render("  ↑↓ " + Itoa(len(suggestions)) + " matches")
			moreW := lipgloss.Width(moreText)
			morePad := modalW - 4 - moreW
			if morePad < 0 {
				morePad = 0
			}
			content.WriteString(StyleFrameBorder.Render("│") + " " + moreText + strings.Repeat(" ", morePad) + " " + StyleFrameBorder.Render("│") + "\n")
		}

		content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")
	}

	hint := StyleHelpDesc.Render("Tab: autocomplete  ↑↓: select")
	hintW := lipgloss.Width(hint)
	hintPad := (modalW - 2 - hintW) / 2
	if hintPad < 0 {
		hintPad = 0
	}
	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", hintPad) + hint + strings.Repeat(" ", modalW-2-hintPad-hintW) + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")

	content.WriteString(StyleFrameBorder.Render("├" + strings.Repeat("─", modalW-2) + "┤") + "\n")

	hints := StyleHelpKey.Render("Enter") + StyleFooter.Render(" open  ") +
		StyleHelpKey.Render("ESC") + StyleFooter.Render(" cancel")
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
	modalW := 60
	innerW := modalW - 4

	var content strings.Builder

	headerText := " KEYBOARD SHORTCUTS "
	headerPadTotal := modalW - 2 - len(headerText)
	leftPad := headerPadTotal / 2
	rightPad := headerPadTotal - leftPad
	content.WriteString(StyleFrameBorder.Render("╭"+strings.Repeat("─", leftPad)) + StyleDetailHeader.Render(headerText) + StyleFrameBorder.Render(strings.Repeat("─", rightPad)+"╮") + "\n")

	emptyLine := func() {
		content.WriteString(StyleFrameBorder.Render("│") + strings.Repeat(" ", modalW-2) + StyleFrameBorder.Render("│") + "\n")
	}
	sectionHeader := func(title string) {
		styled := StyleBarAccent.Render("── " + title + " ")
		styledW := lipgloss.Width(styled)
		pad := innerW - styledW
		if pad < 0 {
			pad = 0
		}
		content.WriteString(StyleFrameBorder.Render("│") + " " + styled + StyleBarDim.Render(strings.Repeat("─", pad)) + " " + StyleFrameBorder.Render("│") + "\n")
	}
	row := func(key1, desc1, key2, desc2 string) {
		padKeyRight := func(s string, w int) string {
			visW := lipgloss.Width(s)
			if visW >= w {
				return s
			}
			return s + strings.Repeat(" ", w-visW)
		}
		col1 := StyleHelpKey.Render(padKeyRight(key1, 8)) + StyleHelpDesc.Render(PadRight(desc1, 18))
		col2 := ""
		if key2 != "" {
			col2 = StyleHelpKey.Render(padKeyRight(key2, 8)) + StyleHelpDesc.Render(desc2)
		}
		line := " " + col1 + col2
		lineW := lipgloss.Width(line)
		pad := innerW - lineW
		if pad < 0 {
			pad = 0
		}
		content.WriteString(StyleFrameBorder.Render("│") + " " + line + strings.Repeat(" ", pad) + " " + StyleFrameBorder.Render("│") + "\n")
	}

	emptyLine()

	sectionHeader("NAVIGATION")
	row("j / ↓", "move down", "k / ↑", "move up")
	row("g", "go to top", "G", "go to bottom")
	row("Enter", "toggle detail", "z", "maximize detail")

	emptyLine()

	sectionHeader("FILTER")
	row("/", "open filter", "Tab", "cycle level")
	row("^R", "clear filter", "", "")

	emptyLine()

	sectionHeader("NOTES")
	row("N", "write/edit note", "n", "toggle note")
	row("m", "show/hide all", "] / [", "next/prev note")
	row("D", "delete note", "", "")

	emptyLine()

	sectionHeader("SELECTION")
	row("s", "toggle select", "S", "select all")
	row("c", "copy selected", "d", "delete selected")

	emptyLine()

	sectionHeader("SIGNAL ANALYSIS")
	row("1", "error frequency", "2", "first/last seen")
	row("3", "burst detector", "4", "error diversity")

	emptyLine()

	sectionHeader("ACTIONS")
	row("y", "copy all", "x", "clear all")
	row("u", "undo delete", "U", "redo delete")
	row("p / ^V", "paste logs", "o", "open file")
	row("^L", "HTTP lookup", "", "")

	emptyLine()

	sectionHeader("WORKSPACE")
	row("T", "new workspace", "W", "close workspace")
	row("Tab", "next workspace", "S-Tab", "prev workspace")

	emptyLine()

	sectionHeader("OTHER")
	row("?", "this help", "q", "quit")

	emptyLine()

	footer := StyleHelpDesc.Render("lx - Log X-Ray by kalayciburak")
	footerW := lipgloss.Width(footer)
	footerPad := (innerW - footerW) / 2
	if footerPad < 0 {
		footerPad = 0
	}
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

func RenderFooter(mode int, statusMsg string, selectionCount, width int) string {
	var hintParts []string

	if selectionCount > 0 && app.Mode(mode) == app.ModeList {
		hintParts = append(hintParts,
			StyleBarAccent.Render("●")+StyleBarText.Render(" "+Itoa(selectionCount)),
			StyleBarAccent.Render("c")+StyleBarText.Render(" copy"),
			StyleBarAccent.Render("d")+StyleBarText.Render(" delete"),
			StyleBarAccent.Render("S")+StyleBarText.Render(" clear"),
			StyleBarAccent.Render("s")+StyleBarText.Render(" +/-"))
	} else {
		switch app.Mode(mode) {
		case app.ModeFilter:
			hintParts = append(hintParts,
				StyleBarAccent.Render("Tab")+StyleBarText.Render(" level"),
				StyleBarAccent.Render("Enter")+StyleBarText.Render(" apply"),
				StyleBarAccent.Render("ESC")+StyleBarText.Render(" cancel"))
		case app.ModeDetail:
			hintParts = append(hintParts,
				StyleBarAccent.Render("z")+StyleBarText.Render(" maximize"),
				StyleBarAccent.Render("^L")+StyleBarText.Render(" lookup"),
				StyleBarAccent.Render("1-4")+StyleBarText.Render(" signal"),
				StyleBarAccent.Render("c")+StyleBarText.Render(" copy"),
				StyleBarAccent.Render("ESC")+StyleBarText.Render(" back"))
		case app.ModeNotes:
			hintParts = append(hintParts,
				StyleBarAccent.Render("!")+StyleBarText.Render(" crit"),
				StyleBarAccent.Render("?")+StyleBarText.Render(" unsure"),
				StyleBarAccent.Render("Enter")+StyleBarText.Render(" save"),
				StyleBarAccent.Render("ESC")+StyleBarText.Render(" cancel"))
		case app.ModeLookup:
			hintParts = append(hintParts,
				StyleBarAccent.Render("↑↓")+StyleBarText.Render(" select"),
				StyleBarAccent.Render("Enter")+StyleBarText.Render(" copy"),
				StyleBarAccent.Render("ESC")+StyleBarText.Render(" close"))
		case app.ModeSignal:
			hintParts = append(hintParts,
				StyleBarAccent.Render("c")+StyleBarText.Render(" copy"),
				StyleBarAccent.Render("ESC")+StyleBarText.Render(" close"))
		default:
			hintParts = append(hintParts,
				StyleBarAccent.Render("s")+StyleBarText.Render(" select"),
				StyleBarAccent.Render("/")+StyleBarText.Render(" filter"),
				StyleBarAccent.Render("N")+StyleBarText.Render(" note"),
				StyleBarAccent.Render("?")+StyleBarText.Render(" help"))
		}
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

	var hints string
	if result.Type == signal.SignalLifetime || result.Type == signal.SignalBurst {
		hints = StyleHelpKey.Render("j/k") + StyleFooter.Render(" nav  ") +
			StyleHelpKey.Render("c") + StyleFooter.Render(" copy  ") +
			StyleHelpKey.Render("ESC") + StyleFooter.Render(" close")
	} else {
		hints = StyleHelpKey.Render("c") + StyleFooter.Render(" copy  ") +
			StyleHelpKey.Render("ESC") + StyleFooter.Render(" close")
	}
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
