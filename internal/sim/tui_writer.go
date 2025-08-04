package sim

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/muesli/reflow/wordwrap"

	"droneops-sim/internal/config"
	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

// teaProgram abstracts bubbletea.Program for testing.
type teaProgram interface {
	Send(tea.Msg)
}

// logMsg carries a log line for the viewport.
type logMsg struct{ line string }

// stateMsg carries a simulation state update.
type stateMsg struct{ telemetry.SimulationStateRow }

// adminMsg reports admin UI status.
type adminMsg struct{ active bool }

type setSpawnMsg struct{ fn func(enemy.Enemy) }

// TUIWriter renders telemetry using a bubbletea TUI.
type TUIWriter struct {
	program       teaProgram
	missionColors map[string]string
	colorIdx      int
	done          chan struct{}
	sendSignal    atomic.Bool
}

// NewTUIWriter starts a bubbletea program and returns a TUIWriter.
func NewTUIWriter(cfg *config.SimulationConfig) *TUIWriter {
	mc := make(map[string]string)
	w := &TUIWriter{missionColors: mc, done: make(chan struct{})}
	w.sendSignal.Store(true)
	m := newTUIModel(cfg, mc)
	p := tea.NewProgram(m, tea.WithAltScreen())
	w.program = p
	for _, ms := range cfg.Missions {
		w.getMissionColor(ms.ID)
	}
	go func() {
		_ = p.Start()
		close(w.done)
		if w.sendSignal.Load() {
			if proc, err := os.FindProcess(os.Getpid()); err == nil {
				_ = proc.Signal(os.Interrupt)
			}
		}
	}()
	return w
}

func (w *TUIWriter) getMissionColor(id string) string {
	if c, ok := w.missionColors[id]; ok {
		return c
	}
	c := missionPalette[w.colorIdx%len(missionPalette)]
	w.missionColors[id] = c
	w.colorIdx++
	return c
}

// Write implements TelemetryWriter.
func (w *TUIWriter) Write(row telemetry.TelemetryRow) error {
	mColor := w.getMissionColor(row.MissionID)
	statusColor := colorGreen
	switch row.Status {
	case telemetry.StatusFailure:
		statusColor = colorRed
	case telemetry.StatusLowBattery:
		statusColor = colorYellow
	}

	line := fmt.Sprintf("%s[%s]%s %scluster=%s%s %smission=%s%s %sdrone=%s%s %slat=%.5f%s %slon=%.5f%s %salt=%.1f%s %sbatt=%.1f%s %spattern=%s%s %sspd=%.1f%s %shdg=%.1f%s %sprev=(%.5f,%.5f,%.1f)%s %sstatus=%s%s",
		colorGray, row.Timestamp.Format(time.RFC3339), colorReset,
		colorBlue, row.ClusterID, colorReset,
		mColor, row.MissionID, colorReset,
		colorWhite(), row.DroneID, colorReset,
		colorGreen, row.Lat, colorReset,
		colorYellow, row.Lon, colorReset,
		colorMagenta, row.Alt, colorReset,
		colorCyan, row.Battery, colorReset,
		colorBlue, row.MovementPattern, colorReset,
		colorYellow, row.SpeedMPS, colorReset,
		colorCyan, row.HeadingDeg, colorReset,
		colorGray, row.PreviousPosition.Lat, row.PreviousPosition.Lon, row.PreviousPosition.Alt, colorReset,
		statusColor, row.Status, colorReset,
	)
	if row.Follow {
		line += fmt.Sprintf(" %sfollow%s", colorMagenta, colorReset)
	}
	w.program.Send(logMsg{line: line})
	return nil
}

// WriteDetection implements DetectionWriter.
func (w *TUIWriter) WriteDetection(d enemy.DetectionRow) error {
	line := fmt.Sprintf("%s[%s]%s %sDETECT%s drone=%s enemy=%s type=%s lat=%.5f lon=%.5f alt=%.1f conf=%.2f",
		colorGray, d.Timestamp.Format(time.RFC3339), colorReset,
		colorRed, colorReset, d.DroneID, d.EnemyID, d.EnemyType,
		d.Lat, d.Lon, d.Alt, d.Confidence)
	w.program.Send(logMsg{line: line})
	return nil
}

// WriteSwarmEvent implements SwarmEventWriter.
func (w *TUIWriter) WriteSwarmEvent(e telemetry.SwarmEventRow) error {
	line := fmt.Sprintf("%s[%s]%s %sSWARM%s type=%s drones=%v",
		colorGray, e.Timestamp.Format(time.RFC3339), colorReset,
		colorCyan, colorReset, e.EventType, e.DroneIDs)
	if e.EnemyID != "" {
		line += fmt.Sprintf(" enemy=%s", e.EnemyID)
	}
	w.program.Send(logMsg{line: line})
	return nil
}

// WriteState implements StateWriter.
func (w *TUIWriter) WriteState(row telemetry.SimulationStateRow) error {
	w.program.Send(stateMsg{SimulationStateRow: row})
	return nil
}

// WriteBatch outputs multiple telemetry rows.
func (w *TUIWriter) WriteBatch(rows []telemetry.TelemetryRow) error {
	for _, r := range rows {
		_ = w.Write(r)
	}
	return nil
}

