package ui

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kalayciburak/lx/internal/app"
	"github.com/kalayciburak/lx/internal/logx"
	"github.com/kalayciburak/lx/internal/lookup"
	"github.com/kalayciburak/lx/internal/signal"
)

func sanitizeForClipboard(s string) string {
	return strings.ReplaceAll(s, "\x00", "")
}

type Model struct {
	State  *app.State
	Width  int
	Height int
}

func NewModel(state *app.State) Model {
	return Model{
		State:  state,
		Width:  80,
		Height: 24,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.State.Mode == app.ModeList || m.State.Mode == app.ModeDetail {
			m.State.MoveCursor(-3)
		}
	case tea.MouseButtonWheelDown:
		if m.State.Mode == app.ModeList || m.State.Mode == app.ModeDetail {
			m.State.MoveCursor(3)
		}
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.State.Mode {
	case app.ModeFilter:
		return m.handleFilterMode(msg)
	case app.ModeDetail:
		return m.handleDetailMode(msg)
	case app.ModeHelp:
		return m.handleHelpMode(msg)
	case app.ModeNotes:
		return m.handleNotesMode(msg)
	case app.ModeLookup:
		return m.handleLookupMode(msg)
	case app.ModeSignal:
		return m.handleSignalMode(msg)
	default:
		return m.handleListMode(msg)
	}
}

func (m Model) handleListMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case IsKey(msg, KeyQ):
		return m, tea.Quit
	case IsKey(msg, KeyCtrlC):
		return m, tea.Quit
	case IsKey(msg, KeyJ, KeyDown):
		m.State.MoveCursor(1)
	case IsKey(msg, KeyK, KeyUp):
		m.State.MoveCursor(-1)
	case IsKey(msg, KeyG):
		m.State.Cursor = 0
	case IsKey(msg, KeyShiftG):
		if len(m.State.Filtered) > 0 {
			m.State.Cursor = len(m.State.Filtered) - 1
		}
	case IsKey(msg, KeyEnter, KeySpace):
		if len(m.State.Filtered) > 0 {
			m.State.Mode = app.ModeDetail
		}
	case IsKey(msg, KeySlash):
		m.State.Mode = app.ModeFilter
	case IsKey(msg, KeyCtrlR):
		m.State.FilterQuery = ""
		m.State.LevelFilter = app.LevelFilterAll
		m.State.Refilter()
		m.State.StatusMsg = "Filter cleared"
	case IsKey(msg, KeyQuestion):
		m.State.Mode = app.ModeHelp
	case IsKey(msg, KeyShiftN):
		idx := m.State.SelectedIndex()
		if idx >= 0 {
			m.State.NoteLineIdx = idx
			if noteObj, ok := m.State.GetNoteObj(idx); ok {
				prefix := noteObj.Level.Symbol()
				m.State.CurrentNote = prefix + noteObj.Text
			} else {
				m.State.CurrentNote = ""
			}
			m.State.NoteCursorPos = len([]rune(m.State.CurrentNote))
			m.State.PrevMode = m.State.Mode
			m.State.Mode = app.ModeNotes
		}
	case IsKey(msg, KeyN):
		idx := m.State.SelectedIndex()
		if idx >= 0 {
			if m.State.HasNote(idx) {
				m.State.ToggleNoteDisplay(idx)
			} else {
				m.State.StatusMsg = "No note on this line"
			}
		}
	case IsKey(msg, KeyU_TR, KeyBracketRight):
		if nextIdx := m.State.NextNotedLine(); nextIdx >= 0 {
			m.State.JumpToEntry(nextIdx)
			m.State.ShowingNotes[nextIdx] = true
			m.State.StatusMsg = "Note: " + Truncate(m.State.GetNote(nextIdx), 30)
		}
	case IsKey(msg, KeyG_TR, KeyBracketLeft):
		if prevIdx := m.State.PrevNotedLine(); prevIdx >= 0 {
			m.State.JumpToEntry(prevIdx)
			m.State.ShowingNotes[prevIdx] = true
			m.State.StatusMsg = "Note: " + Truncate(m.State.GetNote(prevIdx), 30)
		}
	case IsKey(msg, KeyM):
		if m.State.TotalNotes() > 0 {
			m.State.ToggleAllNotesDisplay()
			if m.State.CountShowingNotes() > 0 {
				m.State.StatusMsg = "Showing all notes"
			} else {
				m.State.StatusMsg = "Hiding all notes"
			}
		} else {
			m.State.StatusMsg = "No notes to show"
		}
	case IsKey(msg, KeyCtrlL):
		m.State.Mode = app.ModeLookup
		if entry := m.State.SelectedEntry(); entry != nil {
			if code := lookup.ExtractHTTPCode(entry.Raw); code > 0 {
				m.State.LookupQuery = Itoa(code)
				m.State.UpdateLookup()
			}
		}
	case IsKey(msg, Key1):
		m.State.SignalResult = signal.ErrorFrequency(m.State.VisibleEntries(), 10)
		m.State.Mode = app.ModeSignal
	case IsKey(msg, Key2):
		if entry := m.State.SelectedEntry(); entry != nil {
			m.State.SignalResult = signal.Lifetime(m.State.VisibleEntries(), entry.Message)
			m.State.Mode = app.ModeSignal
		}
	case IsKey(msg, Key3):
		if entry := m.State.SelectedEntry(); entry != nil {
			m.State.SignalResult = signal.DetectBurst(m.State.VisibleEntries(), entry.Message)
			m.State.Mode = app.ModeSignal
		}
	case IsKey(msg, Key4):
		m.State.SignalResult = signal.Diversity(m.State.VisibleEntries())
		m.State.Mode = app.ModeSignal
	case IsKey(msg, KeyY):
		content := app.ExportLogsWithNotes(m.State.VisibleEntries(), m.State.Notes, m.State.Filtered)
		if err := clipboard.WriteAll(sanitizeForClipboard(content)); err != nil {
			m.State.StatusMsg = "Clipboard error"
		} else {
			notesCount := m.State.FilteredNotesCount()
			if notesCount > 0 {
				m.State.StatusMsg = "Copied " + Itoa(len(m.State.Filtered)) + " lines + " + Itoa(notesCount) + " notes"
			} else {
				m.State.StatusMsg = app.CountExport(len(m.State.Filtered), "line")
			}
		}
	case IsKey(msg, KeyC):
		idx := m.State.SelectedIndex()
		if entry := m.State.SelectedEntry(); entry != nil {
			lineNum := idx + 1
			var content string
			if noteObj, ok := m.State.GetNoteObj(idx); ok {
				levelStr := ""
				if noteObj.Level != app.NoteLevelNormal {
					levelStr = noteObj.Level.String() + " "
				}
				content = "=== NOTE (lx) ===\n• [line " + Itoa(lineNum) + "] [" + levelStr + noteObj.CreatedAt.Format("15:04:05") + "] " + noteObj.Text + "\n\n=== LOG ===\nline " + Itoa(lineNum) + ": " + app.ExportEntry(entry)
			} else {
				content = "line " + Itoa(lineNum) + ": " + app.ExportEntry(entry)
			}
			if err := clipboard.WriteAll(sanitizeForClipboard(content)); err != nil {
				m.State.StatusMsg = "Clipboard error"
			} else {
				if m.State.HasNote(idx) {
					m.State.StatusMsg = "Copied line " + Itoa(lineNum) + " + note"
				} else {
					m.State.StatusMsg = "Copied line " + Itoa(lineNum)
				}
			}
		}
	case IsKey(msg, KeyD):
		m.State.DeleteSelected()
		m.State.StatusMsg = "Deleted"
	case IsKey(msg, KeyX):
		m.State.ClearAll()
		m.State.StatusMsg = "Cleared all"
	case IsKey(msg, KeyP, KeyCtrlV):
		content, err := clipboard.ReadAll()
		if err != nil {
			m.State.StatusMsg = "Clipboard error"
		} else {
			lines := splitLines(content)
			if len(lines) > 0 {
				m.State.LoadFromClipboard(lines)
				m.State.StatusMsg = "Loaded " + Itoa(len(lines)) + " lines"
			}
		}
	}
	return m, nil
}

