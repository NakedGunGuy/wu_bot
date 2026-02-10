package tui

import (
	"fmt"
	"strings"
	"time"

	"wu_bot_go/internal/bot"
	"wu_bot_go/internal/manager"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type view int

const (
	viewBotList view = iota
	viewBotDetail
	viewHelp
)

type tickMsg time.Time

type model struct {
	mgr      *manager.BotManager
	cfgPath  string
	view     view
	cursor   int
	width    int
	height   int
	logs     map[string][]string // per-username log ring buffer
	logScroll int
	detailBot string
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func initialModel(mgr *manager.BotManager, cfgPath string) model {
	return model{
		mgr:     mgr,
		cfgPath: cfgPath,
		view:    viewBotList,
		logs:    make(map[string][]string),
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		// Drain log channels for all running bots
		m.drainLogs()
		return m, tickCmd()

	case tea.KeyMsg:
		switch m.view {
		case viewBotList:
			return m.updateBotList(msg)
		case viewBotDetail:
			return m.updateBotDetail(msg)
		case viewHelp:
			return m.updateHelp(msg)
		}
	}

	return m, nil
}

func (m model) View() string {
	switch m.view {
	case viewBotList:
		return m.viewBotList()
	case viewBotDetail:
		return m.viewBotDetail()
	case viewHelp:
		return m.viewHelp()
	}
	return ""
}

// --- Bot List View ---

func (m model) updateBotList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	bots := m.mgr.ListBots()

	switch msg.String() {
	case "q", "ctrl+c":
		m.mgr.StopAll()
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(bots)-1 {
			m.cursor++
		}
	case "enter":
		if len(bots) > 0 {
			m.detailBot = bots[m.cursor].Username
			m.logScroll = 0
			m.view = viewBotDetail
		}
	case "s":
		if len(bots) > 0 {
			username := bots[m.cursor].Username
			m.logs[username] = append(m.logs[username], fmt.Sprintf("[%s] Starting bot...", time.Now().Format("15:04:05")))
			go func() {
				if err := m.mgr.StartBot(username); err != nil {
					m.logs[username] = append(m.logs[username], fmt.Sprintf("[%s] ERROR: %v", time.Now().Format("15:04:05"), err))
				}
			}()
		}
	case "x":
		if len(bots) > 0 {
			username := bots[m.cursor].Username
			if bots[m.cursor].Status != bot.StatusStopped {
				go m.mgr.StopBot(username)
			}
		}
	case "?":
		m.view = viewHelp
	}

	return m, nil
}

func (m model) viewBotList() string {
	var b strings.Builder

	title := titleStyle.Render(" WU Bot Manager ")
	b.WriteString(title + "\n\n")

	bots := m.mgr.ListBots()

	if len(bots) == 0 {
		b.WriteString(dimStyle.Render("  No accounts configured. Edit config.yaml to add accounts."))
		b.WriteString("\n")
	} else {
		// Header
		header := fmt.Sprintf("  %-3s %-15s %-6s %-12s %-12s %-6s %-4s %-6s %-12s %-10s",
			"#", "Username", "Server", "Status", "Mode", "Map", "HP%", "Kills", "Credits/hr", "Runtime")
		b.WriteString(headerStyle.Render(header) + "\n")
		b.WriteString(dimStyle.Render(strings.Repeat("-", 95)) + "\n")

		for i, info := range bots {
			statusStr := formatStatus(info.Status)
			line := fmt.Sprintf("  %-3d %-15s %-6s %-12s %-12s %-6s %-4d %-6d %-12d %-10s",
				i+1, info.Username, info.Server, statusStr, info.Mode,
				info.Map, info.HealthPercent, info.Kills, info.CreditsPerHr, info.RunTime)

			if i == m.cursor {
				b.WriteString(selectedStyle.Render(line) + "\n")
			} else {
				b.WriteString(normalStyle.Render(line) + "\n")
			}
		}
	}

	b.WriteString("\n")

	// Status bar
	helpLine := "  s:start  x:stop  enter:detail  ?:help  q:quit"
	b.WriteString(dimStyle.Render(helpLine))

	return b.String()
}

// --- Bot Detail View ---

func (m model) updateBotDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.view = viewBotList
	case "s":
		engine := m.mgr.GetBot(m.detailBot)
		if engine == nil {
			go m.mgr.StartBot(m.detailBot)
		} else {
			go m.mgr.StopBot(m.detailBot)
		}
	case "j", "down":
		m.logScroll++
	case "k", "up":
		if m.logScroll > 0 {
			m.logScroll--
		}
	case "q":
		m.view = viewBotList
	}
	return m, nil
}