// WriteDetections outputs multiple detection rows.
func (w *TUIWriter) WriteDetections(rows []enemy.DetectionRow) error {
	for _, d := range rows {
		_ = w.WriteDetection(d)
	}
	return nil
}

// WriteSwarmEvents outputs multiple swarm events.
func (w *TUIWriter) WriteSwarmEvents(rows []telemetry.SwarmEventRow) error {
	for _, e := range rows {
		_ = w.WriteSwarmEvent(e)
	}
	return nil
}

// WriteStates outputs multiple state rows.
func (w *TUIWriter) WriteStates(rows []telemetry.SimulationStateRow) error {
	for _, r := range rows {
		_ = w.WriteState(r)
	}
	return nil
}

// SetAdminStatus updates the admin UI indicator.
func (w *TUIWriter) SetAdminStatus(active bool) {
	w.program.Send(adminMsg{active: active})
}

// SetSpawner registers a callback to spawn enemies.
func (w *TUIWriter) SetSpawner(fn func(enemy.Enemy)) {
	w.program.Send(setSpawnMsg{fn: fn})
}

// Close shuts down the TUI program and waits for cleanup.
func (w *TUIWriter) Close() error {
	w.sendSignal.Store(false)
	if w.program != nil {
		w.program.Send(tea.Quit())
	}
	if w.done != nil {
		<-w.done
	}
	return nil
}

type tuiModel struct {
	cfg           *config.SimulationConfig
	table         table.Model
	vp            viewport.Model
	logs          []string
	state         telemetry.SimulationStateRow
	admin         bool
	wrap          bool
	autoscroll    bool
	header        string
	headerHeight  int
	height        int
	missionColors map[string]string
	enemies       []enemy.Enemy
	spawn         func(enemy.Enemy)
	enemyInput    textinput.Model
	enemyDialog   bool
}

func newTUIModel(cfg *config.SimulationConfig, missionColors map[string]string) tuiModel {
	cols := []table.Column{
		{Title: "Config", Width: 20},
		{Title: "Value", Width: 10},
		{Title: "Config", Width: 20},
		{Title: "Value", Width: 10},
	}
	rows := []table.Row{
		{"Follow Confidence", fmt.Sprintf("%.0f", cfg.FollowConfidence), "Mission Criticality", cfg.MissionCriticality},
		{"Detection Radius (m)", fmt.Sprintf("%.0f", cfg.DetectionRadiusM), "Sensor Noise", fmt.Sprintf("%.2f", cfg.SensorNoise)},
		{"Terrain Occlusion", fmt.Sprintf("%.2f", cfg.TerrainOcclusion), "Weather Impact", fmt.Sprintf("%.2f", cfg.WeatherImpact)},
		{"Communication Loss", fmt.Sprintf("%.2f", cfg.CommunicationLoss), "Bandwidth Limit", fmt.Sprintf("%d", cfg.BandwidthLimit)},
	}
	t := table.New(table.WithColumns(cols), table.WithRows(rows), table.WithHeight(len(rows)+1))
	vp := viewport.New(0, 0)
	m := tuiModel{cfg: cfg, table: t, vp: vp, missionColors: missionColors, autoscroll: true}
	return m
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width / 2)
		m.vp.Width = msg.Width
		m.height = msg.Height
		m.header = m.renderHeader()
		m.headerHeight = lipgloss.Height(m.header)
		bottomHeight := lipgloss.Height(m.renderBottom())
		enemyHeight := lipgloss.Height(m.renderEnemies())
		m.vp.Height = m.height - m.headerHeight - bottomHeight - enemyHeight - 3
		m.refreshViewport()
	case tea.KeyMsg:
		if m.enemyDialog {
			switch msg.Type {
			case tea.KeyEnter:
				en, err := parseEnemyInput(m.enemyInput.Value())
				if err == nil {
					if m.spawn != nil {
						m.spawn(en)
					}
					m.enemies = append(m.enemies, en)
				}
				m.enemyDialog = false
				bottomHeight := lipgloss.Height(m.renderBottom())
				enemyHeight := lipgloss.Height(m.renderEnemies())
				m.vp.Height = m.height - m.headerHeight - bottomHeight - enemyHeight - 3
			case tea.KeyEsc:
				m.enemyDialog = false
				bottomHeight := lipgloss.Height(m.renderBottom())
				enemyHeight := lipgloss.Height(m.renderEnemies())
				m.vp.Height = m.height - m.headerHeight - bottomHeight - enemyHeight - 3
			default:
				var cmd tea.Cmd
				m.enemyInput, cmd = m.enemyInput.Update(msg)
				return m, cmd
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "w":
			m.wrap = !m.wrap
			m.refreshViewport()
			m.header = m.renderHeader()
			m.headerHeight = lipgloss.Height(m.header)
			bottomHeight := lipgloss.Height(m.renderBottom())
			enemyHeight := lipgloss.Height(m.renderEnemies())
			m.vp.Height = m.height - m.headerHeight - bottomHeight - enemyHeight - 3
		case "s":
			m.autoscroll = !m.autoscroll
			if m.autoscroll {
				m.vp.GotoBottom()
			}
		case "e":
			m.enemyInput = textinput.New()
			m.enemyInput.Placeholder = "type,lat,lon,alt"
			m.enemyInput.Focus()
			m.enemyDialog = true
			bottomHeight := lipgloss.Height(m.renderBottom())
			enemyHeight := lipgloss.Height(m.renderEnemies())
			m.vp.Height = m.height - m.headerHeight - bottomHeight - enemyHeight - 3
		}
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	case logMsg:
		m.logs = append(m.logs, msg.line)
		if len(m.logs) > 1000 {
			m.logs = m.logs[len(m.logs)-1000:]
		}
		m.refreshViewport()
	case stateMsg:
		m.state = msg.SimulationStateRow
	case adminMsg:
		m.admin = msg.active
	case setSpawnMsg:
		m.spawn = msg.fn
	}
	return m, nil
}