func (m Model) handleFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case IsKey(msg, KeyEsc):
		m.State.Mode = app.ModeList
	case IsKey(msg, KeyEnter):
		m.State.Mode = app.ModeList
		m.State.Refilter()
	case IsKey(msg, KeyTab):
		m.State.CycleLevelFilter()
	case IsKey(msg, KeyBackspace):
		if len(m.State.FilterQuery) > 0 {
			m.State.FilterQuery = m.State.FilterQuery[:len(m.State.FilterQuery)-1]
			m.State.Refilter()
		}
	case IsKey(msg, KeyCtrlC):
		return m, tea.Quit
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			m.State.FilterQuery += msg.String()
			m.State.Refilter()
		}
	}
	return m, nil
}

func (m Model) handleDetailMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case IsKey(msg, KeyEsc, KeyEnter, KeySpace):
		m.State.Mode = app.ModeList
	case IsKey(msg, KeyJ, KeyDown):
		m.State.MoveCursor(1)
		m.State.DetailScroll = 0
	case IsKey(msg, KeyK, KeyUp):
		m.State.MoveCursor(-1)
		m.State.DetailScroll = 0
	case IsKey(msg, KeyPgDn):
		m.State.DetailScroll += 5
	case IsKey(msg, KeyPgUp):
		m.State.DetailScroll -= 5
		if m.State.DetailScroll < 0 {
			m.State.DetailScroll = 0
		}
	case IsKey(msg, KeyShiftN):
		idx := m.State.SelectedIndex()
		if idx >= 0 {
			m.State.NoteLineIdx = idx
			if noteObj, ok := m.State.GetNoteObj(idx); ok {
				prefix := noteObj.Level.Symbol()
				m.State.CurrentNote = prefix + noteObj.Text
			} else {
				m.State.CurrentNote = ""
			}
			m.State.NoteCursorPos = len([]rune(m.State.CurrentNote))
			m.State.PrevMode = app.ModeDetail
			m.State.Mode = app.ModeNotes
		}
	case IsKey(msg, KeyN):
		idx := m.State.SelectedIndex()
		if idx >= 0 {
			if m.State.HasNote(idx) {
				m.State.ToggleNoteDisplay(idx)
			} else {
				m.State.StatusMsg = "No note"
			}
		}
	case IsKey(msg, KeyU_TR, KeyBracketRight):
		if nextIdx := m.State.NextNotedLine(); nextIdx >= 0 {
			m.State.JumpToEntry(nextIdx)
			m.State.ShowingNotes[nextIdx] = true
		}
	case IsKey(msg, KeyG_TR, KeyBracketLeft):
		if prevIdx := m.State.PrevNotedLine(); prevIdx >= 0 {
			m.State.JumpToEntry(prevIdx)
			m.State.ShowingNotes[prevIdx] = true
		}
	case IsKey(msg, KeyC):
		idx := m.State.SelectedIndex()
		if entry := m.State.SelectedEntry(); entry != nil {
			lineNum := idx + 1
			var content string
			if noteObj, ok := m.State.GetNoteObj(idx); ok {
				levelStr := ""
				if noteObj.Level != app.NoteLevelNormal {
					levelStr = noteObj.Level.String() + " "
				}
				content = "=== NOTE (lx) ===\n• [line " + Itoa(lineNum) + "] [" + levelStr + noteObj.CreatedAt.Format("15:04:05") + "] " + noteObj.Text + "\n\n=== LOG ===\nline " + Itoa(lineNum) + ": " + app.ExportEntry(entry)
			} else {
				content = "line " + Itoa(lineNum) + ": " + app.ExportEntry(entry)
			}
			if err := clipboard.WriteAll(sanitizeForClipboard(content)); err != nil {
				m.State.StatusMsg = "Clipboard error"
			} else {
				if m.State.HasNote(idx) {
					m.State.StatusMsg = "Copied line " + Itoa(lineNum) + " + note"
				} else {
					m.State.StatusMsg = "Copied line " + Itoa(lineNum)
				}
			}
		}
	case IsKey(msg, KeyM):
		if m.State.TotalNotes() > 0 {
			m.State.ToggleAllNotesDisplay()
			if m.State.CountShowingNotes() > 0 {
				m.State.StatusMsg = "Showing all notes"
			} else {
				m.State.StatusMsg = "Hiding all notes"
			}
		} else {
			m.State.StatusMsg = "No notes to show"
		}
	case IsKey(msg, KeyG):
		m.State.Cursor = 0
	case IsKey(msg, KeyShiftG):
		if len(m.State.Filtered) > 0 {
			m.State.Cursor = len(m.State.Filtered) - 1
		}
	case IsKey(msg, KeyY):
		content := app.ExportLogsWithNotes(m.State.VisibleEntries(), m.State.Notes, m.State.Filtered)
		if err := clipboard.WriteAll(sanitizeForClipboard(content)); err != nil {
			m.State.StatusMsg = "Clipboard error"
		} else {
			notesCount := m.State.FilteredNotesCount()
			if notesCount > 0 {
				m.State.StatusMsg = "Copied " + Itoa(len(m.State.Filtered)) + " lines + " + Itoa(notesCount) + " notes"
			} else {
				m.State.StatusMsg = app.CountExport(len(m.State.Filtered), "line")
			}
		}
	case IsKey(msg, KeyD):
		m.State.DeleteSelected()
		m.State.StatusMsg = "Deleted"
	case IsKey(msg, KeySlash):
		m.State.Mode = app.ModeFilter
	case IsKey(msg, KeyCtrlR):
		m.State.FilterQuery = ""
		m.State.LevelFilter = app.LevelFilterAll
		m.State.Refilter()
		m.State.StatusMsg = "Filter cleared"
	case IsKey(msg, KeyCtrlL):
		m.State.Mode = app.ModeLookup
		if entry := m.State.SelectedEntry(); entry != nil {
			if code := lookup.ExtractHTTPCode(entry.Raw); code > 0 {
				m.State.LookupQuery = Itoa(code)
				m.State.UpdateLookup()
			}
		}
	case IsKey(msg, Key1):
		m.State.SignalResult = signal.ErrorFrequency(m.State.VisibleEntries(), 10)
		m.State.Mode = app.ModeSignal
	case IsKey(msg, Key2):
		if entry := m.State.SelectedEntry(); entry != nil {
			m.State.SignalResult = signal.Lifetime(m.State.VisibleEntries(), entry.Message)
			m.State.Mode = app.ModeSignal
		}
	case IsKey(msg, Key3):
		if entry := m.State.SelectedEntry(); entry != nil {
			m.State.SignalResult = signal.DetectBurst(m.State.VisibleEntries(), entry.Message)
			m.State.Mode = app.ModeSignal
		}
	case IsKey(msg, Key4):
		m.State.SignalResult = signal.Diversity(m.State.VisibleEntries())
		m.State.Mode = app.ModeSignal
	case IsKey(msg, KeyQuestion):
		m.State.Mode = app.ModeHelp
	case IsKey(msg, KeyCtrlC):
		return m, tea.Quit
	case IsKey(msg, KeyQ):
		m.State.Mode = app.ModeList
	}
	return m, nil
}

