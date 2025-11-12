package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			PaddingLeft(2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00BFFF"))

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)
)

type Model interface {
	tea.Model
}

func RunUI(model Model) error {
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func FormatTitle(text string) string {
	return titleStyle.Render(text)
}

func FormatHeader(text string) string {
	return headerStyle.Render(text)
}

func FormatSuccess(text string) string {
	return successStyle.Render("✓ " + text)
}

func FormatError(text string) string {
	return errorStyle.Render("✗ " + text)
}

func FormatInfo(text string) string {
	return infoStyle.Render("ℹ " + text)
}

func FormatBorder(content string) string {
	return borderStyle.Render(content)
}

func FormatStatusBar(left, right string, width int) string {
	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}

	return statusBarStyle.Render(left + strings.Repeat(" ", padding) + right)
}

func FormatProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	percentage := float64(current) / float64(total)
	filledWidth := int(float64(width) * percentage)
	emptyWidth := width - filledWidth

	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", emptyWidth)

	progressText := fmt.Sprintf("%d/%d (%.0f%%)", current, total, percentage*100)

	bar := successStyle.Render(filled) + lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(empty)

	return fmt.Sprintf("%s %s", bar, progressText)
}

func FormatList(items []string, selectedIndex int) string {
	var builder strings.Builder

	for i, item := range items {
		if i == selectedIndex {
			builder.WriteString(successStyle.Render("▶ " + item))
		} else {
			builder.WriteString("  " + item)
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func FormatTable(headers []string, rows [][]string) string {
	if len(rows) == 0 {
		return "No data"
	}

	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	var builder strings.Builder

	for i, header := range headers {
		builder.WriteString(headerStyle.Render(padRight(header, colWidths[i])))
		builder.WriteString(" ")
	}
	builder.WriteString("\n")

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				builder.WriteString(padRight(cell, colWidths[i]))
				builder.WriteString(" ")
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func padRight(str string, width int) string {
	if len(str) >= width {
		return str
	}
	return str + strings.Repeat(" ", width-len(str))
}

func TruncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	if maxLen <= 3 {
		return str[:maxLen]
	}
	return str[:maxLen-3] + "..."
}