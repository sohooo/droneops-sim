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
	program teaProgram
}

// NewTUIWriter starts a bubbletea program and returns a TUIWriter.
func NewTUIWriter(cfg *config.SimulationConfig) *TUIWriter {
	m := newTUIModel(cfg)
	p := tea.NewProgram(m)
	go func() { _ = p.Start() }()
	return &TUIWriter{program: p}
}

// Write implements TelemetryWriter.
func (w *TUIWriter) Write(row telemetry.TelemetryRow) error {
	line := fmt.Sprintf("[%s] cluster=%s mission=%s drone=%s lat=%.5f lon=%.5f alt=%.1f batt=%.1f pattern=%s spd=%.1f hdg=%.1f status=%s",
		row.Timestamp.Format(time.RFC3339), row.ClusterID, row.MissionID, row.DroneID, row.Lat, row.Lon, row.Alt, row.Battery, row.MovementPattern, row.SpeedMPS, row.HeadingDeg, row.Status)
	w.program.Send(logMsg{line: line})
	return nil
}

// WriteDetection implements DetectionWriter.
func (w *TUIWriter) WriteDetection(d enemy.DetectionRow) error {
	line := fmt.Sprintf("[%s] DETECT drone=%s enemy=%s type=%s lat=%.5f lon=%.5f alt=%.1f conf=%.2f",
		d.Timestamp.Format(time.RFC3339), d.DroneID, d.EnemyID, d.EnemyType, d.Lat, d.Lon, d.Alt, d.Confidence)
	w.program.Send(logMsg{line: line})
	return nil
}

// WriteSwarmEvent implements SwarmEventWriter.
func (w *TUIWriter) WriteSwarmEvent(e telemetry.SwarmEventRow) error {
	line := fmt.Sprintf("[%s] SWARM type=%s drones=%v", e.Timestamp.Format(time.RFC3339), e.EventType, e.DroneIDs)
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
	cfg          *config.SimulationConfig
	table        table.Model
	vp           viewport.Model
	logs         []string
	state        telemetry.SimulationStateRow
	admin        bool
	header       string
	headerHeight int
}

func newTUIModel(cfg *config.SimulationConfig) tuiModel {
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
	m := tuiModel{cfg: cfg, table: t, vp: vp}
	m.header = m.renderHeader()
	m.headerHeight = lipgloss.Height(m.header)
	return m
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.vp.Width = msg.Width
		m.vp.Height = msg.Height - m.headerHeight - 1
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
	return fmt.Sprintf("%s\n%s\n%s", m.header, m.vp.View(), bottom)
}

func (m tuiModel) renderHeader() string {
	var b strings.Builder
	b.WriteString(m.table.View())
	b.WriteString("\n")
	b.WriteString(renderMissionTree(m.cfg))
	return b.String()
}

func renderMissionTree(cfg *config.SimulationConfig) string {
	var b strings.Builder
	b.WriteString("Missions\n")
	for i, ms := range cfg.Missions {
		prefix := "├─"
		if i == len(cfg.Missions)-1 {
			prefix = "└─"
		}
		b.WriteString(fmt.Sprintf("%s %s (%s)\n", prefix, ms.ID, ms.Name))
	}
	return b.String()
}

func (m tuiModel) renderBottom() string {
	color := lipgloss.Color("9")
	if m.admin {
		color = lipgloss.Color("10")
	}
	indicator := lipgloss.NewStyle().Foreground(color).Render("●")
	state := fmt.Sprintf("STATE comm_loss=%.2f msgs=%d sensor=%.2f weather=%.2f chaos=%t",
		m.state.CommunicationLoss, m.state.MessagesSent, m.state.SensorNoise, m.state.WeatherImpact, m.state.ChaosMode)
	return fmt.Sprintf("%s | Admin UI %s", state, indicator)
}