func (m Model) handleHelpMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case IsKey(msg, KeyEsc, KeyQuestion, KeyQ):
		m.State.Mode = app.ModeList
	case IsKey(msg, KeyCtrlC):
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleNotesMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	runes := []rune(m.State.CurrentNote)
	cursorPos := m.State.NoteCursorPos

	if cursorPos > len(runes) {
		cursorPos = len(runes)
	}
	if cursorPos < 0 {
		cursorPos = 0
	}

	switch {
	case IsKey(msg, KeyEsc):
		m.State.SetNote(m.State.NoteLineIdx, m.State.CurrentNote)
		if m.State.CurrentNote != "" {
			m.State.StatusMsg = "Note saved"
			m.State.ShowingNotes[m.State.NoteLineIdx] = true
		}
		m.State.Mode = m.State.PrevMode
		if m.State.Mode == 0 {
			m.State.Mode = app.ModeList
		}
	case IsKey(msg, KeyEnter):
		m.State.SetNote(m.State.NoteLineIdx, m.State.CurrentNote)
		if m.State.CurrentNote != "" {
			m.State.StatusMsg = "Note saved"
			m.State.ShowingNotes[m.State.NoteLineIdx] = true
		}
		m.State.Mode = m.State.PrevMode
		if m.State.Mode == 0 {
			m.State.Mode = app.ModeList
		}
	case IsKey(msg, KeyCtrlC):
		if m.State.CurrentNote != "" {
			if err := clipboard.WriteAll(sanitizeForClipboard(m.State.CurrentNote)); err != nil {
				m.State.StatusMsg = "Clipboard error"
			} else {
				m.State.StatusMsg = "Copied note"
			}
		}
	case IsKey(msg, KeyLeft):
		if cursorPos > 0 {
			m.State.NoteCursorPos = cursorPos - 1
		}
	case IsKey(msg, KeyRight):
		if cursorPos < len(runes) {
			m.State.NoteCursorPos = cursorPos + 1
		}
	case IsKey(msg, KeyHome):
		m.State.NoteCursorPos = 0
	case IsKey(msg, KeyEnd):
		m.State.NoteCursorPos = len(runes)
	case IsKey(msg, KeyBackspace):
		if cursorPos > 0 {
			newRunes := append(runes[:cursorPos-1], runes[cursorPos:]...)
			m.State.CurrentNote = string(newRunes)
			m.State.NoteCursorPos = cursorPos - 1
		}
	case IsKey(msg, KeyDelete):
		if cursorPos < len(runes) {
			newRunes := append(runes[:cursorPos], runes[cursorPos+1:]...)
			m.State.CurrentNote = string(newRunes)
		}
	default:
		char := msg.String()
		charRunes := []rune(char)
		if len(charRunes) == 1 && len(runes) < 255 {
			newRunes := make([]rune, 0, len(runes)+1)
			newRunes = append(newRunes, runes[:cursorPos]...)
			newRunes = append(newRunes, charRunes[0])
			newRunes = append(newRunes, runes[cursorPos:]...)
			m.State.CurrentNote = string(newRunes)
			m.State.NoteCursorPos = cursorPos + 1
		}
	}
	return m, nil
}

