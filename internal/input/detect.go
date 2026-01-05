package input

import (
	"os"
	"path/filepath"
)

type Mode int

const (
	ModeClipboard Mode = iota
	ModeFile
	ModePipe
)

func (m Mode) String() string {
	switch m {
	case ModeFile:
		return "FILE"
	case ModePipe:
		return "PIPE"
	default:
		return "CLIPBOARD"
	}
}

type Source struct {
	Mode     Mode
	FileName string
	Content  []string
}

func Detect(args []string) (*Source, error) {
	if len(args) > 0 {
		fileName := args[0]
		lines, err := ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		return &Source{
			Mode:     ModeFile,
			FileName: filepath.Base(fileName),
			Content:  lines,
		}, nil
	}

	if !isTerminal(os.Stdin) {
		lines, err := ReadStdin()
		if err != nil {
			return nil, err
		}
		return &Source{
			Mode:    ModePipe,
			Content: lines,
		}, nil
	}

	return &Source{
		Mode:    ModeClipboard,
		Content: nil,
	}, nil
}

func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
