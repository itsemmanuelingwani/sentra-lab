package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Table struct {
	Headers []string
	Rows    [][]string
	Width   int
}

func NewTable(headers []string) *Table {
	return &Table{
		Headers: headers,
		Rows:    [][]string{},
		Width:   80,
	}
}

func (t *Table) AddRow(row []string) {
	t.Rows = append(t.Rows, row)
}

func (t *Table) Render() string {
	if len(t.Rows) == 0 {
		return infoStyle.Render("No data to display")
	}

	colWidths := t.calculateColumnWidths()

	var builder strings.Builder

	builder.WriteString(t.renderHeader(colWidths))
	builder.WriteString("\n")
	builder.WriteString(t.renderSeparator(colWidths))
	builder.WriteString("\n")

	for _, row := range t.Rows {
		builder.WriteString(t.renderRow(row, colWidths))
		builder.WriteString("\n")
	}

	return builder.String()
}

func (t *Table) calculateColumnWidths() []int {
	widths := make([]int, len(t.Headers))

	for i, header := range t.Headers {
		widths[i] = lipgloss.Width(header)
	}

	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) {
				cellWidth := lipgloss.Width(cell)
				if cellWidth > widths[i] {
					widths[i] = cellWidth
				}
			}
		}
	}

	return widths
}

func (t *Table) renderHeader(colWidths []int) string {
	var parts []string

	for i, header := range t.Headers {
		cell := headerStyle.Render(padRight(header, colWidths[i]))
		parts = append(parts, cell)
	}

	return strings.Join(parts, " ")
}

func (t *Table) renderSeparator(colWidths []int) string {
	var parts []string

	for _, width := range colWidths {
		parts = append(parts, strings.Repeat("─", width+2))
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Render(strings.Join(parts, "┼"))
}

func (t *Table) renderRow(row []string, colWidths []int) string {
	var parts []string

	for i, cell := range row {
		if i < len(colWidths) {
			paddedCell := padRight(cell, colWidths[i])
			parts = append(parts, paddedCell)
		}
	}

	return strings.Join(parts, " │ ")
}

type SpinnerModel struct {
	frames   []string
	frame    int
	message  string
	done     bool
}

func NewSpinner(message string) *SpinnerModel {
	return &SpinnerModel{
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		frame:   0,
		message: message,
		done:    false,
	}
}

func (s *SpinnerModel) Tick() {
	s.frame = (s.frame + 1) % len(s.frames)
}

func (s *SpinnerModel) SetDone() {
	s.done = true
}

func (s *SpinnerModel) Render() string {
	if s.done {
		return successStyle.Render("✓ " + s.message)
	}

	spinner := infoStyle.Render(s.frames[s.frame])
	return fmt.Sprintf("%s %s", spinner, s.message)
}