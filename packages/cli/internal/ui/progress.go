package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TestProgressModel struct {
	scenarios []ScenarioProgress
	parallel  int
	width     int
	height    int
	startTime time.Time
}

type ScenarioProgress struct {
	Name     string
	Status   string
	Progress float64
	Duration time.Duration
	Cost     float64
}

func NewTestProgressModel(scenarios []string, parallel int) *TestProgressModel {
	progress := make([]ScenarioProgress, len(scenarios))
	for i, scenario := range scenarios {
		progress[i] = ScenarioProgress{
			Name:     scenario,
			Status:   "pending",
			Progress: 0.0,
		}
	}

	return &TestProgressModel{
		scenarios: progress,
		parallel:  parallel,
		startTime: time.Now(),
	}
}

func (m *TestProgressModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

func (m *TestProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tickMsg:
		return m, tickCmd()
	}

	return m, nil
}

func (m *TestProgressModel) View() string {
	var builder strings.Builder

	header := FormatHeader(fmt.Sprintf(" Running Scenarios (%s) ", formatDuration(time.Since(m.startTime))))
	builder.WriteString(header)
	builder.WriteString("\n\n")

	completed := 0
	running := 0
	failed := 0

	for _, scenario := range m.scenarios {
		icon := "⏸"
		style := lipgloss.NewStyle()

		switch scenario.Status {
		case "running":
			icon = "⏳"
			style = infoStyle
			running++
		case "passed":
			icon = "✓"
			style = successStyle
			completed++
		case "failed":
			icon = "✗"
			style = errorStyle
			completed++
			failed++
		case "pending":
			icon = "⏸"
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		}

		name := TruncateString(scenario.Name, 50)
		durationStr := formatDuration(scenario.Duration)
		costStr := fmt.Sprintf("$%.4f", scenario.Cost)

		line := fmt.Sprintf("%s %-50s %8s  %8s",
			style.Render(icon),
			name,
			durationStr,
			costStr,
		)

		builder.WriteString(line)
		builder.WriteString("\n")

		if scenario.Status == "running" && scenario.Progress > 0 {
			progressBar := FormatProgressBar(int(scenario.Progress*100), 100, 40)
			builder.WriteString("    " + progressBar)
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n")

	totalCost := 0.0
	for _, scenario := range m.scenarios {
		totalCost += scenario.Cost
	}

	summary := fmt.Sprintf("Progress: %d/%d | Running: %d | Failed: %d | Cost: $%.4f",
		completed, len(m.scenarios), running, failed, totalCost)
	builder.WriteString(FormatInfo(summary))
	builder.WriteString("\n\n")

	statusBar := FormatStatusBar("[Space] Pause  [Ctrl+C] Stop", "", m.width)
	builder.WriteString(statusBar)

	return builder.String()
}

func (m *TestProgressModel) UpdateProgress(scenario string, status string, progress float64) {
	for i := range m.scenarios {
		if m.scenarios[i].Name == scenario {
			m.scenarios[i].Status = status
			m.scenarios[i].Progress = progress
			if status == "passed" || status == "failed" {
				m.scenarios[i].Duration = time.Since(m.startTime)
			}
			break
		}
	}
}

func RunTestUI(model *TestProgressModel) error {
	return RunUI(model)
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}