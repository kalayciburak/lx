package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kalayciburak/lx/internal/app"
	"github.com/kalayciburak/lx/internal/input"
	"github.com/kalayciburak/lx/internal/logx"
	"github.com/kalayciburak/lx/internal/lookup"
	"github.com/kalayciburak/lx/internal/signal"
)

const (
	MaxCopyLines       = 1000
	MaxSelectLines     = 3000
	MaxTextFilterLines = 15000
	LoadingBatchSize   = 10000
)

type LoadingBatchMsg struct {
	Entries []logx.Entry
}

type LoadingCompleteMsg struct{}

type LiveBatchMsg struct {
	Entries []logx.Entry
}

type LiveStoppedMsg struct{}

func sanitizeForClipboard(s string) string {
	return strings.ReplaceAll(s, "\x00", "")
}

func getFileCompletions(pathPrefix string) []string {
	if pathPrefix == "" {
		pathPrefix = "."
	}

	dir := filepath.Dir(pathPrefix)
	base := filepath.Base(pathPrefix)

	if strings.HasSuffix(pathPrefix, string(filepath.Separator)) || strings.HasSuffix(pathPrefix, "/") {
		dir = pathPrefix
		base = ""
	}

	if strings.HasPrefix(pathPrefix, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			pathPrefix = strings.Replace(pathPrefix, "~", home, 1)
			dir = filepath.Dir(pathPrefix)
			base = filepath.Base(pathPrefix)
			if strings.HasSuffix(pathPrefix, string(filepath.Separator)) || strings.HasSuffix(pathPrefix, "/") {
				dir = pathPrefix
				base = ""
			}
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string
	baseLower := strings.ToLower(base)

	for _, entry := range entries {
		name := entry.Name()
		nameLower := strings.ToLower(name)

		if base == "" || strings.HasPrefix(nameLower, baseLower) {
			fullPath := filepath.Join(dir, name)
			if entry.IsDir() {
				fullPath += string(filepath.Separator)
			}
			matches = append(matches, fullPath)
		}
	}

	sort.Strings(matches)
	return matches
}

func completeFilePath(pathPrefix string) (completed string, suggestions []string) {
	matches := getFileCompletions(pathPrefix)

	if len(matches) == 0 {
		return pathPrefix, nil
	}

	if len(matches) == 1 {
		return matches[0], matches
	}

	common := matches[0]
	for _, m := range matches[1:] {
		for i := 0; i < len(common) && i < len(m); i++ {
			if common[i] != m[i] {
				common = common[:i]
				break
			}
		}
		if len(m) < len(common) {
			common = common[:len(m)]
		}
	}

	if len(common) > len(pathPrefix) {
		return common, matches
	}

	return pathPrefix, matches
}

type Model struct {
	Workspaces      []*app.State
	ActiveWorkspace int
	State           *app.State
	Width           int
	Height          int
}

func NewModel(state *app.State) Model {
	workspaces := []*app.State{state}
	return Model{
		Workspaces:      workspaces,
		ActiveWorkspace: 0,
		State:           state,
		Width:           80,
		Height:          24,
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
	case LoadingBatchMsg:
		m.State.AppendEntries(msg.Entries)
		m.State.StatusMsg = "Loading... " + Itoa(m.State.LoadingProgress) + " lines"
		return m, nil
	case LoadingCompleteMsg:
		m.State.FinishLoading()
		m.State.StatusMsg = "Loaded " + Itoa(len(m.State.Entries)) + " lines"
		return m, nil
	case LiveBatchMsg:
		m.State.AppendEntries(msg.Entries)
		return m, nil
	case LiveStoppedMsg:
		m.State.IsLive = false
		m.State.StatusMsg = "Stream ended"
		return m, nil
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
	case app.ModeOpenFile:
		return m.handleOpenFileMode(msg)
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
	case IsKey(msg, KeyS):
		idx := m.State.SelectedIndex()
		if idx >= 0 {
			m.State.ToggleSelection(idx)
			if m.State.IsSelected(idx) {
				m.State.StatusMsg = "Selected (" + Itoa(m.State.SelectionCount()) + " total)"
			} else {
				if m.State.SelectionCount() > 0 {
					m.State.StatusMsg = Itoa(m.State.SelectionCount()) + " selected"
				} else {
					m.State.StatusMsg = ""
				}
			}
		}
	case IsKey(msg, KeyShiftS):
		if m.State.SelectionCount() > 0 {
			m.State.ClearSelection()
			m.State.StatusMsg = "Selection cleared"
		} else if len(m.State.Filtered) > MaxSelectLines {
			m.State.StatusMsg = "Too many lines (" + Itoa(len(m.State.Filtered)) + "). Max " + Itoa(MaxSelectLines)
		} else {
			m.State.SelectAll()
			m.State.StatusMsg = "Selected all (" + Itoa(m.State.SelectionCount()) + ")"
		}
	case IsKey(msg, KeyShiftD):
		idx := m.State.SelectedIndex()
		if idx >= 0 && m.State.HasNote(idx) {
			m.State.DeleteNote(idx)
			m.State.StatusMsg = "Note deleted"
		} else {
			m.State.StatusMsg = "No note to delete"
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
		if len(m.State.Filtered) > MaxCopyLines {
			m.State.StatusMsg = "Too many lines (" + Itoa(len(m.State.Filtered)) + "). Max " + Itoa(MaxCopyLines)
		} else {
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
		}
	case IsKey(msg, KeyC):
		if m.State.SelectionCount() > MaxCopyLines {
			m.State.StatusMsg = "Too many selected (" + Itoa(m.State.SelectionCount()) + "). Max " + Itoa(MaxCopyLines)
		} else if m.State.SelectionCount() > 0 {
			indices := m.State.SelectedIndices()
			entries := m.State.SelectedEntries()
			content := app.ExportLogsWithNotes(entries, m.State.Notes, indices)
			if err := clipboard.WriteAll(sanitizeForClipboard(content)); err != nil {
				m.State.StatusMsg = "Clipboard error"
			} else {
				m.State.StatusMsg = "Copied " + Itoa(len(indices)) + " selected lines"
				m.State.ClearSelection()
			}
		} else {
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
		}
	case IsKey(msg, KeyD):
		count := m.State.SelectionCount()
		m.State.DeleteSelected()
		if count > 0 {
			m.State.StatusMsg = "Deleted " + Itoa(count) + " lines"
		} else {
			m.State.StatusMsg = "Deleted"
		}
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
	case IsKey(msg, KeyO):
		m.State.OpenFilePath = ""
		m.State.OpenFileCursor = 0
		m.State.Mode = app.ModeOpenFile
	case IsKey(msg, KeyShiftT):
		if len(m.Workspaces) >= 10 {
			m.State.StatusMsg = "Max 10 workspaces allowed"
		} else {
			newState := app.NewState(nil, input.ModeClipboard, "")
			m.Workspaces = append(m.Workspaces, newState)
			m.ActiveWorkspace = len(m.Workspaces) - 1
			m.State = m.Workspaces[m.ActiveWorkspace]
			m.State.StatusMsg = "Workspace " + Itoa(m.ActiveWorkspace+1) + " created"
		}
	case IsKey(msg, KeyTab):
		if len(m.Workspaces) > 1 {
			m.ActiveWorkspace = (m.ActiveWorkspace + 1) % len(m.Workspaces)
			m.State = m.Workspaces[m.ActiveWorkspace]
			m.State.StatusMsg = "Workspace " + Itoa(m.ActiveWorkspace+1) + "/" + Itoa(len(m.Workspaces))
		}
	case IsKey(msg, KeyShiftTab):
		if len(m.Workspaces) > 1 {
			m.ActiveWorkspace = (m.ActiveWorkspace - 1 + len(m.Workspaces)) % len(m.Workspaces)
			m.State = m.Workspaces[m.ActiveWorkspace]
			m.State.StatusMsg = "Workspace " + Itoa(m.ActiveWorkspace+1) + "/" + Itoa(len(m.Workspaces))
		}
	case IsKey(msg, KeyShiftW):
		if len(m.Workspaces) > 1 {
			m.Workspaces = append(m.Workspaces[:m.ActiveWorkspace], m.Workspaces[m.ActiveWorkspace+1:]...)
			if m.ActiveWorkspace >= len(m.Workspaces) {
				m.ActiveWorkspace = len(m.Workspaces) - 1
			}
			m.State = m.Workspaces[m.ActiveWorkspace]
			m.State.StatusMsg = "Workspace closed (" + Itoa(len(m.Workspaces)) + " remaining)"
		} else {
			m.State.StatusMsg = "Cannot close last workspace"
		}
	case IsKey(msg, KeyU):
		count := m.State.Undo()
		if count > 0 {
			m.State.StatusMsg = "Restored " + Itoa(count) + " lines"
		} else {
			m.State.StatusMsg = "Nothing to undo"
		}
	case IsKey(msg, KeyShiftU):
		count := m.State.Redo()
		if count > 0 {
			m.State.StatusMsg = "Re-deleted " + Itoa(count) + " lines"
		} else {
			m.State.StatusMsg = "Nothing to redo"
		}
	}
	return m, nil
}

func (m Model) handleFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tooManyLines := len(m.State.Entries) > MaxTextFilterLines

	switch {
	case IsKey(msg, KeyEsc):
		m.State.Mode = app.ModeList
	case IsKey(msg, KeyEnter):
		m.State.Mode = app.ModeList
		if !tooManyLines || m.State.FilterQuery == "" {
			m.State.Refilter()
		}
	case IsKey(msg, KeyTab):
		m.State.CycleLevelFilter()
	case IsKey(msg, KeyBackspace):
		if len(m.State.FilterQuery) > 0 {
			m.State.FilterQuery = m.State.FilterQuery[:len(m.State.FilterQuery)-1]
			if !tooManyLines {
				m.State.Refilter()
			}
		}
	case IsKey(msg, KeyCtrlC):
		return m, tea.Quit
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			if tooManyLines {
				m.State.StatusMsg = "Text filter disabled (>" + Itoa(MaxTextFilterLines) + " lines)"
			} else {
				m.State.FilterQuery += msg.String()
				m.State.Refilter()
			}
		}
	}
	return m, nil
}

