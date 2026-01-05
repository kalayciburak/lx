package app

import (
	"github.com/kalayciburak/lx/internal/input"
	"github.com/kalayciburak/lx/internal/logx"
	"github.com/kalayciburak/lx/internal/lookup"
	"github.com/kalayciburak/lx/internal/signal"
)

type Mode int

const (
	ModeList Mode = iota
	ModeFilter
	ModeDetail
	ModeHelp
	ModeNotes
	ModeLookup
	ModeSignal
)

type LevelFilter int

const (
	LevelFilterAll LevelFilter = iota
	LevelFilterError
	LevelFilterWarn
	LevelFilterInfo
	LevelFilterDebug
)

var LevelFilters = []LevelFilter{LevelFilterAll, LevelFilterError, LevelFilterWarn, LevelFilterInfo, LevelFilterDebug}

func (lf LevelFilter) String() string {
	switch lf {
	case LevelFilterError:
		return "ERROR"
	case LevelFilterWarn:
		return "WARN"
	case LevelFilterInfo:
		return "INFO"
	case LevelFilterDebug:
		return "DEBUG"
	default:
		return "ALL"
	}
}

type State struct {
	Entries  []logx.Entry
	Filtered []int

	Cursor int

	FilterQuery string
	LevelFilter LevelFilter

	Mode Mode

	InputMode input.Mode
	FileName  string

	Notes        map[int]string
	CurrentNote  string
	NoteLineIdx  int
	ShowingNotes map[int]bool

	LookupQuery   string
	LookupResults []lookup.StatusInfo
	LookupCursor  int

	SignalResult *signal.SignalResult

	PrevMode Mode

	StatusMsg string
}

func NewState(entries []logx.Entry, inputMode input.Mode, fileName string) *State {
	filtered := make([]int, 0, len(entries))
	for i, e := range entries {
		if !e.Deleted {
			filtered = append(filtered, i)
		}
	}

	return &State{
		Entries:      entries,
		Filtered:     filtered,
		InputMode:    inputMode,
		FileName:     fileName,
		Mode:         ModeList,
		Notes:        make(map[int]string),
		NoteLineIdx:  -1,
		ShowingNotes: make(map[int]bool),
	}
}

func (s *State) Refilter() {
	var levelPtr *logx.Level
	if s.LevelFilter != LevelFilterAll {
		level := s.levelFilterToLogxLevel()
		levelPtr = &level
	}
	s.Filtered = logx.ApplyWithLevel(s.Entries, s.FilterQuery, levelPtr)
	if s.Cursor >= len(s.Filtered) {
		s.Cursor = len(s.Filtered) - 1
	}
	if s.Cursor < 0 {
		s.Cursor = 0
	}
}

func (s *State) levelFilterToLogxLevel() logx.Level {
	switch s.LevelFilter {
	case LevelFilterError:
		return logx.LevelError
	case LevelFilterWarn:
		return logx.LevelWarn
	case LevelFilterInfo:
		return logx.LevelInfo
	case LevelFilterDebug:
		return logx.LevelDebug
	default:
		return logx.LevelUnknown
	}
}

func (s *State) CycleLevelFilter() {
	currentIdx := 0
	for i, lf := range LevelFilters {
		if lf == s.LevelFilter {
			currentIdx = i
			break
		}
	}
	nextIdx := (currentIdx + 1) % len(LevelFilters)
	s.LevelFilter = LevelFilters[nextIdx]
	s.Refilter()
}

func (s *State) SelectedEntry() *logx.Entry {
	if len(s.Filtered) == 0 || s.Cursor >= len(s.Filtered) {
		return nil
	}
	return &s.Entries[s.Filtered[s.Cursor]]
}

func (s *State) SelectedIndex() int {
	if len(s.Filtered) == 0 || s.Cursor >= len(s.Filtered) {
		return -1
	}
	return s.Filtered[s.Cursor]
}

func (s *State) MoveCursor(delta int) {
	s.Cursor += delta
	if s.Cursor < 0 {
		s.Cursor = 0
	}
	if s.Cursor >= len(s.Filtered) {
		s.Cursor = len(s.Filtered) - 1
	}
	if s.Cursor < 0 {
		s.Cursor = 0
	}
}

