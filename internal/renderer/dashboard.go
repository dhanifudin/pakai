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

// Styles — using terminal-adaptive colors that auto-switch between light/dark themes.
// Based on Catppuccin Latte (light) and Catppuccin Mocha (dark) palettes.
var (
	accentColor = lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	textColor   = lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"}
	greenColor  = lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
	yellowColor = lipgloss.AdaptiveColor{Light: "#df8e1d", Dark: "#f9e2af"}
	redColor    = lipgloss.AdaptiveColor{Light: "#d20f39", Dark: "#f38ba8"}
	dimColor    = lipgloss.AdaptiveColor{Light: "#9ca0b0", Dark: "#585b70"}
	subtleColor = lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#6c7086"}
	emptyColor  = lipgloss.AdaptiveColor{Light: "#e6e9ef", Dark: "#313244"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(dimColor)

	providerStyle = lipgloss.NewStyle().
			Foreground(textColor)

	okStyle = lipgloss.NewStyle().
		Foreground(greenColor)

	warningStyle = lipgloss.NewStyle().
			Foreground(yellowColor)

	criticalStyle = lipgloss.NewStyle().
			Foreground(redColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(redColor).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	subtleStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	barFilledStyle = lipgloss.NewStyle().
			Foreground(greenColor)

	barEmptyStyle = lipgloss.NewStyle().
			Foreground(emptyColor)

	warningNoteStyle = lipgloss.NewStyle().
				Foreground(yellowColor)
)

var (
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	cardHeaderStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Bold(true)

	windowLabelStyle = lipgloss.NewStyle().
				Foreground(dimColor).
				Width(3)

	reserveStyle = lipgloss.NewStyle().
			Foreground(dimColor)
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
	sb.WriteString(titleStyle.Render("PakAI — AI Usage Tracker"))
	sb.WriteString("\n")

	providerCount := len(m.usages)
	refreshedLine := "waiting for first refresh"
	if !m.refreshedAt.IsZero() {
		refreshedLine = fmt.Sprintf("refreshed %s ago", time.Since(m.refreshedAt).Round(time.Second))
	}

	// Daemon info
	if m.daemonInfo != nil {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("Daemon %ds  |  %d providers  |  %s", m.daemonInfo.UptimeSeconds, providerCount, refreshedLine)))
	} else {
		sb.WriteString(dimStyle.Render("Daemon connecting..."))
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
		standalone, subs := partitionOpencode(m.usages)
		for _, u := range standalone {
			sb.WriteString(renderProviderCard(u))
			sb.WriteString("\n")
		}
		if len(subs) > 0 {
			sb.WriteString(renderOpencodeDashboardSection(subs))
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

func renderProviderCard(u *schema.Usage) string {
	var body strings.Builder

	label := u.Label
	if label == "" {
		label = u.Provider
	}
	header := label
	if icon := providerIcon(u.Provider); icon != "" {
		header = icon + " " + label
	}

	// Error state: simplified card
	if u.Status == schema.StatusError {
		body.WriteString(cardHeaderStyle.Render(header))
		body.WriteString("\n")
		body.WriteString(errorStyle.Render("error: " + u.Error))
		wrapped := cardStyle.Render(body.String())
		return "  " + wrapped
	}

	body.WriteString(cardHeaderStyle.Render(header))
	body.WriteString("\n")

	// Warning / mock notes
	if u.Warning != "" {
		body.WriteString(warningNoteStyle.Render("  note: " + u.Warning))
		body.WriteString("\n")
	}
	if u.Status == schema.StatusMock {
		body.WriteString(dimStyle.Render("  mock data"))
		body.WriteString("\n")
	}

	for _, w := range u.WindowsOrDefault() {
		if w.Pct() <= 0 && w.Used == 0 {
			continue
		}
		body.WriteString(renderDashboardWindowRow(w))
	}

	out := strings.TrimRight(body.String(), "\n")
	wrapped := cardStyle.Render(out)
	return "  " + wrapped
}

// renderOpencodeDashboardSection renders an opencode group header with each
// sub-provider nested underneath.
func renderOpencodeDashboardSection(subs []*schema.Usage) string {
	var body strings.Builder
	header := providerIcon("opencode") + " opencode"
	body.WriteString(cardHeaderStyle.Render(header))
	body.WriteString("\n")
	for _, u := range subs {
		body.WriteString(renderDashboardRowIndented(u, subLabel(u)))
	}
	out := strings.TrimRight(body.String(), "\n")
	wrapped := cardStyle.Render(out)
	return "  " + wrapped
}

// renderDashboardRowIndented is like renderDashboardRow but uses an explicit
// label and indentation prefix, for rendering opencode sub-providers.
func renderDashboardRowIndented(u *schema.Usage, label string) string {
	var body strings.Builder
	icon := providerIcon(u.Provider)
	header := label
	if icon != "" {
		header = icon + " " + label
	}

	if u.Status == schema.StatusError {
		body.WriteString(cardHeaderStyle.Render(header))
		body.WriteString("\n")
		body.WriteString(errorStyle.Render("error: " + u.Error))
		return strings.TrimRight(body.String(), "\n")
	}

	body.WriteString(cardHeaderStyle.Render(header))
	body.WriteString("\n")

	if u.Warning != "" {
		body.WriteString(warningNoteStyle.Render("  note: " + u.Warning))
		body.WriteString("\n")
	}
	if u.Status == schema.StatusMock {
		body.WriteString(dimStyle.Render("  mock data"))
		body.WriteString("\n")
	}

	for _, w := range u.WindowsOrDefault() {
		if w.Pct() <= 0 && w.Used == 0 {
			continue
		}
		body.WriteString(renderDashboardWindowRow(w))
	}

	return strings.TrimRight(body.String(), "\n")
}

// renderDashboardWindowRowIndented is like renderDashboardWindowRow but with
// a custom indentation prefix.
func renderDashboardWindowRow(w schema.UsageWindow) string {
	wname := shortWindowLabel(w)
	pct := w.Pct()
	leftPct := w.PctLeft()
	pctStr := fmt.Sprintf("%.0f%% left", leftPct)
	var pctStyled string
	switch {
	case leftPct >= 80:
		pctStyled = okStyle.Render(pctStr)
	case leftPct >= 50:
		pctStyled = warningStyle.Render(pctStr)
	default:
		pctStyled = criticalStyle.Render(pctStr)
	}

	if pct < 0 {
		// No limit configured: just show usage and reset
		line := fmt.Sprintf("%s  %s", wname, subtleStyle.Render(w.FormatUsed()))
		if reset := formatResetAt(w.ResetAt); reset != "" {
			line += "  " + dimStyle.Render(reset)
		}
		return "  " + line + "\n"
	}

	bar := coloredProgressBar(pct, 12, leftPct)
	line1 := fmt.Sprintf("%s %s %s", wname, bar, pctStyled)

	// Reserve status + reset timer on line 2
	var metaParts []string
	if reserve := renderReserve(w); reserve != "" {
		metaParts = append(metaParts, reserve)
	}
	if reset := formatResetAt(w.ResetAt); reset != "" {
		metaParts = append(metaParts, reset)
	}
	line2 := ""
	if len(metaParts) > 0 {
		// Indent to align with the label width + bar gap
		indent := strings.Repeat(" ", len(wname)+1)
		line2 = indent + dimStyle.Render(strings.Join(metaParts, "  ·  "))
	}

	return "  " + line1 + "\n" + line2 + "\n"
}

func coloredProgressBar(pct float64, width int, leftPct float64) string {
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
	case leftPct >= 80:
		fillStyle = okStyle
	case leftPct >= 50:
		fillStyle = warningStyle
	default:
		fillStyle = criticalStyle
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