func (m model) viewBotDetail() string {
	var b strings.Builder

	title := titleStyle.Render(fmt.Sprintf(" %s ", m.detailBot))
	b.WriteString(title + "\n\n")

	engine := m.mgr.GetBot(m.detailBot)

	// Left panel: stats
	var statsLines []string
	if engine != nil {
		snap := engine.GetStatsSnapshot()
		statsLines = append(statsLines,
			headerStyle.Render("Status")+": "+formatStatus(engine.GetStatus()),
			headerStyle.Render("Message")+": "+snap.MessageState,
			"",
			headerStyle.Render("Health")+": "+renderBar(snap.HealthPercent, 20, hpColor(snap.HealthPercent)),
			headerStyle.Render("Shield")+": "+renderBar(snap.ShieldPercent, 20, shieldBar),
			"",
			fmt.Sprintf("%s: %s", headerStyle.Render("Map"), snap.Map),
			fmt.Sprintf("%s: %d, %d", headerStyle.Render("Position"), snap.PosX, snap.PosY),
			fmt.Sprintf("%s: %d", headerStyle.Render("Config"), snap.ConfigNum),
			"",
			fmt.Sprintf("%s: %d  %s: %d  %s: %s",
				headerStyle.Render("Kills"), snap.Kills,
				headerStyle.Render("Deaths"), snap.Deaths,
				headerStyle.Render("K/D"), snap.KDRatio),
			"",
			fmt.Sprintf("%s: %d/hr", headerStyle.Render("Credits"), snap.CreditsPerHr),
			fmt.Sprintf("%s: %d/hr", headerStyle.Render("PLT"), snap.PLTPerHr),
			fmt.Sprintf("%s: %d/hr", headerStyle.Render("Honor"), snap.HonorPerHr),
			"",
			fmt.Sprintf("%s: %d  %s: %d  %s: %d",
				headerStyle.Render("Cargo"), snap.CargoBoxes,
				headerStyle.Render("Resource"), snap.ResourceBoxes,
				headerStyle.Render("Green"), snap.GreenBoxes),
			"",
			fmt.Sprintf("%s: %s", headerStyle.Render("Runtime"), snap.RunTime),
			fmt.Sprintf("%s: %d", headerStyle.Render("Credits"), snap.Credits),
			fmt.Sprintf("%s: %d", headerStyle.Render("PLT"), snap.PLT),
		)
	} else {
		statsLines = append(statsLines, dimStyle.Render("Bot not running"))
	}

	statsPanel := borderStyle.Width(40).Render(strings.Join(statsLines, "\n"))

	// Right panel: logs
	logs := m.logs[m.detailBot]
	maxLogLines := 25
	start := len(logs) - maxLogLines - m.logScroll
	if start < 0 {
		start = 0
	}
	end := start + maxLogLines
	if end > len(logs) {
		end = len(logs)
	}

	var logLines []string
	for i := start; i < end; i++ {
		logLines = append(logLines, logStyle.Render(logs[i]))
	}
	if len(logLines) == 0 {
		logLines = append(logLines, dimStyle.Render("No logs yet..."))
	}

	logPanel := borderStyle.Width(m.width - 44).Render(strings.Join(logLines, "\n"))

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, statsPanel, "  ", logPanel))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  esc:back  s:start/stop  j/k:scroll logs"))

	return b.String()
}

// --- Help View ---

func (m model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?":
		m.view = viewBotList
	}
	return m, nil
}

func (m model) viewHelp() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Help ") + "\n\n")

	keys := []struct{ key, desc string }{
		{"enter", "View bot detail"},
		{"s", "Start selected bot"},
		{"x", "Stop selected bot"},
		{"j/k", "Navigate / scroll"},
		{"esc", "Back to list"},
		{"?", "Toggle help"},
		{"q", "Quit"},
	}

	for _, k := range keys {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			helpKeyStyle.Render(fmt.Sprintf("%-8s", k.key)),
			helpDescStyle.Render(k.desc)))
	}

	b.WriteString("\n" + dimStyle.Render("  Press esc to go back"))
	return b.String()
}

// --- Helpers ---

func (m *model) drainLogs() {
	for _, acc := range m.mgr.GetConfig().Accounts {
		engine := m.mgr.GetBot(acc.Username)
		if engine == nil {
			continue
		}
		for {
			select {
			case entry := <-engine.LogCh():
				line := fmt.Sprintf("[%s] %s", entry.Time.Format("15:04:05"), entry.Message)
				m.logs[acc.Username] = append(m.logs[acc.Username], line)
				// Ring buffer: keep last 500 lines
				if len(m.logs[acc.Username]) > 500 {
					m.logs[acc.Username] = m.logs[acc.Username][len(m.logs[acc.Username])-500:]
				}
			default:
				goto done
			}
		}
	done:
	}
}

func formatStatus(s bot.BotStatus) string {
	switch s {
	case bot.StatusRunning:
		return successStyle.Render("Running")
	case bot.StatusConnecting:
		return warningStyle.Render("Connecting")
	case bot.StatusError:
		return errorStyle.Render("Error")
	case bot.StatusBreak:
		return warningStyle.Render("Break")
	default:
		return dimStyle.Render("Stopped")
	}
}

func hpColor(pct int) lipgloss.Style {
	if pct > 70 {
		return hpBarFull
	}
	if pct > 30 {
		return hpBarMed
	}
	return hpBarLow
}

func renderBar(pct, width int, style lipgloss.Style) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := pct * width / 100
	empty := width - filled
	return style.Render(strings.Repeat("█", filled)) + dimStyle.Render(strings.Repeat("░", empty)) + fmt.Sprintf(" %d%%", pct)
}

// Run starts the Bubble Tea TUI.
func Run(mgr *manager.BotManager, cfgPath string) error {
	p := tea.NewProgram(initialModel(mgr, cfgPath), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
