package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func SmallLogo() string {
	return StyleLogo.Render("★") + StyleLogoText.Render("LX")
}

func Logo() string {
	return StyleLogo.Render("★") + " " + StyleLogoText.Render("LX") + " " + StyleLogoText.Render("LOG X-RAY")
}

func Divider(width int) string {
	if width <= 0 {
		return ""
	}
	return StyleDivider.Render(strings.Repeat("─", width))
}

func Frame(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder

	borderStyle := StyleBar
	topLeft := borderStyle.Render("╭")
	topRight := borderStyle.Render("╮")
	bottomLeft := borderStyle.Render("╰")
	bottomRight := borderStyle.Render("╯")
	vertical := borderStyle.Render("│")

	contentW := width - 2

	for i := 0; i < height; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}

		lineW := lipgloss.Width(line)
		padding := contentW - lineW
		if padding < 0 {
			padding = 0
			line = TruncateVisual(line, contentW)
		}

		if i == 0 {
			b.WriteString(topLeft)
			b.WriteString(line)
			b.WriteString(strings.Repeat(" ", padding))
			b.WriteString(topRight)
		} else if i == height-1 {
			b.WriteString(bottomLeft)
			b.WriteString(line)
			b.WriteString(strings.Repeat(" ", padding))
			b.WriteString(bottomRight)
		} else {
			b.WriteString(vertical)
			b.WriteString(line)
			b.WriteString(strings.Repeat(" ", padding))
			b.WriteString(vertical)
		}

		if i < height-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func TruncateVisual(s string, maxW int) string {
	if lipgloss.Width(s) <= maxW {
		return s
	}
	for len(s) > 0 && lipgloss.Width(s) > maxW-3 {
		s = s[:len(s)-1]
	}
	return s + "..."
}

func Truncate(s string, maxLen int) string {
	if maxLen <= 3 {
		if len(s) <= maxLen {
			return s
		}
		return s[:maxLen]
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func PadLeft(s string, width int) string {
	w := lipgloss.Width(s)
	if w < width {
		s = strings.Repeat(" ", width-w) + s
	}
	return s
}

func PadRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w < width {
		s = s + strings.Repeat(" ", width-w)
	}
	return s
}

func PadCenter(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return TruncateVisual(s, width)
	}
	pad := width - w
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func CenterText(text string, width int, style lipgloss.Style) string {
	textW := lipgloss.Width(text)
	if textW >= width {
		return style.Render(TruncateVisual(text, width))
	}
	pad := (width - textW) / 2
	return strings.Repeat(" ", pad) + style.Render(text) + strings.Repeat(" ", width-textW-pad)
}

func WordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}

	var result strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if len(line) <= width {
			result.WriteString(line + "\n")
			continue
		}

		for len(line) > width {
			breakAt := width
			for i := width; i > width/2; i-- {
				if i < len(line) && line[i] == ' ' {
					breakAt = i
					break
				}
			}
			if breakAt > len(line) {
				breakAt = len(line)
			}
			result.WriteString(line[:breakAt] + "\n")
			line = strings.TrimLeft(line[breakAt:], " ")
		}
		if len(line) > 0 {
			result.WriteString(line + "\n")
		}
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func Itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + Itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
