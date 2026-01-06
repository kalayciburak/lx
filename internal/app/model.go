package app

import (
	"time"

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
	ModeOpenFile
)

type LevelFilter int

const (
	LevelFilterAll LevelFilter = iota
	LevelFilterError
	LevelFilterWarn
	LevelFilterInfo
	LevelFilterDebug
	LevelFilterTrace
)

var LevelFilters = []LevelFilter{LevelFilterAll, LevelFilterError, LevelFilterWarn, LevelFilterInfo, LevelFilterDebug, LevelFilterTrace}

type NoteLevel int

const (
	NoteLevelNormal NoteLevel = iota
	NoteLevelCritical
	NoteLevelUnsure
)

func (nl NoteLevel) String() string {
	switch nl {
	case NoteLevelCritical:
		return "CRIT"
	case NoteLevelUnsure:
		return "UNSURE"
	default:
		return ""
	}
}

func (nl NoteLevel) Symbol() string {
	switch nl {
	case NoteLevelCritical:
		return "!"
	case NoteLevelUnsure:
		return "?"
	default:
		return ""
	}
}

type Note struct {
	Text      string
	Level     NoteLevel
	CreatedAt time.Time
}

func ParseNoteLevel(text string) (NoteLevel, string) {
	if len(text) == 0 {
		return NoteLevelNormal, text
	}
	switch text[0] {
	case '!':
		return NoteLevelCritical, text[1:]
	case '?':
		return NoteLevelUnsure, text[1:]
	default:
		return NoteLevelNormal, text
	}
}

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
	case LevelFilterTrace:
		return "TRACE"
	default:
		return "ALL"
	}
}

type State struct {
	Entries  []logx.Entry
	Filtered []int

	Cursor   int
	Selected map[int]bool

	FilterQuery string
	LevelFilter LevelFilter

	Mode Mode

	InputMode input.Mode
	FileName  string

	Notes         map[int]Note
	CurrentNote   string
	NoteCursorPos int
	NoteLineIdx   int
	ShowingNotes  map[int]bool

	LookupQuery   string
	LookupResults []lookup.StatusInfo
	LookupCursor  int

	DetailScroll    int
	DetailMaximized bool
	SignalResult    *signal.SignalResult

	OpenFilePath        string
	OpenFileCursor      int
	OpenFileSuggestions []string
	OpenFileSuggIdx     int

	UndoStack [][]int
	RedoStack [][]int

	PrevMode Mode

	StatusMsg string

	IsLoading       bool
	IsLive          bool
	LoadingProgress int
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
		Notes:        make(map[int]Note),
		NoteLineIdx:  -1,
		ShowingNotes: make(map[int]bool),
		Selected:     make(map[int]bool),
	}
}

func NewLoadingState(inputMode input.Mode, fileName string) *State {
	return &State{
		Entries:      make([]logx.Entry, 0),
		Filtered:     make([]int, 0),
		InputMode:    inputMode,
		FileName:     fileName,
		Mode:         ModeList,
		Notes:        make(map[int]Note),
		NoteLineIdx:  -1,
		ShowingNotes: make(map[int]bool),
		Selected:     make(map[int]bool),
		IsLoading:    true,
	}
}

func (s *State) AppendEntries(newEntries []logx.Entry) {
	startIdx := len(s.Entries)
	s.Entries = append(s.Entries, newEntries...)

	for i := range newEntries {
		if !newEntries[i].Deleted {
			s.Filtered = append(s.Filtered, startIdx+i)
		}
	}
	s.LoadingProgress = len(s.Entries)
}