func (m Model) handleLookupMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case IsKey(msg, KeyEsc):
		m.State.Mode = app.ModeList
		m.State.LookupQuery = ""
		m.State.LookupResults = nil
	case IsKey(msg, KeyJ, KeyDown):
		if m.State.LookupCursor < len(m.State.LookupResults)-1 {
			m.State.LookupCursor++
		}
	case IsKey(msg, KeyK, KeyUp):
		if m.State.LookupCursor > 0 {
			m.State.LookupCursor--
		}
	case IsKey(msg, KeyEnter):
		if result := m.State.SelectedLookupResult(); result != nil {
			content := Itoa(result.Code) + " " + result.Name + "\n" + result.Description + "\nExample: " + result.Example
			if err := clipboard.WriteAll(sanitizeForClipboard(content)); err != nil {
				m.State.StatusMsg = "Clipboard error"
			} else {
				m.State.StatusMsg = "Copied: " + result.Name
			}
		}
	case IsKey(msg, KeyBackspace):
		if len(m.State.LookupQuery) > 0 {
			m.State.LookupQuery = m.State.LookupQuery[:len(m.State.LookupQuery)-1]
			m.State.UpdateLookup()
		}
	case IsKey(msg, KeyCtrlC):
		return m, tea.Quit
	default:
		if len(msg.String()) == 1 {
			m.State.LookupQuery += msg.String()
			m.State.UpdateLookup()
		}
	}
	return m, nil
}

