package ui

import (

	tea "github.com/charmbracelet/bubbletea"
)

type ReplayModel struct {
	recording   interface{}
	currentIdx  int
	paused      bool
	speed       float64
	stepMode    bool
	breakpoint  string
	width       int
	height      int
}

func NewReplayModel(recording interface{}, speed float64, stepMode bool, breakpoint string) *ReplayModel {
	return &ReplayModel{
		recording:  recording,
		currentIdx: 0,
		paused:     stepMode,
		speed:      speed,
		stepMode:   stepMode,
		breakpoint: breakpoint,
	}
}

func (m *ReplayModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

func (m *ReplayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ":
			m.paused = !m.paused
		case "left":
			if m.currentIdx > 0 {
				m.currentIdx--
			}
		case "right":
			m.currentIdx++
		case "b":
			return m, nil
		}

	case tickMsg:
		if !m.paused {
			m.currentIdx++
		}
		return m, tickCmd()
	}

	return m, nil
}

func (m *ReplayModel) View() string {
	header := FormatHeader(" Replay Debug Session ")
	
	content := "\n\n"
	content += FormatInfo("Timeline: Event " + string(rune(m.currentIdx)) + "\n")
	content += "\n"
	content += "Event details will appear here\n"
	content += "\n\n"
	
	statusBar := FormatStatusBar("[←/→] Navigate  [Space] Play/Pause  [Q] Quit", "", m.width)
	
	return header + content + statusBar
}

func RunReplayUI(model *ReplayModel) error {
	return RunUI(model)
}

type ComparisonModel struct {
	recording1 interface{}
	recording2 interface{}
	width      int
	height     int
}

func NewComparisonModel(recording1, recording2 interface{}) *ComparisonModel {
	return &ComparisonModel{
		recording1: recording1,
		recording2: recording2,
	}
}

func (m *ComparisonModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m *ComparisonModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	}

	return m, nil
}

func (m *ComparisonModel) View() string {
	header := FormatHeader(" Run Comparison ")
	
	content := "\n\n"
	content += "Comparison view\n"
	content += "\n"
	
	statusBar := FormatStatusBar("[Q] Quit", "", m.width)
	
	return header + content + statusBar
}

func RunComparisonUI(model *ComparisonModel) error {
	return RunUI(model)
}