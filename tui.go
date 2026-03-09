package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("30")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("36"))

	normalStyle = lipgloss.NewStyle()

	runningDot  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("●")
	stoppedDot  = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("○")
	failedDot   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("●")
	stoppingDot = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("◍")

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("30"))

	logLineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

type tickMsg time.Time

type model struct {
	services  []*Service
	cursor    int
	width     int
	height    int
	logScroll int
}

func newModel(services []*Service) model {
	return model{
		services: services,
		width:    80,
		height:   24,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "Q":
			for _, s := range m.services {
				s.Stop()
			}
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.logScroll = 0
			}
		case "down", "j":
			if m.cursor < len(m.services)-1 {
				m.cursor++
				m.logScroll = 0
			}

		case "enter":
			s := m.services[m.cursor]
			status := s.GetStatus()
			if status == "running" {
				go s.Stop()
			} else if status == "stopped" {
				go s.Start()
			}
			// "stopping" → queued: opMu will serialize, Start runs after Stop completes

		case "a":
			for _, s := range m.services {
				go s.Start()
			}

		case "s":
			for _, s := range m.services {
				go s.Stop()
			}

		case "r":
			s := m.services[m.cursor]
			go func() {
				s.Stop()
				s.Start()
			}()

		case "G":
			m.logScroll = 0

		case "g":
			if len(m.services) > 0 {
				logs := m.services[m.cursor].GetLogs()
				logAreaHeight := m.height - 4
				if len(logs) > logAreaHeight {
					m.logScroll = len(logs) - logAreaHeight
				}
			}
		}

	case tickMsg:
		for _, s := range m.services {
			s.RefreshStatus()
			s.RefreshProcs()
		}
		return m, tickCmd()
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	leftWidth := 30
	rightWidth := m.width - leftWidth - 5
	if rightWidth < 20 {
		rightWidth = 20
	}
	contentHeight := m.height - 4

	// Left panel: service list
	left := m.renderServiceList(leftWidth, contentHeight)

	// Right panel: logs
	right := m.renderLogs(rightWidth, contentHeight)

	// Join panels
	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	// Help bar
	help := helpStyle.Render("  ↑↓/jk select  enter toggle  a start all  s stop all  r restart  q quit  Q quit+stop all")

	return panels + "\n" + help
}

func (m model) renderServiceList(width, height int) string {
	title := titleStyle.Render(" Services ")

	var items []string
	for i, s := range m.services {
		dotChar := "○"
		status := s.GetStatus()
		switch status {
		case "running":
			dotChar = "●"
		case "failed":
			dotChar = "●"
		case "stopping":
			dotChar = "◍"
		}

		label := fmt.Sprintf(" %s %s", dotChar, s.Config.Name)
		statusTag := fmt.Sprintf("[%s]", status)

		// Pad to width
		padLen := width - 4 - lipgloss.Width(label) - lipgloss.Width(statusTag)
		if padLen < 0 {
			padLen = 0
		}
		line := label + strings.Repeat(" ", padLen) + statusTag

		if i == m.cursor {
			line = selectedStyle.Width(width - 2).Render(line)
		} else {
			// Apply dot color only for non-selected rows
			var dot string
			switch status {
			case "running":
				dot = runningDot
			case "failed":
				dot = failedDot
			case "stopping":
				dot = stoppingDot
			default:
				dot = stoppedDot
			}
			coloredLabel := fmt.Sprintf(" %s %s", dot, s.Config.Name)
			line = coloredLabel + strings.Repeat(" ", padLen) + statusTag
			line = normalStyle.Width(width - 2).Render(line)
		}
		items = append(items, line)
	}

	// Pad remaining height
	for len(items) < height {
		items = append(items, strings.Repeat(" ", width-2))
	}

	content := title + "\n" + strings.Join(items[:height], "\n")
	return borderStyle.Width(width).Render(content)
}

func (m model) renderLogs(width, height int) string {
	logTitle := "Logs"
	if len(m.services) > 0 {
		logTitle = fmt.Sprintf("Logs (%s)", m.services[m.cursor].Config.Name)
	}
	title := titleStyle.Render(fmt.Sprintf(" %s ", logTitle))

	// Process tree info
	var procLine string
	if len(m.services) > 0 {
		procs := m.services[m.cursor].GetProcs()
		if len(procs) > 0 {
			var parts []string
			for _, p := range procs {
				entry := fmt.Sprintf("%d(%s)", p.PID, p.Comm)
				if len(p.Ports) > 0 {
					entry += " :" + strings.Join(p.Ports, ",:")
				}
				parts = append(parts, entry)
			}
			procLine = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
				"  procs: " + strings.Join(parts, "  "))
			height-- // reserve a line for process info
		}
	}

	var lines []string
	if len(m.services) > 0 {
		logs := m.services[m.cursor].GetLogs()

		// Calculate visible window
		visibleLines := height
		start := len(logs) - visibleLines - m.logScroll
		if start < 0 {
			start = 0
		}
		end := start + visibleLines
		if end > len(logs) {
			end = len(logs)
		}

		for _, l := range logs[start:end] {
			// Truncate long lines
			if lipgloss.Width(l) > width-4 {
				l = l[:width-4]
			}
			lines = append(lines, logLineStyle.Render(l))
		}
	}

	// Pad remaining height
	for len(lines) < height {
		lines = append(lines, "")
	}

	content := title + "\n"
	if procLine != "" {
		content += procLine + "\n"
	}
	content += strings.Join(lines[:height], "\n")
	return borderStyle.Width(width).Render(content)
}
