package main

import (
	"fmt"
	"os"
	"time"

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

	if source != nil && source.IsLive {
		state := app.NewLoadingState(inputMode, fileName)
		state.IsLive = true
		state.IsLoading = false
		model := ui.NewModel(state)
		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithInputTTY())

		lineCh := make(chan string, 1000)
		go input.StreamStdin(lineCh)

		go func() {
			var batch []string
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case line, ok := <-lineCh:
					if !ok {
						if len(batch) > 0 {
							p.Send(ui.LiveBatchMsg{Entries: logx.ParseLines(batch)})
						}
						p.Send(ui.LiveStoppedMsg{})
						return
					}
					batch = append(batch, line)
					if len(batch) >= 100 {
						p.Send(ui.LiveBatchMsg{Entries: logx.ParseLines(batch)})
						batch = nil
					}
				case <-ticker.C:
					if len(batch) > 0 {
						p.Send(ui.LiveBatchMsg{Entries: logx.ParseLines(batch)})
						batch = nil
					}
				}
			}
		}()

		p.Run()
		os.Exit(0)
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
