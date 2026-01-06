package input

import (
	"bufio"
	"os"
	"strings"

	"github.com/atotto/clipboard"
)

func ReadFile(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

func ReadStdin() ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)

	const maxBuf = 1024 * 1024
	buf := make([]byte, maxBuf)
	scanner.Buffer(buf, maxBuf)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

func ReadClipboard() ([]string, error) {
	content, err := clipboard.ReadAll()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}
	return strings.Split(content, "\n"), nil
}

func WriteClipboard(content string) error {
	return clipboard.WriteAll(content)
}

func StreamStdin(ch chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	const maxBuf = 1024 * 1024
	buf := make([]byte, maxBuf)
	scanner.Buffer(buf, maxBuf)
	for scanner.Scan() {
		ch <- scanner.Text()
	}
	close(ch)
}
