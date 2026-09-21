package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"go-fast-cli/pkg/fast"
)

// Styles used across the TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#E50914")).
			Padding(0, 1).
			MarginLeft(2)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#E50914")).
			Padding(1, 3).
			MarginLeft(2)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Width(18)

	valueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EDEDED"))

	speedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00E676"))

	speedLargeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00E676"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#757575"))

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5252"))
)

// Model represents the Bubble Tea state for the speed test TUI.
type Model struct {
	ctx        context.Context
	cancel     context.CancelFunc
	tester     *fast.Tester
	progressCh <-chan fast.Progress

	spinner  spinner.Model
	progress progress.Model

	lastProgress fast.Progress
	err          error
	quitting     bool
	finished     bool
}

type progressMsg fast.Progress

// NewModel creates a new TUI model.
func NewModel(ctx context.Context, cancel context.CancelFunc, tester *fast.Tester) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#E50914"))

	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(48),
	)

	return Model{
		ctx:        ctx,
		cancel:     cancel,
		tester:     tester,
		progressCh: tester.Run(ctx),
		spinner:    s,
		progress:   prog,
	}
}

// Init starts the speed test runner and begins listening for progress updates.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForProgressCmd(m.progressCh),
	)
}

// waitForProgressCmd returns a command that waits for the next update from the channel.
func waitForProgressCmd(ch <-chan fast.Progress) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		p, ok := <-ch
		if !ok {
			return nil
		}
		return progressMsg(p)
	}
}

// Update handles incoming messages and updates state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progressMsg:
		m.lastProgress = fast.Progress(msg)
		if m.lastProgress.Err != nil {
			m.err = m.lastProgress.Err
			m.finished = true
			return m, tea.Quit
		}

		if m.lastProgress.Phase == fast.PhaseCompleted {
			m.finished = true
			return m, tea.Quit
		}

		return m, waitForProgressCmd(m.progressCh)
	}

	return m, nil
}

// View renders the TUI screen.
func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("\n  ✖ Error: %v\n\n", m.err))
	}
	if m.quitting {
		return mutedStyle.Render("\n  Test aborted by user.\n\n")
	}

	var b strings.Builder

	b.WriteString("\n ")
	b.WriteString(titleStyle.Render("FAST.COM SPEED TEST"))
	b.WriteString("\n\n")

	p := m.lastProgress

	switch p.Phase {
	case fast.PhaseInit, fast.PhaseConnecting:
		b.WriteString(fmt.Sprintf("  %s Connecting to Fast.com servers...\n", m.spinner.View()))
		if p.Client != nil {
			b.WriteString(fmt.Sprintf("    %s %s (%s)\n",
				labelStyle.Render("Client ISP:"),
				valueStyle.Render(p.Client.ISP),
				p.Client.IP))
		}
		b.WriteString("\n" + mutedStyle.Render("  Press 'q' to abort.") + "\n")

	case fast.PhaseTesting:
		speedStr := fast.FormatSpeed(p.InstantSpeedMbps)
		b.WriteString(fmt.Sprintf("  %s %s\n", m.spinner.View(), speedLargeStyle.Render(speedStr)))
		b.WriteString("\n")

		// Progress bar
		b.WriteString("  " + m.progress.ViewAs(p.Percent) + "\n\n")

		// Stats
		b.WriteString(fmt.Sprintf("  %s %s\n",
			labelStyle.Render("Transferred:"),
			valueStyle.Render(fmt.Sprintf("%s (avg %s)",
				fast.FormatBytes(p.BytesTransferred),
				fast.FormatSpeed(p.AverageSpeedMbps)))))

		if p.Latency > 0 {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				labelStyle.Render("Latency:"),
				valueStyle.Render(fmt.Sprintf("%d ms", p.Latency.Milliseconds()))))
		}

		if p.Client != nil {
			clientLoc := fmt.Sprintf("%s, %s", p.Client.Location.City, p.Client.Location.Country)
			b.WriteString(fmt.Sprintf("  %s %s (%s)\n",
				labelStyle.Render("Client:"),
				valueStyle.Render(p.Client.ISP),
				mutedStyle.Render(clientLoc)))
		}

		if len(p.Targets) > 0 {
			targetLoc := fmt.Sprintf("%s, %s", p.Targets[0].Location.City, p.Targets[0].Location.Country)
			b.WriteString(fmt.Sprintf("  %s %s\n",
				labelStyle.Render("Server Node:"),
				valueStyle.Render(targetLoc)))
		}

		rem := p.TotalDuration - p.Elapsed
		if rem < 0 {
			rem = 0
		}
		b.WriteString(fmt.Sprintf("  %s %s remaining\n",
			labelStyle.Render("Time:"),
			mutedStyle.Render(fmt.Sprintf("%.1fs / %.1fs", p.Elapsed.Seconds(), p.TotalDuration.Seconds()))))

		b.WriteString("\n" + mutedStyle.Render("  Press 'q' or Ctrl+C to abort.") + "\n")

	case fast.PhaseCompleted:
		var card strings.Builder

		card.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render("SPEED TEST RESULTS") + "\n\n")

		speedFinal := speedStyle.Render(fast.FormatSpeed(p.AverageSpeedMbps))
		card.WriteString(fmt.Sprintf("%s %s\n\n", labelStyle.Render("Download Speed:"), speedFinal))

		if p.Latency > 0 {
			card.WriteString(fmt.Sprintf("%s %s\n",
				labelStyle.Render("Latency:"),
				valueStyle.Render(fmt.Sprintf("%d ms", p.Latency.Milliseconds()))))
		}

		card.WriteString(fmt.Sprintf("%s %s\n",
			labelStyle.Render("Data Transferred:"),
			valueStyle.Render(fast.FormatBytes(p.BytesTransferred))))

		card.WriteString(fmt.Sprintf("%s %s\n\n",
			labelStyle.Render("Total Time:"),
			valueStyle.Render(fmt.Sprintf("%.1fs", p.Elapsed.Seconds()))))

		if p.Client != nil {
			card.WriteString(fmt.Sprintf("%s %s\n",
				labelStyle.Render("Provider:"),
				valueStyle.Render(p.Client.ISP)))
			card.WriteString(fmt.Sprintf("%s %s\n",
				labelStyle.Render("Client IP:"),
				valueStyle.Render(fmt.Sprintf("%s (%s, %s)",
					p.Client.IP,
					p.Client.Location.City,
					p.Client.Location.Country))))
		}

		if len(p.Targets) > 0 {
			card.WriteString(fmt.Sprintf("%s %s, %s\n",
				labelStyle.Render("CDN Target:"),
				valueStyle.Render(p.Targets[0].Location.City),
				p.Targets[0].Location.Country))
		}

		b.WriteString(cardStyle.Render(card.String()) + "\n\n")

	default:
		b.WriteString(fmt.Sprintf("  %s Preparing test...\n\n", m.spinner.View()))
	}

	return b.String()
}