func (m Model) handleSignalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case IsKey(msg, KeyEsc, KeyQ):
		m.State.Mode = app.ModeList
		m.State.SignalResult = nil
	case IsKey(msg, KeyJ, KeyDown):
		if m.State.SignalResult != nil {
			switch m.State.SignalResult.Type {
			case signal.SignalLifetime, signal.SignalBurst:
				m.State.MoveCursor(1)
				m.updateSignalForCurrentEntry()
			}
		}
	case IsKey(msg, KeyK, KeyUp):
		if m.State.SignalResult != nil {
			switch m.State.SignalResult.Type {
			case signal.SignalLifetime, signal.SignalBurst:
				m.State.MoveCursor(-1)
				m.updateSignalForCurrentEntry()
			}
		}
	case IsKey(msg, KeyC):
		if m.State.SignalResult != nil {
			content := m.State.SignalResult.FormatForClipboard()
			if err := clipboard.WriteAll(sanitizeForClipboard(content)); err != nil {
				m.State.StatusMsg = "Clipboard error"
			} else {
				m.State.StatusMsg = "Copied signal data"
			}
			m.State.Mode = app.ModeList
			m.State.SignalResult = nil
		}
	case IsKey(msg, KeyCtrlC):
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) updateSignalForCurrentEntry() {
	if m.State.SignalResult == nil {
		return
	}
	entry := m.State.SelectedEntry()
	if entry == nil {
		return
	}
	switch m.State.SignalResult.Type {
	case signal.SignalLifetime:
		m.State.SignalResult = signal.Lifetime(m.State.VisibleEntries(), entry.Message)
	case signal.SignalBurst:
		m.State.SignalResult = signal.DetectBurst(m.State.VisibleEntries(), entry.Message)
	}
}

