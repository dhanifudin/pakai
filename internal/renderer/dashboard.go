package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dhanifudin/pakai/internal/client"
	"github.com/dhanifudin/pakai/internal/schema"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("240"))

	providerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	okStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("46"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226"))

	criticalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	barFilledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	barEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))
)

// dashboardModel is the Bubbletea model for the dashboard.
type dashboardModel struct {
	usages      []*schema.Usage
	daemonInfo  *client.HealthResponse
	refreshedAt time.Time
	debug       bool
	err         string
	port        int
	width       int
	height      int
}

// usageUpdateMsg is sent when new usage data arrives.
type usageUpdateMsg struct {
	usages []*schema.Usage
}

// healthUpdateMsg is sent when health data arrives.
type healthUpdateMsg struct {
	health *client.HealthResponse
}

// errMsg is sent when an error occurs.
type errMsg struct {
	err string
}

// tickMsg is sent periodically for clock updates.
type tickMsg time.Time

// NewDashboardModel creates a new dashboard model.
func NewDashboardModel(port int) dashboardModel {
	return dashboardModel{
		port: port,
	}
}

func (m dashboardModel) Init() tea.Cmd {
	return tea.Batch(
		fetchHealth(m.port),
		subscribeSSE(m.port),
		tick(),
	)
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, fetchHealth(m.port)
		case "d":
			m.debug = !m.debug
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case usageUpdateMsg:
		m.usages = msg.usages
		m.refreshedAt = time.Now()
		m.err = ""

	case healthUpdateMsg:
		m.daemonInfo = msg.health

	case errMsg:
		m.err = msg.err

	case tickMsg:
		return m, tick()
	}

	return m, nil
}

func (m dashboardModel) View() string {
	var sb strings.Builder

	// Title
	sb.WriteString(titleStyle.Render("pakai — AI Usage Tracker"))
	sb.WriteString("\n")

	// Daemon info
	if m.daemonInfo != nil {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("Daemon uptime: %ds", m.daemonInfo.UptimeSeconds)))
	} else {
		sb.WriteString(dimStyle.Render("Daemon: connecting..."))
	}
	sb.WriteString("\n\n")

	if m.err != "" {
		sb.WriteString(errorStyle.Render("Error: " + m.err))
		sb.WriteString("\n\n")
	}

	if len(m.usages) == 0 {
		sb.WriteString(dimStyle.Render("No data yet..."))
		sb.WriteString("\n")
	} else {
		for _, u := range m.usages {
			sb.WriteString(renderDashboardRow(u))
			sb.WriteString("\n")
		}

		if !m.refreshedAt.IsZero() {
			ago := time.Since(m.refreshedAt).Round(time.Second)
			sb.WriteString("\n")
			sb.WriteString(dimStyle.Render(fmt.Sprintf("Refreshed %s ago", ago)))
			sb.WriteString("\n")
		}
	}

	if m.debug && len(m.usages) > 0 {
		sb.WriteString("\n")
		sb.WriteString(headerStyle.Render("Debug (raw JSON):"))
		sb.WriteString("\n")
		b, _ := json.MarshalIndent(m.usages, "", "  ")
		sb.WriteString(dimStyle.Render(string(b)))
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("q: quit  r: refresh  d: toggle debug"))

	return sb.String()
}

func renderDashboardRow(u *schema.Usage) string {
	label := u.Label
	if label == "" {
		label = u.Provider
	}

	if u.Status == schema.StatusError {
		return fmt.Sprintf("  %-20s %s",
			providerStyle.Render(label),
			errorStyle.Render("error: "+u.Error))
	}

	if len(u.WindowsOrDefault()) > 1 {
		return fmt.Sprintf("  %-20s %s",
			providerStyle.Render(label),
			dimStyle.Render(strings.Join(func() []string {
				parts := make([]string, 0, len(u.WindowsOrDefault()))
				for _, w := range u.WindowsOrDefault() {
					parts = append(parts, renderWindowCompact(w))
				}
				return parts
			}(), "  ")))
	}

	pct := u.Pct()
	usedStr := u.FormatUsed()

	var statusStr string
	if pct < 0 {
		statusStr = fmt.Sprintf("%-12s %s", usedStr, dimStyle.Render("(no limit)"))
	} else {
		bar := coloredProgressBar(pct, 12)
		pctStr := fmt.Sprintf("%3.0f%%", pct)
		var pctStyled string
		switch {
		case pct >= 95:
			pctStyled = criticalStyle.Render(pctStr)
		case pct >= 80:
			pctStyled = criticalStyle.Render(pctStr)
		case pct >= 50:
			pctStyled = warningStyle.Render(pctStr)
		default:
			pctStyled = okStyle.Render(pctStr)
		}
		statusStr = fmt.Sprintf("%s %s %s", bar, pctStyled, dimStyle.Render(usedStr))
	}

	var mockTag string
	if u.Status == schema.StatusMock {
		mockTag = dimStyle.Render(" [mock]")
	}

	return fmt.Sprintf("  %-20s %s%s",
		providerStyle.Render(label),
		statusStr,
		mockTag)
}

func coloredProgressBar(pct float64, width int) string {
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	var fillStyle lipgloss.Style
	switch {
	case pct >= 80:
		fillStyle = criticalStyle
	case pct >= 50:
		fillStyle = warningStyle
	default:
		fillStyle = okStyle
	}

	return fillStyle.Render(strings.Repeat("█", filled)) +
		barEmptyStyle.Render(strings.Repeat("░", empty))
}

// Commands

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchHealth(port int) tea.Cmd {
	return func() tea.Msg {
		c := client.New(port)
		h, err := c.GetHealth()
		if err != nil {
			return errMsg{err: err.Error()}
		}
		return healthUpdateMsg{health: h}
	}
}

func subscribeSSE(port int) tea.Cmd {
	return func() tea.Msg {
		c := client.New(port)
		ctx := context.Background()

		var firstUsages []*schema.Usage
		err := c.Events(ctx, func(usages []*schema.Usage) {
			firstUsages = usages
			// We only return the first batch; subsequent updates handled by re-subscribing
		})

		if err != nil {
			return errMsg{err: "SSE: " + err.Error()}
		}

		if firstUsages != nil {
			return usageUpdateMsg{usages: firstUsages}
		}
		return errMsg{err: "SSE stream ended"}
	}
}

// RunDashboard runs the Bubbletea dashboard.
func RunDashboard(port int) error {
	m := NewDashboardModel(port)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Subscribe to SSE in a goroutine and send updates to the program
	go func() {
		c := client.New(port)
		ctx := context.Background()

		for {
			err := c.Events(ctx, func(usages []*schema.Usage) {
				p.Send(usageUpdateMsg{usages: usages})
			})
			if err != nil {
				p.Send(errMsg{err: "SSE disconnected: " + err.Error()})
			}
			// Reconnect after a brief pause
			time.Sleep(2 * time.Second)
		}
	}()

	// Periodically refresh health
	go func() {
		for {
			time.Sleep(10 * time.Second)
			c := client.New(port)
			h, err := c.GetHealth()
			if err == nil {
				p.Send(healthUpdateMsg{health: h})
			}
		}
	}()

	_, err := p.Run()
	return err
}