func (m *tuiModel) refreshViewport() {
	var lines []string
	for _, l := range m.logs {
		if m.wrap {
			lines = append(lines, wordwrap.String(l, m.vp.Width))
		} else {
			lines = append(lines, l)
		}
	}
	m.vp.SetContent(strings.Join(lines, "\n"))
	if m.autoscroll {
		m.vp.GotoBottom()
	}
}

func (m tuiModel) View() string {
	bottom := m.renderBottom()
	divider := strings.Repeat("─", m.vp.Width)
	enemies := m.renderEnemies()
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s", m.header, divider, m.vp.View(), divider, enemies, divider, bottom)
}

func (m tuiModel) renderHeader() string {
	tableView := m.table.View()
	missionsWidth := m.vp.Width/2 - 1
	missions := renderMissionTree(m.cfg, m.missionColors, m.wrap, missionsWidth)
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("│")
	return lipgloss.JoinHorizontal(lipgloss.Top, tableView, sep, missions)
}

func renderMissionTree(cfg *config.SimulationConfig, colors map[string]string, wrap bool, width int) string {
	var b strings.Builder
	b.WriteString("Missions\n")
	for i, ms := range cfg.Missions {
		prefix := "├─"
		if i == len(cfg.Missions)-1 {
			prefix = "└─"
		}
		c := colors[ms.ID]
		line := fmt.Sprintf("%s %s%s%s %s - %s", prefix, c, ms.ID, colorReset, ms.Name, ms.Description)
		if wrap && width > 0 {
			line = wordwrap.String(line, width)
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m tuiModel) renderBottom() string {
	adminColor := lipgloss.Color("9")
	if m.admin {
		adminColor = lipgloss.Color("10")
	}
	wrapColor := lipgloss.Color("9")
	if m.wrap {
		wrapColor = lipgloss.Color("10")
	}
	scrollColor := lipgloss.Color("10")
	if !m.autoscroll {
		scrollColor = lipgloss.Color("9")
	}
	adminIndicator := lipgloss.NewStyle().Foreground(adminColor).Render("●")
	wrapIndicator := lipgloss.NewStyle().Foreground(wrapColor).Render("●")
	scrollIndicator := lipgloss.NewStyle().Foreground(scrollColor).Render("●")
	state := fmt.Sprintf("%sSTATE%s comm_loss=%.2f msgs=%d sensor=%.2f weather=%.2f chaos=%t",
		colorBlue, colorReset, m.state.CommunicationLoss, m.state.MessagesSent, m.state.SensorNoise, m.state.WeatherImpact, m.state.ChaosMode)
	keys := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("q:quit w:wrap s:scroll e:enemy")
	return fmt.Sprintf("%s | Admin UI %s | Wrap %s | Scroll %s | %s", state, adminIndicator, wrapIndicator, scrollIndicator, keys)
}

func (m tuiModel) renderEnemies() string {
	if m.enemyDialog {
		return fmt.Sprintf("Spawn Enemy (type,lat,lon,alt) - Enter to spawn, Esc to cancel: %s", m.enemyInput.View())
	}
	if len(m.enemies) == 0 {
		return "Enemies: none"
	}
	var b strings.Builder
	b.WriteString("Enemies:\n")
	for _, e := range m.enemies {
		b.WriteString(fmt.Sprintf("%s %s lat=%.5f lon=%.5f alt=%.1f\n", e.ID, e.Type, e.Position.Lat, e.Position.Lon, e.Position.Alt))
	}
	return strings.TrimRight(b.String(), "\n")
}

func parseEnemyInput(val string) (enemy.Enemy, error) {
	parts := strings.Split(val, ",")
	if len(parts) < 4 {
		return enemy.Enemy{}, fmt.Errorf("expected type,lat,lon,alt")
	}
	typ := enemy.EnemyType(strings.TrimSpace(parts[0]))
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return enemy.Enemy{}, err
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return enemy.Enemy{}, err
	}
	alt, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
	if err != nil {
		return enemy.Enemy{}, err
	}
	return enemy.Enemy{ID: uuid.New().String(), Type: typ, Position: telemetry.Position{Lat: lat, Lon: lon, Alt: alt}}, nil
}
