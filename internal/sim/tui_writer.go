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

// detectionMsg carries a detection log line.
type detectionMsg struct{ line string }

// swarmMsg carries a swarm event log line.
type swarmMsg struct{ line string }

// stateMsg carries a simulation state update.
type stateMsg struct{ telemetry.SimulationStateRow }

// adminMsg reports admin UI status.
type adminMsg struct{ active bool }

type setSpawnMsg struct{ fn func(enemy.Enemy) }
type setRemoveMsg struct{ fn func(string) }
type setStatusMsg struct {
	fn func(string, enemy.EnemyStatus)
}
type telemetryMsg struct{ telemetry.TelemetryRow }

const (
	fallbackEnemyInput  = "vehicle,0,0,0"
	enemyOffset         = 0.0001
	maxSectionHeightPct = 0.2
)

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
	w.program.Send(telemetryMsg{row})
	return nil
}

// WriteDetection implements DetectionWriter.
func (w *TUIWriter) WriteDetection(d enemy.DetectionRow) error {
	line := fmt.Sprintf("%s[%s]%s %sDETECT%s %sdrone=%s%s %senemy=%s%s %stype=%s%s %slat=%.5f%s %slon=%.5f%s %salt=%.1f%s %sconf=%.2f%s",
		colorGray, d.Timestamp.Format(time.RFC3339), colorReset,
		colorRed, colorReset,
		colorWhite(), d.DroneID, colorReset,
		colorBlue, d.EnemyID, colorReset,
		colorMagenta, d.EnemyType, colorReset,
		colorGreen, d.Lat, colorReset,
		colorYellow, d.Lon, colorReset,
		colorCyan, d.Alt, colorReset,
		colorGreen, d.Confidence, colorReset)
	w.program.Send(detectionMsg{line: line})
	return nil
}