func (s *State) DeleteSelected() {
	if len(s.Filtered) == 0 {
		return
	}
	idx := s.Filtered[s.Cursor]
	s.Entries[idx].Deleted = true
	s.Refilter()
}

func (s *State) ClearAll() {
	for i := range s.Entries {
		s.Entries[i].Deleted = true
	}
	s.Refilter()
}

func (s *State) LoadFromClipboard(lines []string) {
	s.Entries = logx.ParseLines(lines)
	s.InputMode = input.ModeClipboard
	s.FilterQuery = ""
	s.Refilter()
	s.Cursor = 0
	s.StatusMsg = ""
}

func (s *State) UpdateLookup() {
	s.LookupResults = lookup.Search(s.LookupQuery, 10)
	if s.LookupCursor >= len(s.LookupResults) {
		s.LookupCursor = len(s.LookupResults) - 1
	}
	if s.LookupCursor < 0 {
		s.LookupCursor = 0
	}
}

func (s *State) SelectedLookupResult() *lookup.StatusInfo {
	if len(s.LookupResults) == 0 || s.LookupCursor >= len(s.LookupResults) {
		return nil
	}
	return &s.LookupResults[s.LookupCursor]
}

func (s *State) VisibleEntries() []logx.Entry {
	result := make([]logx.Entry, 0, len(s.Filtered))
	for _, idx := range s.Filtered {
		result = append(result, s.Entries[idx])
	}
	return result
}

func (s *State) HasNote(idx int) bool {
	_, ok := s.Notes[idx]
	return ok
}

func (s *State) GetNote(idx int) string {
	return s.Notes[idx]
}

func (s *State) SetNote(idx int, note string) {
	if note == "" {
		delete(s.Notes, idx)
	} else {
		s.Notes[idx] = note
	}
}

func (s *State) NotedLines() []int {
	var lines []int
	for idx := range s.Notes {
		lines = append(lines, idx)
	}
	for i := 0; i < len(lines); i++ {
		for j := i + 1; j < len(lines); j++ {
			if lines[i] > lines[j] {
				lines[i], lines[j] = lines[j], lines[i]
			}
		}
	}
	return lines
}

func (s *State) NextNotedLine() int {
	currentIdx := s.SelectedIndex()
	noted := s.NotedLines()
	for _, idx := range noted {
		if idx > currentIdx {
			return idx
		}
	}
	if len(noted) > 0 {
		return noted[0]
	}
	return -1
}

func (s *State) PrevNotedLine() int {
	currentIdx := s.SelectedIndex()
	noted := s.NotedLines()
	for i := len(noted) - 1; i >= 0; i-- {
		if noted[i] < currentIdx {
			return noted[i]
		}
	}
	if len(noted) > 0 {
		return noted[len(noted)-1]
	}
	return -1
}

func (s *State) JumpToEntry(entryIdx int) bool {
	for i, idx := range s.Filtered {
		if idx == entryIdx {
			s.Cursor = i
			return true
		}
	}
	return false
}

func (s *State) TotalNotes() int {
	return len(s.Notes)
}

func (s *State) IsNoteShowing(idx int) bool {
	return s.ShowingNotes[idx]
}

func (s *State) ToggleNoteDisplay(idx int) {
	if s.ShowingNotes[idx] {
		delete(s.ShowingNotes, idx)
	} else {
		s.ShowingNotes[idx] = true
	}
}

func (s *State) ToggleAllNotesDisplay() {
	anyShown := false
	for idx := range s.Notes {
		if s.ShowingNotes[idx] {
			anyShown = true
			break
		}
	}

	if anyShown {
		s.ShowingNotes = make(map[int]bool)
	} else {
		for idx := range s.Notes {
			s.ShowingNotes[idx] = true
		}
	}
}

func (s *State) CountShowingNotes() int {
	count := 0
	for idx := range s.ShowingNotes {
		if s.HasNote(idx) {
			count++
		}
	}
	return count
}

func (s *State) AllNotesText() string {
	if len(s.Notes) == 0 {
		return ""
	}
	var result string
	for _, idx := range s.NotedLines() {
		note := s.Notes[idx]
		result += "Line " + itoa(idx+1) + ": " + note + "\n"
	}
	return result
}
