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

const asyncLoadingThreshold = 5000

func main() {
	source, err := input.Detect(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var inputMode input.Mode
	var fileName string
	if source != nil {
		inputMode = source.Mode
		fileName = source.FileName
	}

	if source != nil && len(source.Content) > asyncLoadingThreshold {
		state := app.NewLoadingState(inputMode, fileName)
		model := ui.NewModel(state)
		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

		go func() {
			lines := source.Content
			batchSize := ui.LoadingBatchSize

			for i := 0; i < len(lines); i += batchSize {
				end := i + batchSize
				if end > len(lines) {
					end = len(lines)
				}
				batch := logx.ParseLines(lines[i:end])
				p.Send(ui.LoadingBatchMsg{Entries: batch})
			}
			p.Send(ui.LoadingCompleteMsg{})
		}()

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
			os.Exit(1)
		}
		return
	}

	var entries []logx.Entry
	if source != nil && len(source.Content) > 0 {
		entries = logx.ParseLines(source.Content)
	}

	state := app.NewState(entries, inputMode, fileName)
	model := ui.NewModel(state)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