func (s *State) FinishLoading() {
	s.IsLoading = false
	s.Refilter()
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
	s.DetailScroll = 0
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
	case LevelFilterTrace:
		return logx.LevelTrace
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
	s.DetailScroll = 0
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

func (s *State) LoadFromFile(path string) error {
	lines, err := input.ReadFile(path)
	if err != nil {
		return err
	}
	s.Entries = logx.ParseLines(lines)
	s.InputMode = input.ModeFile
	s.FileName = path
	s.FilterQuery = ""
	s.Refilter()
	s.Cursor = 0
	s.StatusMsg = ""
	return nil
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
	if note, ok := s.Notes[idx]; ok {
		return note.Text
	}
	return ""
}

func (s *State) GetNoteObj(idx int) (Note, bool) {
	note, ok := s.Notes[idx]
	return note, ok
}

func (s *State) SetNote(idx int, noteText string) {
	if noteText == "" {
		delete(s.Notes, idx)
		return
	}
	level, text := ParseNoteLevel(noteText)
	if text == "" {
		delete(s.Notes, idx)
		return
	}
	s.Notes[idx] = Note{
		Text:      text,
		Level:     level,
		CreatedAt: time.Now(),
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

func (s *State) FilteredNotesCount() int {
	count := 0
	for _, idx := range s.Filtered {
		if _, ok := s.Notes[idx]; ok {
			count++
		}
	}
	return count
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
		levelStr := ""
		if note.Level != NoteLevelNormal {
			levelStr = "[" + note.Level.String() + "] "
		}
		result += "Line " + itoa(idx+1) + ": " + levelStr + note.Text + "\n"
	}
	return result
}

func (s *State) ToggleSelection(idx int) {
	if s.Selected[idx] {
		delete(s.Selected, idx)
	} else {
		s.Selected[idx] = true
	}
}

func (s *State) IsSelected(idx int) bool {
	return s.Selected[idx]
}

func (s *State) SelectionCount() int {
	return len(s.Selected)
}

func (s *State) ClearSelection() {
	s.Selected = make(map[int]bool)
}

func (s *State) SelectAll() {
	for _, idx := range s.Filtered {
		s.Selected[idx] = true
	}
}

func (s *State) SelectedIndices() []int {
	var indices []int
	for idx := range s.Selected {
		indices = append(indices, idx)
	}
	for i := 0; i < len(indices); i++ {
		for j := i + 1; j < len(indices); j++ {
			if indices[i] > indices[j] {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}
	return indices
}

func (s *State) SelectedEntries() []logx.Entry {
	var entries []logx.Entry
	for _, idx := range s.SelectedIndices() {
		entries = append(entries, s.Entries[idx])
	}
	return entries
}

func (s *State) DeleteSelected() {
	indices := s.SelectedIndices()
	var deleted []int
	if len(indices) == 0 {
		if len(s.Filtered) > 0 {
			idx := s.Filtered[s.Cursor]
			s.Entries[idx].Deleted = true
			deleted = []int{idx}
		}
	} else {
		for _, idx := range indices {
			s.Entries[idx].Deleted = true
			deleted = append(deleted, idx)
		}
		s.ClearSelection()
	}
	if len(deleted) > 0 {
		s.UndoStack = append(s.UndoStack, deleted)
		s.RedoStack = nil
	}
	s.Refilter()
}

func (s *State) DeleteNote(idx int) {
	delete(s.Notes, idx)
	delete(s.ShowingNotes, idx)
}

func (s *State) Undo() int {
	if len(s.UndoStack) == 0 {
		return 0
	}
	lastIdx := len(s.UndoStack) - 1
	deleted := s.UndoStack[lastIdx]
	s.UndoStack = s.UndoStack[:lastIdx]

	count := 0
	for _, idx := range deleted {
		if idx >= 0 && idx < len(s.Entries) && s.Entries[idx].Deleted {
			s.Entries[idx].Deleted = false
			count++
		}
	}
	if count > 0 {
		s.RedoStack = append(s.RedoStack, deleted)
		s.Refilter()
	}
	return count
}

func (s *State) Redo() int {
	if len(s.RedoStack) == 0 {
		return 0
	}
	lastIdx := len(s.RedoStack) - 1
	undone := s.RedoStack[lastIdx]
	s.RedoStack = s.RedoStack[:lastIdx]

	count := 0
	for _, idx := range undone {
		if idx >= 0 && idx < len(s.Entries) && !s.Entries[idx].Deleted {
			s.Entries[idx].Deleted = true
			count++
		}
	}
	if count > 0 {
		s.UndoStack = append(s.UndoStack, undone)
		s.Refilter()
	}
	return count
}