func (m Model) handleDetailMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case IsKey(msg, KeyEsc):
		if m.State.DetailMaximized {
			m.State.DetailMaximized = false
		} else {
			m.State.Mode = app.ModeList
		}
	case IsKey(msg, KeyEnter, KeySpace):
		m.State.Mode = app.ModeList
		m.State.DetailMaximized = false
	case IsKey(msg, KeyZ):
		m.State.DetailMaximized = !m.State.DetailMaximized
		m.State.DetailScroll = 0
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
		if m.State.SelectionCount() > MaxCopyLines {
			m.State.StatusMsg = "Too many selected (" + Itoa(m.State.SelectionCount()) + "). Max " + Itoa(MaxCopyLines)
		} else if m.State.SelectionCount() > 0 {
			indices := m.State.SelectedIndices()
			entries := m.State.SelectedEntries()
			content := app.ExportLogsWithNotes(entries, m.State.Notes, indices)
			if err := clipboard.WriteAll(sanitizeForClipboard(content)); err != nil {
				m.State.StatusMsg = "Clipboard error"
			} else {
				m.State.StatusMsg = "Copied " + Itoa(len(indices)) + " selected lines"
				m.State.ClearSelection()
			}
		} else {
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
		if len(m.State.Filtered) > MaxCopyLines {
			m.State.StatusMsg = "Too many lines (" + Itoa(len(m.State.Filtered)) + "). Max " + Itoa(MaxCopyLines)
		} else {
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
		}
	case IsKey(msg, KeyD):
		count := m.State.SelectionCount()
		m.State.DeleteSelected()
		if count > 0 {
			m.State.StatusMsg = "Deleted " + Itoa(count) + " lines"
		} else {
			m.State.StatusMsg = "Deleted"
		}
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
	case IsKey(msg, KeyShiftT):
		if len(m.Workspaces) >= 10 {
			m.State.StatusMsg = "Max 10 workspaces allowed"
		} else {
			newState := app.NewState(nil, input.ModeClipboard, "")
			m.Workspaces = append(m.Workspaces, newState)
			m.ActiveWorkspace = len(m.Workspaces) - 1
			m.State = m.Workspaces[m.ActiveWorkspace]
			m.State.StatusMsg = "Workspace " + Itoa(m.ActiveWorkspace+1) + " created"
		}
	case IsKey(msg, KeyTab):
		if len(m.Workspaces) > 1 {
			m.ActiveWorkspace = (m.ActiveWorkspace + 1) % len(m.Workspaces)
			m.State = m.Workspaces[m.ActiveWorkspace]
			m.State.StatusMsg = "Workspace " + Itoa(m.ActiveWorkspace+1) + "/" + Itoa(len(m.Workspaces))
		}
	case IsKey(msg, KeyShiftTab):
		if len(m.Workspaces) > 1 {
			m.ActiveWorkspace = (m.ActiveWorkspace - 1 + len(m.Workspaces)) % len(m.Workspaces)
			m.State = m.Workspaces[m.ActiveWorkspace]
			m.State.StatusMsg = "Workspace " + Itoa(m.ActiveWorkspace+1) + "/" + Itoa(len(m.Workspaces))
		}
	case IsKey(msg, KeyShiftW):
		if len(m.Workspaces) > 1 {
			m.Workspaces = append(m.Workspaces[:m.ActiveWorkspace], m.Workspaces[m.ActiveWorkspace+1:]...)
			if m.ActiveWorkspace >= len(m.Workspaces) {
				m.ActiveWorkspace = len(m.Workspaces) - 1
			}
			m.State = m.Workspaces[m.ActiveWorkspace]
			m.State.StatusMsg = "Workspace closed (" + Itoa(len(m.Workspaces)) + " remaining)"
		} else {
			m.State.StatusMsg = "Cannot close last workspace"
		}
	case IsKey(msg, KeyS):
		idx := m.State.SelectedIndex()
		if idx >= 0 {
			m.State.ToggleSelection(idx)
			if m.State.IsSelected(idx) {
				m.State.StatusMsg = "Selected (" + Itoa(m.State.SelectionCount()) + " total)"
			} else {
				if m.State.SelectionCount() > 0 {
					m.State.StatusMsg = Itoa(m.State.SelectionCount()) + " selected"
				} else {
					m.State.StatusMsg = ""
				}
			}
		}
	case IsKey(msg, KeyShiftS):
		if m.State.SelectionCount() > 0 {
			m.State.ClearSelection()
			m.State.StatusMsg = "Selection cleared"
		} else if len(m.State.Filtered) > MaxSelectLines {
			m.State.StatusMsg = "Too many lines (" + Itoa(len(m.State.Filtered)) + "). Max " + Itoa(MaxSelectLines)
		} else {
			m.State.SelectAll()
			m.State.StatusMsg = "Selected all (" + Itoa(m.State.SelectionCount()) + ")"
		}
	case IsKey(msg, KeyShiftD):
		idx := m.State.SelectedIndex()
		if idx >= 0 && m.State.HasNote(idx) {
			m.State.DeleteNote(idx)
			m.State.StatusMsg = "Note deleted"
		} else {
			m.State.StatusMsg = "No note to delete"
		}
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
	case IsKey(msg, KeyO):
		m.State.OpenFilePath = ""
		m.State.OpenFileCursor = 0
		m.State.Mode = app.ModeOpenFile
	case IsKey(msg, KeyU):
		count := m.State.Undo()
		if count > 0 {
			m.State.StatusMsg = "Restored " + Itoa(count) + " lines"
		} else {
			m.State.StatusMsg = "Nothing to undo"
		}
	case IsKey(msg, KeyShiftU):
		count := m.State.Redo()
		if count > 0 {
			m.State.StatusMsg = "Re-deleted " + Itoa(count) + " lines"
		} else {
			m.State.StatusMsg = "Nothing to redo"
		}
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
	case IsKey(msg, KeyCtrlV):
		content, err := clipboard.ReadAll()
		if err != nil {
			m.State.StatusMsg = "Clipboard error"
		} else {
			content = sanitizeForClipboard(content)
			content = strings.ReplaceAll(content, "\n", " ")
			content = strings.ReplaceAll(content, "\r", " ")
			content = strings.ReplaceAll(content, "\t", " ")
			toInsert := []rune(content)
			remaining := 100 - len(runes)
			if remaining > 0 {
				if len(toInsert) > remaining {
					toInsert = toInsert[:remaining]
					m.State.StatusMsg = "Pasted (truncated)"
				} else {
					m.State.StatusMsg = "Pasted"
				}
				newRunes := make([]rune, 0, len(runes)+len(toInsert))
				newRunes = append(newRunes, runes[:cursorPos]...)
				newRunes = append(newRunes, toInsert...)
				newRunes = append(newRunes, runes[cursorPos:]...)
				m.State.CurrentNote = string(newRunes)
				m.State.NoteCursorPos = cursorPos + len(toInsert)
			} else {
				m.State.StatusMsg = "Note full"
			}
		}
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
		if len(charRunes) == 1 && len(runes) < 100 {
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
	case IsKey(msg, KeyEnter, KeySpace, KeyC):
		if result := m.State.SelectedLookupResult(); result != nil {
			content := Itoa(result.Code) + " " + result.Name + "\n" + result.Description + "\nExample: " + result.Example
			if err := clipboard.WriteAll(sanitizeForClipboard(content)); err != nil {
				m.State.StatusMsg = "Clipboard error"
			} else {
				m.State.StatusMsg = "Copied: " + result.Name
			}
			m.State.Mode = app.ModeList
			m.State.LookupQuery = ""
			m.State.LookupResults = nil
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

func (m Model) handleOpenFileMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	runes := []rune(m.State.OpenFilePath)
	cursorPos := m.State.OpenFileCursor

	if cursorPos > len(runes) {
		cursorPos = len(runes)
	}
	if cursorPos < 0 {
		cursorPos = 0
	}

	clearSuggestions := func() {
		m.State.OpenFileSuggestions = nil
		m.State.OpenFileSuggIdx = 0
	}

	switch {
	case IsKey(msg, KeyEsc):
		m.State.Mode = app.ModeList
		m.State.OpenFilePath = ""
		clearSuggestions()
	case IsKey(msg, KeyEnter):
		if len(m.State.OpenFileSuggestions) > 0 && m.State.OpenFileSuggIdx < len(m.State.OpenFileSuggestions) {
			selected := m.State.OpenFileSuggestions[m.State.OpenFileSuggIdx]
			if strings.HasSuffix(selected, string(filepath.Separator)) {
				m.State.OpenFilePath = selected
				m.State.OpenFileCursor = len([]rune(selected))
				clearSuggestions()
				return m, nil
			}
			m.State.OpenFilePath = selected
		}
		if m.State.OpenFilePath != "" {
			if err := m.State.LoadFromFile(m.State.OpenFilePath); err != nil {
				m.State.StatusMsg = "Error: " + err.Error()
			} else {
				m.State.StatusMsg = "Loaded " + Itoa(len(m.State.Entries)) + " lines"
			}
		}
		m.State.Mode = app.ModeList
		m.State.OpenFilePath = ""
		clearSuggestions()
	case IsKey(msg, KeyTab):
		completed, suggestions := completeFilePath(m.State.OpenFilePath)
		m.State.OpenFilePath = completed
		m.State.OpenFileCursor = len([]rune(completed))
		m.State.OpenFileSuggestions = suggestions
		m.State.OpenFileSuggIdx = 0
	case IsKey(msg, KeyDown):
		if len(m.State.OpenFileSuggestions) > 0 {
			m.State.OpenFileSuggIdx = (m.State.OpenFileSuggIdx + 1) % len(m.State.OpenFileSuggestions)
		}
	case IsKey(msg, KeyUp):
		if len(m.State.OpenFileSuggestions) > 0 {
			m.State.OpenFileSuggIdx = (m.State.OpenFileSuggIdx - 1 + len(m.State.OpenFileSuggestions)) % len(m.State.OpenFileSuggestions)
		}
	case IsKey(msg, KeyLeft):
		if cursorPos > 0 {
			m.State.OpenFileCursor = cursorPos - 1
		}
		clearSuggestions()
	case IsKey(msg, KeyRight):
		if cursorPos < len(runes) {
			m.State.OpenFileCursor = cursorPos + 1
		}
		clearSuggestions()
	case IsKey(msg, KeyHome):
		m.State.OpenFileCursor = 0
		clearSuggestions()
	case IsKey(msg, KeyEnd):
		m.State.OpenFileCursor = len(runes)
		clearSuggestions()
	case IsKey(msg, KeyBackspace):
		if cursorPos > 0 {
			newRunes := append(runes[:cursorPos-1], runes[cursorPos:]...)
			m.State.OpenFilePath = string(newRunes)
			m.State.OpenFileCursor = cursorPos - 1
		}
		clearSuggestions()
	case IsKey(msg, KeyDelete):
		if cursorPos < len(runes) {
			newRunes := append(runes[:cursorPos], runes[cursorPos+1:]...)
			m.State.OpenFilePath = string(newRunes)
		}
		clearSuggestions()
	case IsKey(msg, KeyCtrlC):
		return m, tea.Quit
	default:
		char := msg.String()
		charRunes := []rune(char)
		if len(charRunes) == 1 && len(runes) < 512 {
			newRunes := make([]rune, 0, len(runes)+1)
			newRunes = append(newRunes, runes[:cursorPos]...)
			newRunes = append(newRunes, charRunes[0])
			newRunes = append(newRunes, runes[cursorPos:]...)
			m.State.OpenFilePath = string(newRunes)
			m.State.OpenFileCursor = cursorPos + 1
		}
		clearSuggestions()
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
	case app.ModeOpenFile:
		content = m.renderWithOpenFile(w, h)
	default:
		content = m.renderNormal(w, h)
	}
	return Frame(content, m.Width, m.Height)
}

func (m Model) renderNormal(w, h int) string {
	titleBar := RenderTitleBar(m.State, m.ActiveWorkspace, len(m.Workspaces), w)
	footer := RenderFooter(int(m.State.Mode), m.State.StatusMsg, m.State.SelectionCount(), w)
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
	titleBar := RenderTitleBar(m.State, m.ActiveWorkspace, len(m.Workspaces), w)
	footer := RenderFooter(int(m.State.Mode), m.State.StatusMsg, m.State.SelectionCount(), w)
	entry := m.State.SelectedEntry()

	if m.State.DetailMaximized {
		detailH := h - 2
		if detailH < 6 {
			detailH = 6
		}
		detail := RenderDetail(entry, detailH, w, m.State.DetailScroll)
		return titleBar + "\n" + detail + "\n" + footer
	}

	contentH := h - 3
	if contentH < 6 {
		contentH = 6
	}
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
	titleBar := RenderTitleBar(m.State, m.ActiveWorkspace, len(m.Workspaces), w)
	footer := RenderFooter(int(m.State.Mode), m.State.StatusMsg, m.State.SelectionCount(), w)
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
	textFilterDisabled := len(m.State.Entries) > MaxTextFilterLines
	modal := RenderFilterModal(m.State.FilterQuery, int(m.State.LevelFilter), h-2, w, textFilterDisabled)
	modalLines := splitLines(modal)
	return overlayModal(bgLines, modalLines, w, h-2)
}

func (m Model) renderWithOpenFile(w, h int) string {
	bg := m.renderNormal(w, h)
	bgLines := splitLines(bg)
	modal := RenderOpenFileModal(m.State.OpenFilePath, m.State.OpenFileCursor, m.State.OpenFileSuggestions, m.State.OpenFileSuggIdx, h-2, w)
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