func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}
	w := m.Width - 2
	h := m.Height
	if w < 40 {
		w = 40
	}
	if h < 10 {
		h = 10
	}
	var content string
	switch m.State.Mode {
	case app.ModeHelp:
		content = m.renderWithHelp(w, h)
	case app.ModeNotes:
		content = m.renderWithNotes(w, h)
	case app.ModeLookup:
		content = m.renderWithLookup(w, h)
	case app.ModeSignal:
		content = m.renderWithSignal(w, h)
	case app.ModeFilter:
		content = m.renderWithFilter(w, h)
	case app.ModeDetail:
		content = m.renderWithDetail(w, h)
	default:
		content = m.renderNormal(w, h)
	}
	return Frame(content, m.Width, m.Height)
}

func (m Model) renderNormal(w, h int) string {
	titleBar := RenderTitleBar(m.State, w)
	footer := RenderFooter(int(m.State.Mode), m.State.StatusMsg, w)
	listH := h - 2
	if listH < 1 {
		listH = 1
	}
	var list string
	if len(m.State.Filtered) == 0 {
		list = RenderEmpty(listH, w)
	} else {
		list = RenderList(m.State, listH, w)
	}
	return titleBar + "\n" + list + "\n" + footer
}

func (m Model) renderWithDetail(w, h int) string {
	titleBar := RenderTitleBar(m.State, w)
	footer := RenderFooter(int(m.State.Mode), m.State.StatusMsg, w)
	contentH := h - 3
	if contentH < 6 {
		contentH = 6
	}
	entry := m.State.SelectedEntry()
	detailH := calcDetailHeight(entry, contentH, w)
	listH := contentH - detailH
	if listH < 3 {
		listH = 3
		detailH = contentH - listH
	}
	if detailH < 3 {
		detailH = 3
		listH = contentH - detailH
	}
	if listH < 1 {
		listH = 1
	}
	list := RenderList(m.State, listH, w)
	divider := Divider(w)
	detail := RenderDetail(entry, detailH, w, m.State.DetailScroll)
	return titleBar + "\n" + list + "\n" + divider + "\n" + detail + "\n" + footer
}

