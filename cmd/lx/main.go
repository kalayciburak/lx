package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kalayciburak/lx/internal/app"
	"github.com/kalayciburak/lx/internal/input"
	"github.com/kalayciburak/lx/internal/logx"
	"github.com/kalayciburak/lx/internal/ui"
)

func main() {
	source, err := input.Detect(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var entries []logx.Entry
	if source != nil && len(source.Content) > 0 {
		entries = logx.ParseLines(source.Content)
	}

	var inputMode input.Mode
	var fileName string
	if source != nil {
		inputMode = source.Mode
		fileName = source.FileName
	}

	state := app.NewState(entries, inputMode, fileName)

	model := ui.NewModel(state)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