// WriteSwarmEvent implements SwarmEventWriter.
func (w *TUIWriter) WriteSwarmEvent(e telemetry.SwarmEventRow) error {
	line := fmt.Sprintf("%s[%s]%s %sSWARM%s %stype=%s%s %sdrones=%v%s",
		colorGray, e.Timestamp.Format(time.RFC3339), colorReset,
		colorCyan, colorReset,
		colorBlue, e.EventType, colorReset,
		colorWhite(), e.DroneIDs, colorReset)
	if e.EnemyID != "" {
		line += fmt.Sprintf(" %senemy=%s%s", colorMagenta, e.EnemyID, colorReset)
	}
	w.program.Send(swarmMsg{line: line})
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

// SetEnemyRemover registers a callback to remove enemies.
func (w *TUIWriter) SetEnemyRemover(fn func(string)) {
	w.program.Send(setRemoveMsg{fn: fn})
}

// SetEnemyStatusUpdater registers a callback to update enemy status.
func (w *TUIWriter) SetEnemyStatusUpdater(fn func(string, enemy.EnemyStatus)) {
	w.program.Send(setStatusMsg{fn: fn})
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
	cfg             *config.SimulationConfig
	table           table.Model
	vp              viewport.Model
	detVP           viewport.Model
	swarmVP         viewport.Model
	logs            []string
	detLogs         []string
	swarmLogs       []string
	state           telemetry.SimulationStateRow
	admin           bool
	wrap            bool
	autoscroll      bool
	header          string
	headerHeight    int
	height          int
	missionColors   map[string]string
	enemies         []enemy.Enemy
	spawn           func(enemy.Enemy)
	enemyInput      textinput.Model
	enemyDialog     bool
	editEnemyInput  textinput.Model
	editEnemyDialog bool
	remove          func(string)
	updateStatus    func(string, enemy.EnemyStatus)
	lastDrone       telemetry.Position
	haveDrone       bool
	summary         bool
	help            bool
	showMissions    bool
	showEnemies     bool
	droneBatteries  map[string]float64
	missionTotals   map[string]int
	missionCounts   map[string]map[string]struct{}
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
	detVP := viewport.New(0, 0)
	swarmVP := viewport.New(0, 0)
	missionTotals := make(map[string]int)
	for _, f := range cfg.Fleets {
		missionTotals[f.MissionID] += f.Count
	}
	m := tuiModel{
		cfg:            cfg,
		table:          t,
		vp:             vp,
		detVP:          detVP,
		swarmVP:        swarmVP,
		missionColors:  missionColors,
		autoscroll:     true,
		showMissions:   true,
		showEnemies:    true,
		droneBatteries: make(map[string]float64),
		missionTotals:  missionTotals,
		missionCounts:  make(map[string]map[string]struct{}),
	}
	return m
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tableWidth := msg.Width
		if m.showMissions {
			tableWidth = msg.Width / 2
		}
		m.table.SetWidth(tableWidth)
		m.vp.Width = msg.Width
		m.detVP.Width = msg.Width
		m.swarmVP.Width = msg.Width
		m.height = msg.Height
		m.header = m.renderHeader()
		m.headerHeight = lipgloss.Height(m.header)
		m.updateViewportHeight()
		m.refreshViewport()
		m.refreshDetections()
		m.refreshSwarmEvents()
	case tea.KeyMsg:
		if m.enemyDialog {
			switch msg.Type {
			case tea.KeyEnter:
				en, err := parseEnemyInput(m.enemyInput.Value())
				if err == nil {
					if m.spawn != nil {
						go m.spawn(en)
					}
					m.enemies = append(m.enemies, en)
				}
				m.enemyDialog = false
				m.updateViewportHeight()
			case tea.KeyEsc:
				m.enemyDialog = false
				m.updateViewportHeight()
			default:
				var cmd tea.Cmd
				m.enemyInput, cmd = m.enemyInput.Update(msg)
				return m, cmd
			}
			return m, nil
		}
		if m.editEnemyDialog {
			switch msg.Type {
			case tea.KeyEnter:
				parts := strings.Split(m.editEnemyInput.Value(), ",")
				if len(parts) >= 2 {
					id := strings.TrimSpace(parts[0])
					action := strings.TrimSpace(parts[1])
					switch action {
					case "delete":
						for i, e := range m.enemies {
							if e.ID == id {
								m.enemies = append(m.enemies[:i], m.enemies[i+1:]...)
								if m.remove != nil {
									go m.remove(id)
								}
								break
							}
						}
					case string(enemy.EnemyActive), string(enemy.EnemyNeutralized):
						st := enemy.EnemyStatus(action)
						for i := range m.enemies {
							if m.enemies[i].ID == id {
								m.enemies[i].Status = st
								break
							}
						}
						if m.updateStatus != nil {
							go m.updateStatus(id, st)
						}
					}
				}
				m.editEnemyDialog = false
				m.updateViewportHeight()
			case tea.KeyEsc:
				m.editEnemyDialog = false
				m.updateViewportHeight()
			default:
				var cmd tea.Cmd
				m.editEnemyInput, cmd = m.editEnemyInput.Update(msg)
				return m, cmd
			}
			return m, nil
		}
		if m.help {
			switch msg.String() {
			case "?", "esc":
				m.help = false
				m.updateViewportHeight()
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
			m.updateViewportHeight()
			return m, nil
		case "s":
			m.autoscroll = !m.autoscroll
			if m.autoscroll {
				m.vp.GotoBottom()
				m.detVP.GotoBottom()
				m.swarmVP.GotoBottom()
			}
			return m, nil
		case "e":
			m.enemyInput = textinput.New()
			m.enemyInput.Placeholder = "type,lat,lon,alt"
			val := fallbackEnemyInput
			if m.haveDrone {
				val = fmt.Sprintf("vehicle,%.5f,%.5f,%.1f", m.lastDrone.Lat+enemyOffset, m.lastDrone.Lon+enemyOffset, m.lastDrone.Alt)
			}
			m.enemyInput.SetValue(val)
			m.enemyInput.CursorEnd()
			m.enemyInput.Focus()
			m.enemyDialog = true
			m.updateViewportHeight()
			return m, nil
		case "E":
			m.editEnemyInput = textinput.New()
			m.editEnemyInput.Placeholder = "id,status|delete"
			m.editEnemyInput.Focus()
			m.editEnemyDialog = true
			m.updateViewportHeight()
			return m, nil
		case "m":
			m.showMissions = !m.showMissions
			width := m.vp.Width
			if m.showMissions {
				m.table.SetWidth(width / 2)
			} else {
				m.table.SetWidth(width)
			}
			m.header = m.renderHeader()
			m.headerHeight = lipgloss.Height(m.header)
			m.updateViewportHeight()
			return m, nil
		case "n":
			m.showEnemies = !m.showEnemies
			m.updateViewportHeight()
			return m, nil
		case "t":
			m.summary = !m.summary
			m.updateViewportHeight()
			return m, nil
		case "?":
			m.help = !m.help
			m.updateViewportHeight()
			return m, nil
		}
		if !m.autoscroll {
			switch msg.String() {
			case "j", "down":
				m.vp.LineDown(1)
				m.detVP.LineDown(1)
				m.swarmVP.LineDown(1)
			case "k", "up":
				m.vp.LineUp(1)
				m.detVP.LineUp(1)
				m.swarmVP.LineUp(1)
			case "pgdown", "ctrl+n":
				m.vp.LineDown(10)
				m.detVP.LineDown(10)
				m.swarmVP.LineDown(10)
			case "pgup", "ctrl+p":
				m.vp.LineUp(10)
				m.detVP.LineUp(10)
				m.swarmVP.LineUp(10)
			default:
				var cmd tea.Cmd
				m.vp, cmd = m.vp.Update(msg)
				m.detVP, _ = m.detVP.Update(msg)
				m.swarmVP, _ = m.swarmVP.Update(msg)
				return m, cmd
			}
			return m, nil
		}
		return m, nil
	case logMsg:
		m.logs = append(m.logs, msg.line)
		if len(m.logs) > 1000 {
			m.logs = m.logs[len(m.logs)-1000:]
		}
		m.refreshViewport()
	case detectionMsg:
		m.detLogs = append(m.detLogs, msg.line)
		if len(m.detLogs) > 1000 {
			m.detLogs = m.detLogs[len(m.detLogs)-1000:]
		}
		m.updateViewportHeight()
		m.refreshDetections()
		m.refreshViewport()
	case swarmMsg:
		m.swarmLogs = append(m.swarmLogs, msg.line)
		if len(m.swarmLogs) > 1000 {
			m.swarmLogs = m.swarmLogs[len(m.swarmLogs)-1000:]
		}
		m.updateViewportHeight()
		m.refreshSwarmEvents()
		m.refreshViewport()
	case telemetryMsg:
		m.lastDrone = telemetry.Position{Lat: msg.Lat, Lon: msg.Lon, Alt: msg.Alt}
		m.haveDrone = true
		if m.droneBatteries == nil {
			m.droneBatteries = make(map[string]float64)
		}
		m.droneBatteries[msg.DroneID] = msg.Battery
		if m.missionCounts == nil {
			m.missionCounts = make(map[string]map[string]struct{})
		}
		if m.missionCounts[msg.MissionID] == nil {
			m.missionCounts[msg.MissionID] = make(map[string]struct{})
		}
		m.missionCounts[msg.MissionID][msg.DroneID] = struct{}{}
	case stateMsg:
		m.state = msg.SimulationStateRow
	case adminMsg:
		m.admin = msg.active
	case setSpawnMsg:
		m.spawn = msg.fn
	case setRemoveMsg:
		m.remove = msg.fn
	case setStatusMsg:
		m.updateStatus = msg.fn
	}
	return m, nil
}

func (m *tuiModel) updateViewportHeight() {
	bottomHeight := lipgloss.Height(m.renderBottom())

	maxLines := m.maxSectionLines()

	detLines := len(m.detLogs)
	if detLines == 0 {
		detLines = 1
	}
	if detLines > maxLines {
		detLines = maxLines
	}
	m.detVP.Height = detLines

	swarmLines := len(m.swarmLogs)
	if swarmLines == 0 {
		swarmLines = 1
	}
	if swarmLines > maxLines {
		swarmLines = maxLines
	}
	m.swarmVP.Height = swarmLines

	detHeight := 1 + m.detVP.Height
	swarmHeight := 1 + m.swarmVP.Height
	enemyHeight := 0
	if m.showEnemies || m.enemyDialog || m.editEnemyDialog {
		enemyHeight = lipgloss.Height(m.renderEnemies())
	}
	h := m.height - m.headerHeight - bottomHeight - detHeight - swarmHeight - enemyHeight - 5
	if h < 0 {
		h = 0
	}
	m.vp.Height = h
	if m.autoscroll {
		m.detVP.GotoBottom()
		m.swarmVP.GotoBottom()
		m.vp.GotoBottom()
	}
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

func (m *tuiModel) refreshDetections() {
	content := "none"
	if len(m.detLogs) > 0 {
		content = strings.Join(m.detLogs, "\n")
	}
	m.detVP.SetContent(content)
	if m.autoscroll {
		m.detVP.GotoBottom()
	}
}

func (m *tuiModel) refreshSwarmEvents() {
	content := "none"
	if len(m.swarmLogs) > 0 {
		content = strings.Join(m.swarmLogs, "\n")
	}
	m.swarmVP.SetContent(content)
	if m.autoscroll {
		m.swarmVP.GotoBottom()
	}
}

func (m tuiModel) maxSectionLines() int {
	h := int(float64(m.height) * maxSectionHeightPct)
	if h < 1 {
		h = 1
	}
	return h
}

func (m tuiModel) View() string {
	if m.help {
		return m.renderHelp()
	}
	bottom := m.renderBottom()
	divider := strings.Repeat("─", m.vp.Width)
	sections := []string{
		m.header,
		divider,
		m.vp.View(),
		divider,
		"Detections:",
		m.detVP.View(),
		divider,
		"Swarm Events:",
		m.swarmVP.View(),
	}
	if m.showEnemies || m.enemyDialog || m.editEnemyDialog {
		enemies := m.renderEnemies()
		sections = append(sections, divider, enemies)
	}
	sections = append(sections, divider, bottom)
	return strings.Join(sections, "\n")
}

func (m tuiModel) renderHeader() string {
	tableView := m.table.View()
	if !m.showMissions {
		return tableView
	}
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

func (m tuiModel) renderSummary() string {
	total := len(m.droneBatteries)
	var sum float64
	for _, b := range m.droneBatteries {
		sum += b
	}
	avg := 0.0
	if total > 0 {
		avg = sum / float64(total)
	}
	var missionParts []string
	for _, ms := range m.cfg.Missions {
		totalMission := m.missionTotals[ms.ID]
		active := len(m.missionCounts[ms.ID])
		pct := 0.0
		if totalMission > 0 {
			pct = float64(active) / float64(totalMission) * 100
		}
		c := m.missionColors[ms.ID]
		part := fmt.Sprintf("%s%s%s=%d/%d(%.0f%%)%s", c, ms.ID, colorReset, active, totalMission, pct, colorReset)
		missionParts = append(missionParts, part)
	}
	missions := strings.Join(missionParts, " ")
	summary := fmt.Sprintf("%sSUMMARY%s %sdrones=%d%s %savg_batt=%.1f%s", colorBlue, colorReset, colorGreen, total, colorReset, colorCyan, avg, colorReset)
	if missions != "" {
		summary = fmt.Sprintf("%s %s", summary, missions)
	}
	return summary
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
	summaryColor := lipgloss.Color("9")
	if m.summary {
		summaryColor = lipgloss.Color("10")
	}
	summaryIndicator := lipgloss.NewStyle().Foreground(summaryColor).Render("●")
	helpColor := lipgloss.Color("9")
	if m.help {
		helpColor = lipgloss.Color("10")
	}
	helpIndicator := lipgloss.NewStyle().Foreground(helpColor).Render("●")
	missionsColor := lipgloss.Color("10")
	if !m.showMissions {
		missionsColor = lipgloss.Color("9")
	}
	missionsIndicator := lipgloss.NewStyle().Foreground(missionsColor).Render("●")
	enemiesColor := lipgloss.Color("10")
	if !m.showEnemies {
		enemiesColor = lipgloss.Color("9")
	}
	enemiesIndicator := lipgloss.NewStyle().Foreground(enemiesColor).Render("●")
	state := fmt.Sprintf("%sSTATE%s %scomm_loss=%.2f%s %smsgs=%d%s %ssensor=%.2f%s %sweather=%.2f%s %schaos=%t%s",
		colorBlue, colorReset,
		colorYellow, m.state.CommunicationLoss, colorReset,
		colorGreen, m.state.MessagesSent, colorReset,
		colorMagenta, m.state.SensorNoise, colorReset,
		colorCyan, m.state.WeatherImpact, colorReset,
		colorRed, m.state.ChaosMode, colorReset)
	keys := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("q:quit w:wrap s:scroll e:enemy E:edit t:summary m:missions n:enemies ?:help")
	line := fmt.Sprintf("%s | Admin UI %s | Wrap %s | Scroll %s | Summary %s | Help %s | Missions %s | Enemies %s | %s", state, adminIndicator, wrapIndicator, scrollIndicator, summaryIndicator, helpIndicator, missionsIndicator, enemiesIndicator, keys)
	if m.summary {
		return fmt.Sprintf("%s\n%s", m.renderSummary(), line)
	}
	return line
}

func (m tuiModel) renderHelp() string {
	lines := []string{
		"Key Bindings:",
		" q  quit",
		" w  toggle wrap for mission list",
		" s  toggle auto-scroll",
		" e  spawn enemy (type,lat,lon,alt)",
		" E  edit/remove enemy (id,status|delete)",
		" t  toggle summary footer",
		" m  toggle mission tree",
		" n  toggle enemies section",
		" ?  toggle this help view",
		"",
		"When auto-scroll is disabled:",
		" j/k or up/down    scroll one line",
		" pgdown/pgup       scroll a page",
	}
	return strings.Join(lines, "\n")
}

func (m tuiModel) renderEnemies() string {
	if m.enemyDialog {
		return fmt.Sprintf("Spawn Enemy (type,lat,lon,alt) - Enter to spawn, Esc to cancel: %s", m.enemyInput.View())
	}
	if m.editEnemyDialog {
		return fmt.Sprintf("Edit Enemy (id,status|delete) - Enter to apply, Esc to cancel: %s", m.editEnemyInput.View())
	}
	if len(m.enemies) == 0 {
		return "Enemies: none"
	}
	maxLines := m.maxSectionLines()
	start := len(m.enemies) - maxLines
	if start < 0 {
		start = 0
	}
	var b strings.Builder
	b.WriteString("Enemies:\n")
	for _, e := range m.enemies[start:] {
		b.WriteString(fmt.Sprintf("%s %s status=%s lat=%.5f lon=%.5f alt=%.1f\n", e.ID, e.Type, e.Status, e.Position.Lat, e.Position.Lon, e.Position.Alt))
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
	return enemy.Enemy{ID: uuid.New().String(), Type: typ, Position: telemetry.Position{Lat: lat, Lon: lon, Alt: alt}, Status: enemy.EnemyActive}, nil
}
