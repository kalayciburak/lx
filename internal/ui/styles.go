package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/kalayciburak/lx/internal/logx"
)

var (
	ColorAccent = lipgloss.Color("#FF6B00")

	ColorError   = lipgloss.Color("#E54B4B")
	ColorWarn    = lipgloss.Color("#D4915D")
	ColorInfo    = lipgloss.Color("#5B8FB9")
	ColorDebug   = lipgloss.Color("#8B7CB3")
	ColorSuccess = lipgloss.Color("#6B9B6B")

	ColorBg       = lipgloss.Color("#0C0C10")
	ColorBgAlt    = lipgloss.Color("#121218")
	ColorBgPanel  = lipgloss.Color("#2A2A38")
	ColorBgSelect = lipgloss.Color("#1E1E28")

	ColorTextPrimary   = lipgloss.Color("#D0D0DC")
	ColorTextSecondary = lipgloss.Color("#8888A0")
	ColorTextMuted     = lipgloss.Color("#505068")
	ColorTextBright    = lipgloss.Color("#EEEEF8")

	ColorBorder  = lipgloss.Color("#282838")
	ColorDivider = lipgloss.Color("#303040")
	ColorNoteBox = lipgloss.Color("#3D3D50")

	StyleLogo = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleLogoText = lipgloss.NewStyle().
			Foreground(ColorTextPrimary).
			Bold(true)

	StyleBar = lipgloss.NewStyle().
			Background(ColorBgPanel).
			Foreground(ColorTextPrimary)

	StyleBarText = lipgloss.NewStyle().
			Background(ColorBgPanel).
			Foreground(ColorTextSecondary)

	StyleBarAccent = lipgloss.NewStyle().
			Background(ColorBgPanel).
			Foreground(ColorAccent).
			Bold(true)

	StyleBarHighlight = lipgloss.NewStyle().
				Background(ColorBgPanel).
				Foreground(ColorTextBright).
				Bold(true)

	StyleBarDim = lipgloss.NewStyle().
			Background(ColorBgPanel).
			Foreground(ColorTextMuted)

	StyleDivider = lipgloss.NewStyle().
			Foreground(ColorDivider)

	StyleLineNum = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	StyleLineNumSelected = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	StyleCursorIndicator = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	StyleSelectedLine = lipgloss.NewStyle().
				Background(ColorBgSelect).
				Foreground(ColorTextBright)

	StyleTimestamp = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	StyleMessage = lipgloss.NewStyle().
			Foreground(ColorTextPrimary)

	StyleStack = lipgloss.NewStyle().
			Foreground(ColorTextSecondary).
			Italic(true)

	StyleLevelError = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorError).
			Bold(true).
			Padding(0, 1)

	StyleLevelWarn = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorWarn).
			Bold(true).
			Padding(0, 1)

	StyleLevelInfo = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorInfo).
			Bold(true).
			Padding(0, 1)

	StyleLevelDebug = lipgloss.NewStyle().
			Foreground(ColorTextPrimary).
			Background(ColorBgPanel).
			Padding(0, 1)

	StyleLevelTrace = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Background(ColorBgAlt).
			Padding(0, 1)

	StyleLevelUnknown = lipgloss.NewStyle().
				Foreground(ColorTextMuted).
				Padding(0, 1)

	StyleFilter = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleFilterInput = lipgloss.NewStyle().
			Foreground(ColorTextBright).
			Bold(true)

	StyleFilterActive = lipgloss.NewStyle().
				Foreground(ColorAccent)

	StyleStatus = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StyleEmpty = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	StyleEmptyBox = lipgloss.NewStyle().
			Foreground(ColorBorder)

	StyleDetailHeader = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	StyleDetailLabel = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleDetailValue = lipgloss.NewStyle().
			Foreground(ColorTextPrimary)

	StyleDetailDim = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	StyleJSONKey = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleJSONString = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StyleJSONNumber = lipgloss.NewStyle().
			Foreground(ColorInfo)

	StyleJSONBool = lipgloss.NewStyle().
			Foreground(ColorDebug)

	StyleJSONNull = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	StyleHelp = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2).
			Background(ColorBgPanel)

	StyleHelpSection = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleHelpKey = lipgloss.NewStyle().
			Foreground(ColorTextBright).
			Bold(true)

	StyleHelpDesc = lipgloss.NewStyle().
			Foreground(ColorTextSecondary)

	StyleFooter = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	StyleCredit = lipgloss.NewStyle().
			Foreground(ColorTextSecondary)

	StyleNotesInput = lipgloss.NewStyle().
			Foreground(ColorTextPrimary)

	StyleNotesHeader = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleNoteIndicator = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	StyleNoteBox = lipgloss.NewStyle().
			Background(ColorNoteBox).
			Foreground(ColorTextPrimary)

	StyleNoteBoxBorder = lipgloss.NewStyle().
				Background(ColorNoteBox).
				Foreground(ColorAccent)

	StyleLookupInput = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleLookupResult = lipgloss.NewStyle().
			Foreground(ColorTextPrimary)

	StyleLookupSelected = lipgloss.NewStyle().
				Background(ColorAccent).
				Foreground(ColorBg).
				Bold(true)

	StyleFrameBorder = lipgloss.NewStyle().
				Foreground(ColorBorder)

	StyleModalInner = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorTextPrimary)

	StyleModalText = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorTextSecondary)

	StyleModalHighlight = lipgloss.NewStyle().
				Background(ColorBg).
				Foreground(ColorTextBright).
				Bold(true)

	StyleModalDim = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorTextMuted)

	StyleModalAccent = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorAccent).
			Bold(true)
)

func LevelStyle(level logx.Level) lipgloss.Style {
	switch level {
	case logx.LevelError:
		return StyleLevelError
	case logx.LevelWarn:
		return StyleLevelWarn
	case logx.LevelInfo:
		return StyleLevelInfo
	case logx.LevelDebug:
		return StyleLevelDebug
	case logx.LevelTrace:
		return StyleLevelTrace
	default:
		return StyleLevelUnknown
	}
}