func (m Model) renderWithHelp(w, h int) string {
	titleBar := RenderTitleBar(m.State, w)
	footer := RenderFooter(int(m.State.Mode), m.State.StatusMsg, w)
	helpH := h - 2
	help := RenderHelp(helpH, w)
	return titleBar + "\n" + help + "\n" + footer
}

func (m Model) renderWithNotes(w, h int) string {
	var bg string
	if m.State.PrevMode == app.ModeDetail {
		bg = m.renderWithDetail(w, h)
	} else {
		bg = m.renderNormal(w, h)
	}
	bgLines := splitLines(bg)
	lineNum := m.State.NoteLineIdx + 1
	modal := RenderNotesModal(m.State.CurrentNote, m.State.NoteCursorPos, lineNum, h-2, w)
	modalLines := splitLines(modal)
	return overlayModal(bgLines, modalLines, w, h-2)
}

func (m Model) renderWithLookup(w, h int) string {
	bg := m.renderNormal(w, h)
	bgLines := splitLines(bg)
	modal := RenderLookupModal(m.State.LookupQuery, m.State.LookupResults, m.State.LookupCursor, h-2, w)
	modalLines := splitLines(modal)
	return overlayModal(bgLines, modalLines, w, h-2)
}

func (m Model) renderWithSignal(w, h int) string {
	bg := m.renderNormal(w, h)
	bgLines := splitLines(bg)
	modal := RenderSignalModal(m.State.SignalResult, h-2, w)
	modalLines := splitLines(modal)
	return overlayModal(bgLines, modalLines, w, h-2)
}

func (m Model) renderWithFilter(w, h int) string {
	bg := m.renderNormal(w, h)
	bgLines := splitLines(bg)
	modal := RenderFilterModal(m.State.FilterQuery, int(m.State.LevelFilter), h-2, w)
	modalLines := splitLines(modal)
	return overlayModal(bgLines, modalLines, w, h-2)
}

func calcDetailHeight(entry *logx.Entry, maxH, w int) int {
	if entry == nil {
		return 5
	}
	lines := 3
	if entry.Timestamp != "" {
		lines++
	}
	if entry.Level != logx.LevelUnknown {
		lines++
	}
	if entry.IsJSON && entry.Fields != nil {
		lines += Min(len(entry.Fields)*2, 12) + 2
	} else {
		rawLines := len(entry.Raw) / Max(w-4, 20)
		lines += Min(rawLines+3, 8)
	}
	if entry.IsStack {
		lines += 5
	}
	if lines < 8 {
		lines = 8
	}
	if lines > maxH*2/3 {
		lines = maxH * 2 / 3
	}
	if lines > maxH-4 {
		lines = maxH - 4
	}
	return lines
}

func splitLines(s string) []string {
	var lines []string
	var current []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, string(current))
			current = nil
		} else if s[i] != '\r' {
			current = append(current, s[i])
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}

func overlayModal(bg, modal []string, w, h int) string {
	result := make([]string, len(bg))
	copy(result, bg)
	for i, line := range modal {
		if i < len(result) && line != "" {
			trimmed := trimTrailingSpaces(line)
			if trimmed != "" {
				result[i] = line
			}
		}
	}
	return joinLines(result)
}

func trimTrailingSpaces(s string) string {
	end := len(s)
	for end > 0 && s[end-1] == ' ' {
		end--
	}
	return s[:end]
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n" + lines[i]
	}
	return result
}
