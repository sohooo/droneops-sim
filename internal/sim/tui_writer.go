package sim

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

// TUIWriter renders telemetry using a bubbletea TUI.
type TUIWriter struct {
	program       teaProgram
	missionColors map[string]string
	colorIdx      int
}

// NewTUIWriter starts a bubbletea program and returns a TUIWriter.
func NewTUIWriter(cfg *config.SimulationConfig) *TUIWriter {
	mc := make(map[string]string)
	w := &TUIWriter{missionColors: mc}
	m := newTUIModel(cfg, mc)
	p := tea.NewProgram(m)
	w.program = p
	for _, ms := range cfg.Missions {
		w.getMissionColor(ms.ID)
	}
	go func() { _ = p.Start() }()
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

type tuiModel struct {
	cfg           *config.SimulationConfig
	table         table.Model
	vp            viewport.Model
	logs          []string
	state         telemetry.SimulationStateRow
	admin         bool
	header        string
	headerHeight  int
	missionColors map[string]string
	width         int
}

func newTUIModel(cfg *config.SimulationConfig, missionColors map[string]string) tuiModel {
	cols := []table.Column{
		{Title: "Config", Width: 20},
		{Title: "Value", Width: 20},
	}
	rows := []table.Row{
		{"Follow Confidence", fmt.Sprintf("%.0f", cfg.FollowConfidence)},
		{"Mission Criticality", cfg.MissionCriticality},
		{"Detection Radius (m)", fmt.Sprintf("%.0f", cfg.DetectionRadiusM)},
		{"Sensor Noise", fmt.Sprintf("%.2f", cfg.SensorNoise)},
		{"Terrain Occlusion", fmt.Sprintf("%.2f", cfg.TerrainOcclusion)},
		{"Weather Impact", fmt.Sprintf("%.2f", cfg.WeatherImpact)},
		{"Communication Loss", fmt.Sprintf("%.2f", cfg.CommunicationLoss)},
		{"Bandwidth Limit", fmt.Sprintf("%d", cfg.BandwidthLimit)},
	}
	t := table.New(table.WithColumns(cols), table.WithRows(rows))
	vp := viewport.New(0, 0)
	m := tuiModel{cfg: cfg, table: t, vp: vp, missionColors: missionColors}
	return m
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.table.SetWidth(msg.Width / 2)
		m.vp.Width = msg.Width
		m.header = m.renderHeader()
		m.headerHeight = lipgloss.Height(m.header)
		bottomHeight := lipgloss.Height(m.renderBottom())
		m.vp.Height = msg.Height - m.headerHeight - bottomHeight - 2
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	case logMsg:
		m.logs = append(m.logs, msg.line)
		if len(m.logs) > 1000 {
			m.logs = m.logs[len(m.logs)-1000:]
		}
		m.vp.SetContent(strings.Join(m.logs, "\n"))
		m.vp.GotoBottom()
	case stateMsg:
		m.state = msg.SimulationStateRow
	case adminMsg:
		m.admin = msg.active
	}
	return m, nil
}

func (m tuiModel) View() string {
	bottom := m.renderBottom()
	divider := strings.Repeat("─", m.vp.Width)
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s", m.header, divider, m.vp.View(), divider, bottom)
}

func (m tuiModel) renderHeader() string {
	tableView := m.table.View()
	missions := renderMissionTree(m.cfg, m.missionColors)
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("│")
	return lipgloss.JoinHorizontal(lipgloss.Top, tableView, sep, missions)
}

func renderMissionTree(cfg *config.SimulationConfig, colors map[string]string) string {
	var b strings.Builder
	b.WriteString("Missions\n")
	for i, ms := range cfg.Missions {
		prefix := "├─"
		if i == len(cfg.Missions)-1 {
			prefix = "└─"
		}
		c := colors[ms.ID]
		b.WriteString(fmt.Sprintf("%s %s%s%s %s - %s\n", prefix, c, ms.ID, colorReset, ms.Name, ms.Description))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m tuiModel) renderBottom() string {
	color := lipgloss.Color("9")
	if m.admin {
		color = lipgloss.Color("10")
	}
	indicator := lipgloss.NewStyle().Foreground(color).Render("●")
	state := fmt.Sprintf("%sSTATE%s comm_loss=%.2f msgs=%d sensor=%.2f weather=%.2f chaos=%t",
		colorBlue, colorReset, m.state.CommunicationLoss, m.state.MessagesSent, m.state.SensorNoise, m.state.WeatherImpact, m.state.ChaosMode)
	return fmt.Sprintf("%s | Admin UI %s | q:quit ctrl+c:quit", state, indicator)
}
