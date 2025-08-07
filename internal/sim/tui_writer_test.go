package sim

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"droneops-sim/internal/config"
	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

type fakeProgram struct{ msgs []tea.Msg }

func (f *fakeProgram) Send(msg tea.Msg) { f.msgs = append(f.msgs, msg) }

func TestTUIWriterMessages(t *testing.T) {
	p := &fakeProgram{}
	w := &TUIWriter{program: p, missionColors: map[string]string{}}
	tRow := telemetry.TelemetryRow{ClusterID: "c", DroneID: "d", MissionID: "m", Timestamp: time.Unix(0, 0).UTC()}
	if err := w.Write(tRow); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, ok := p.msgs[0].(logMsg); !ok {
		t.Fatalf("expected logMsg, got %T", p.msgs[0])
	}
	if _, ok := p.msgs[1].(telemetryMsg); !ok {
		t.Fatalf("expected telemetryMsg, got %T", p.msgs[1])
	}
	st := telemetry.SimulationStateRow{MessagesSent: 1}
	if err := w.WriteState(st); err != nil {
		t.Fatalf("state: %v", err)
	}
	if _, ok := p.msgs[2].(stateMsg); !ok {
		t.Fatalf("expected stateMsg, got %T", p.msgs[2])
	}
	w.SetAdminStatus(true)
	if _, ok := p.msgs[3].(adminMsg); !ok {
		t.Fatalf("expected adminMsg, got %T", p.msgs[3])
	}
	d := enemy.DetectionRow{DroneID: "d", EnemyID: "e", Timestamp: time.Unix(0, 0).UTC()}
	if err := w.WriteDetection(d); err != nil {
		t.Fatalf("detect: %v", err)
	}
	if dm, ok := p.msgs[4].(detectionMsg); !ok {
		t.Fatalf("expected detectionMsg for detection")
	} else if dm.row.DroneID != d.DroneID {
		t.Fatalf("unexpected row in detectionMsg: %+v", dm.row)
	}
	e := telemetry.SwarmEventRow{EventType: "test", Timestamp: time.Unix(0, 0).UTC()}
	if err := w.WriteSwarmEvent(e); err != nil {
		t.Fatalf("swarm: %v", err)
	}
	if _, ok := p.msgs[5].(swarmMsg); !ok {
		t.Fatalf("expected swarmMsg for swarm event")
	}
}

func TestDetectionSwarmAndStateColors(t *testing.T) {
	p := &fakeProgram{}
	w := &TUIWriter{program: p, missionColors: map[string]string{}}
	d := enemy.DetectionRow{
		DroneID: "d1", EnemyID: "e1", EnemyType: "vehicle", Lat: 1, Lon: 2, Alt: 3, Confidence: 0.9, Timestamp: time.Unix(0, 0).UTC(),
	}
	if err := w.WriteDetection(d); err != nil {
		t.Fatalf("write detection: %v", err)
	}
	dm := p.msgs[0].(detectionMsg)
	if !strings.Contains(dm.line, fmt.Sprintf("%sdrone=%s%s", colorWhite(), d.DroneID, colorReset)) {
		t.Fatalf("expected colored drone field: %q", dm.line)
	}
	if !strings.Contains(dm.line, fmt.Sprintf("%slat=%.5f%s", colorGreen, d.Lat, colorReset)) {
		t.Fatalf("expected colored lat field: %q", dm.line)
	}

	p.msgs = nil
	e := telemetry.SwarmEventRow{EventType: "join", DroneIDs: []string{"d1", "d2"}, EnemyID: "e1", Timestamp: time.Unix(0, 0).UTC()}
	if err := w.WriteSwarmEvent(e); err != nil {
		t.Fatalf("write swarm: %v", err)
	}
	sm := p.msgs[0].(swarmMsg)
	if !strings.Contains(sm.line, fmt.Sprintf("%stype=%s%s", colorBlue, e.EventType, colorReset)) {
		t.Fatalf("expected colored type field: %q", sm.line)
	}
	if !strings.Contains(sm.line, fmt.Sprintf("%senemy=%s%s", colorMagenta, e.EnemyID, colorReset)) {
		t.Fatalf("expected colored enemy field: %q", sm.line)
	}

	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	m.state = telemetry.SimulationStateRow{CommunicationLoss: 0.5, MessagesSent: 2, SensorNoise: 0.3, WeatherImpact: 0.4, ChaosMode: true}
	bottom := m.renderBottom()
	if !strings.Contains(bottom, fmt.Sprintf("%scomm_loss=%.2f%s", colorYellow, m.state.CommunicationLoss, colorReset)) {
		t.Fatalf("expected colored comm_loss: %q", bottom)
	}
	if !strings.Contains(bottom, fmt.Sprintf("%smsgs=%d%s", colorGreen, m.state.MessagesSent, colorReset)) {
		t.Fatalf("expected colored msgs: %q", bottom)
	}
	if !strings.Contains(bottom, fmt.Sprintf("%schaos=%t%s", colorRed, m.state.ChaosMode, colorReset)) {
		t.Fatalf("expected colored chaos: %q", bottom)
	}
}

func TestRenderDetectionsAndSwarmEvents(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 40})
	m = mi.(tuiModel)
	mi, _ = m.Update(detectionMsg{line: "det1", row: enemy.DetectionRow{DroneID: "d1", Timestamp: time.Unix(0, 0).UTC()}})
	m = mi.(tuiModel)
	mi, _ = m.Update(swarmMsg{line: "sw1"})
	m = mi.(tuiModel)
	if strings.TrimSpace(m.detVP.View()) != "det1" {
		t.Fatalf("unexpected detections view: %q", m.detVP.View())
	}
	if strings.TrimSpace(m.swarmVP.View()) != "sw1" {
		t.Fatalf("unexpected swarm view: %q", m.swarmVP.View())
	}
	view := m.View()
	detIdx := strings.Index(view, "Detections:")
	swarmIdx := strings.Index(view, "Swarm Events:")
	enemyIdx := strings.Index(view, "Enemies:")
	if detIdx == -1 || swarmIdx == -1 || enemyIdx == -1 {
		t.Fatalf("missing sections in view")
	}
	if !(detIdx < swarmIdx && swarmIdx < enemyIdx) {
		t.Fatalf("sections out of order")
	}
}

func TestTelemetrySectionsCapped(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = mi.(tuiModel)
	for i := 0; i < 50; i++ {
		mi, _ = m.Update(detectionMsg{line: fmt.Sprintf("det%d", i), row: enemy.DetectionRow{DroneID: "d", Timestamp: time.Unix(0, 0).UTC()}})
		m = mi.(tuiModel)
	}
	if m.detVP.Height != m.maxSectionLines() {
		t.Fatalf("unexpected det height: %d != %d", m.detVP.Height, m.maxSectionLines())
	}
	if m.detVP.YOffset != len(m.detLogs)-m.detVP.Height {
		t.Fatalf("detections not scrolled to bottom")
	}
	for i := 0; i < 50; i++ {
		mi, _ = m.Update(swarmMsg{line: fmt.Sprintf("sw%d", i)})
		m = mi.(tuiModel)
	}
	if m.swarmVP.Height != m.maxSectionLines() {
		t.Fatalf("unexpected swarm height: %d != %d", m.swarmVP.Height, m.maxSectionLines())
	}
	if m.swarmVP.YOffset != len(m.swarmLogs)-m.swarmVP.Height {
		t.Fatalf("swarm events not scrolled to bottom")
	}
}

func TestRenderMapAddsContext(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = mi.(tuiModel)
	m.dronePositions["d1"] = telemetry.Position{Lat: 10, Lon: 20}
	m.droneHeadings["d1"] = 0
	out := m.renderMap()
	if !strings.Contains(out, "N↑") {
		t.Fatalf("expected north indicator in map: %q", out)
	}
	if !strings.Contains(out, "Scale:") {
		t.Fatalf("expected scale bar in map: %q", out)
	}
	lines := strings.Split(out, "\n")
	hasGrid := false
	for _, line := range lines[2:] {
		inner := strings.Trim(line, "│")
		if strings.HasPrefix(inner, "Scale:") {
			break
		}
		if strings.ContainsAny(inner, "│─┼") {
			hasGrid = true
			break
		}
	}
	if !hasGrid {
		t.Fatalf("expected gridlines in map: %q", out)
	}
}

func TestRenderMapShowsDetectionAndTrails(t *testing.T) {
	cfg := &config.SimulationConfig{DetectionRadiusM: 1000}
	m := newTUIModel(cfg, nil)
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = mi.(tuiModel)
	m.dronePositions["d1"] = telemetry.Position{Lat: 10, Lon: 20}
	m.droneHeadings["d1"] = 0
	m.droneTrails["d1"] = []telemetry.Position{{Lat: 10.001, Lon: 20.001}}
	m.mapShowDetection = true
	m.mapShowTrails = true
	m.initMapViewport()
	out := m.renderMap()
	if strings.Count(out, "*") < 2 {
		t.Fatalf("expected detection radius in map: %q", out)
	}
	if strings.Count(out, "·") < 2 {
		t.Fatalf("expected trail in map: %q", out)
	}
}

func TestHeadingIcon(t *testing.T) {
	cases := []struct {
		h    float64
		icon string
	}{
		{0, "↑"},
		{45, "↗"},
		{90, "→"},
		{135, "↘"},
		{180, "↓"},
		{225, "↙"},
		{270, "←"},
		{315, "↖"},
	}
	for _, tt := range cases {
		if got := headingIcon(tt.h); got != tt.icon {
			t.Fatalf("heading %v: expected %q, got %q", tt.h, tt.icon, got)
		}
	}
}

func TestAltitudeIcon(t *testing.T) {
	below := altitudeIcon(0, highAltThreshold-1)
	if below != "↑" {
		t.Fatalf("expected low altitude icon ↑, got %q", below)
	}
	cases := []struct {
		h    float64
		icon string
	}{
		{0, "⬆"},
		{45, "⬈"},
		{90, "➡"},
	}
	for _, tt := range cases {
		if got := altitudeIcon(tt.h, highAltThreshold); got != tt.icon {
			t.Fatalf("heading %v: expected %q, got %q", tt.h, tt.icon, got)
		}
	}
}

func TestRenderMapLegendExpanded(t *testing.T) {
	cfg := &config.SimulationConfig{
		Missions: []config.Mission{{ID: "m1", Name: "Alpha"}},
	}
	m := newTUIModel(cfg, map[string]string{"m1": colorGreen})
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = mi.(tuiModel)
	out := m.renderMap()
	if !strings.Contains(out, "m1(Alpha)") {
		t.Fatalf("expected mission name in legend: %q", out)
	}
	if !strings.Contains(out, "active") {
		t.Fatalf("expected active enemy legend: %q", out)
	}
	if !strings.Contains(out, "neutralized") {
		t.Fatalf("expected neutralized enemy legend: %q", out)
	}
	if !strings.Contains(out, "detection") {
		t.Fatalf("expected detection legend: %q", out)
	}
}

func TestWrapToggle(t *testing.T) {
	cfg := &config.SimulationConfig{
		Missions: []config.Mission{{ID: "m1", Name: "M1", Description: "alpha beta gamma delta epsilon zeta"}},
	}
	m := newTUIModel(cfg, map[string]string{"m1": colorBlue})
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 20})
	m = mi.(tuiModel)
	long := "one two three four five six"
	mi, _ = m.Update(logMsg{line: long})
	m = mi.(tuiModel)
	lines := strings.Split(m.vp.View(), "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[1]) != "" {
		t.Fatalf("expected single line before wrap")
	}
	before := m.header
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = mi.(tuiModel)
	if !m.wrap {
		t.Fatalf("wrap not toggled")
	}
	lines = strings.Split(m.vp.View(), "\n")
	if m.vp.View() == long {
		t.Fatalf("expected wrapped content on second line")
	}
	if strings.Count(m.header, "\n") <= strings.Count(before, "\n") {
		t.Fatalf("expected mission description to wrap")
	}
}

func TestScrollToggle(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 20})
	m = mi.(tuiModel)
	m.vp.Height = 1
	m.detVP.Height = 1
	m.swarmVP.Height = 1
	mi, _ = m.Update(logMsg{line: "l1"})
	m = mi.(tuiModel)
	mi, _ = m.Update(logMsg{line: "l2"})
	m = mi.(tuiModel)
	mi, _ = m.Update(detectionMsg{line: "d1", row: enemy.DetectionRow{DroneID: "d1", Timestamp: time.Unix(0, 0).UTC()}})
	m = mi.(tuiModel)
	mi, _ = m.Update(detectionMsg{line: "d2", row: enemy.DetectionRow{DroneID: "d2", Timestamp: time.Unix(0, 0).UTC()}})
	m = mi.(tuiModel)
	mi, _ = m.Update(swarmMsg{line: "s1"})
	m = mi.(tuiModel)
	mi, _ = m.Update(swarmMsg{line: "s2"})
	m = mi.(tuiModel)
	if m.vp.YOffset != len(m.logs)-m.vp.Height || m.detVP.YOffset != len(m.detLogs)-m.detVP.Height || m.swarmVP.YOffset != len(m.swarmLogs)-m.swarmVP.Height {
		t.Fatalf("expected initial offsets at bottom")
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = mi.(tuiModel)
	if m.autoscroll {
		t.Fatalf("autoscroll should be off")
	}
	offLog, offDet, offSw := m.vp.YOffset, m.detVP.YOffset, m.swarmVP.YOffset
	mi, _ = m.Update(logMsg{line: "l3"})
	m = mi.(tuiModel)
	mi, _ = m.Update(detectionMsg{line: "d3", row: enemy.DetectionRow{DroneID: "d3", Timestamp: time.Unix(0, 0).UTC()}})
	m = mi.(tuiModel)
	mi, _ = m.Update(swarmMsg{line: "s3"})
	m = mi.(tuiModel)
	if m.vp.YOffset != offLog || m.detVP.YOffset != offDet || m.swarmVP.YOffset != offSw {
		t.Fatalf("offsets changed with autoscroll off")
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = mi.(tuiModel)
	if !m.autoscroll {
		t.Fatalf("autoscroll should be on")
	}
	if m.vp.YOffset != len(m.logs)-m.vp.Height || m.detVP.YOffset != len(m.detLogs)-m.detVP.Height || m.swarmVP.YOffset != len(m.swarmLogs)-m.swarmVP.Height {
		t.Fatalf("expected offsets at bottom when autoscroll on")
	}
}

func TestEnemySpawn(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	m.spawn = func(enemy.Enemy) {}
	// provide last known drone position
	mi, _ := m.Update(telemetryMsg{telemetry.TelemetryRow{Lat: 1, Lon: 2, Alt: 3}})
	m = mi.(tuiModel)
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = mi.(tuiModel)
	if !m.enemyDialog {
		t.Fatalf("expected enemy dialog to open")
	}
	expected := fmt.Sprintf("vehicle,%.5f,%.5f,%.1f", 1+enemyOffset, 2+enemyOffset, 3.0)
	if m.enemyInput.Value() != expected {
		t.Fatalf("expected default input %q, got %q", expected, m.enemyInput.Value())
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mi.(tuiModel)
	if len(m.enemies) != 1 {
		t.Fatalf("expected enemy added")
	}
	en := m.enemies[0]
	if en.Type != enemy.EnemyVehicle || en.Position.Lat != 1+enemyOffset || en.Position.Lon != 2+enemyOffset || en.Position.Alt != 3 {
		t.Fatalf("unexpected enemy spawned: %+v", en)
	}
}

func TestEnemySpawnFallback(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	m.spawn = func(enemy.Enemy) {}
	mi, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = mi.(tuiModel)
	if m.enemyInput.Value() != fallbackEnemyInput {
		t.Fatalf("expected fallback input %q, got %q", fallbackEnemyInput, m.enemyInput.Value())
	}
}

func TestEnemySpawnNonBlocking(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	start := make(chan struct{})
	block := make(chan struct{})
	m.spawn = func(enemy.Enemy) {
		close(start)
		<-block
	}
	m.enemyDialog = true
	m.enemyInput.SetValue(fallbackEnemyInput)
	done := make(chan struct{})
	go func() {
		mi, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = mi.(tuiModel)
		close(done)
	}()
	<-start
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("update blocked when spawn callback is slow")
	}
	if len(m.enemies) != 1 {
		t.Fatalf("expected enemy added")
	}
	if m.enemyDialog {
		t.Fatalf("expected enemy dialog closed")
	}
	close(block)
}

func TestEnemySpawnHint(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	mi, _ := m.Update(telemetryMsg{telemetry.TelemetryRow{Lat: 4, Lon: 5, Alt: 6}})
	m = mi.(tuiModel)
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = mi.(tuiModel)
	hint := m.renderEnemies()
	if !strings.Contains(hint, "type,lat,lon,alt") {
		t.Fatalf("expected input format hint, got %q", hint)
	}
	if !strings.Contains(hint, "Enter to spawn") {
		t.Fatalf("expected Enter instruction, got %q", hint)
	}
	if !strings.Contains(hint, "Esc to cancel") {
		t.Fatalf("expected Esc instruction, got %q", hint)
	}
	expected := fmt.Sprintf("vehicle,%.5f,%.5f,%.1f", 4+enemyOffset, 5+enemyOffset, 6.0)
	if !strings.Contains(hint, expected) {
		t.Fatalf("expected default value %q in hint, got %q", expected, hint)
	}
}

func TestSummaryToggle(t *testing.T) {
	cfg := &config.SimulationConfig{
		Missions: []config.Mission{{ID: "m1"}, {ID: "m2"}},
		Fleets:   []config.Fleet{{MissionID: "m1", Count: 2}, {MissionID: "m2", Count: 3}},
	}
	colors := map[string]string{"m1": colorRed, "m2": colorGreen}
	m := newTUIModel(cfg, colors)
	mi, _ := m.Update(telemetryMsg{telemetry.TelemetryRow{DroneID: "d1", MissionID: "m1", Battery: 80}})
	m = mi.(tuiModel)
	mi, _ = m.Update(telemetryMsg{telemetry.TelemetryRow{DroneID: "d2", MissionID: "m2", Battery: 40}})
	m = mi.(tuiModel)
	bottom := m.renderBottom()
	if !strings.Contains(bottom, "SUMMARY") {
		t.Fatalf("summary should be shown by default: %q", bottom)
	}
	if !strings.Contains(bottom, fmt.Sprintf("%sdrones=%d%s", colorGreen, 2, colorReset)) {
		t.Fatalf("missing drone count: %q", bottom)
	}
	if !strings.Contains(bottom, fmt.Sprintf("%savg_batt=%.1f%s", colorCyan, 60.0, colorReset)) {
		t.Fatalf("missing avg battery: %q", bottom)
	}
	if !strings.Contains(bottom, fmt.Sprintf("%senemies=%d%s", colorRed, 0, colorReset)) {
		t.Fatalf("missing enemy count: %q", bottom)
	}
	if !strings.Contains(bottom, fmt.Sprintf("%sdet=%d%s", colorMagenta, 0, colorReset)) {
		t.Fatalf("missing detection count: %q", bottom)
	}
	if !strings.Contains(bottom, fmt.Sprintf("%s%s%s=1/2", colorRed, "m1", colorReset)) {
		t.Fatalf("missing mission m1 progress: %q", bottom)
	}
	if !strings.Contains(bottom, fmt.Sprintf("%s%s%s=1/3", colorGreen, "m2", colorReset)) {
		t.Fatalf("missing mission m2 progress: %q", bottom)
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = mi.(tuiModel)
	bottom = m.renderBottom()
	if strings.Contains(bottom, "SUMMARY") {
		t.Fatalf("summary not hidden after toggle: %q", bottom)
	}
}

func TestHelpHintInFooter(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, map[string]string{})
	bottom := m.renderBottom()
	if !strings.Contains(bottom, "(h)elp") {
		t.Fatalf("missing help hint: %q", bottom)
	}
}

func TestSummaryIncludesEnemyAndDetectionStats(t *testing.T) {
	cfg := &config.SimulationConfig{Missions: []config.Mission{{ID: "m1"}}}
	m := newTUIModel(cfg, map[string]string{"m1": colorRed})
	m.enemies = []enemy.Enemy{{ID: "e1", Status: enemy.EnemyActive}, {ID: "e2", Status: enemy.EnemyNeutralized}}
	mi, _ := m.Update(detectionMsg{line: "det", row: enemy.DetectionRow{DroneID: "d1", Timestamp: time.Unix(0, 0).UTC()}})
	m = mi.(tuiModel)
	bottom := m.renderBottom()
	if !strings.Contains(bottom, fmt.Sprintf("%senemies=%d%s", colorRed, 1, colorReset)) {
		t.Fatalf("missing enemy count: %q", bottom)
	}
	if !strings.Contains(bottom, fmt.Sprintf("%sdet=%d%s", colorMagenta, 1, colorReset)) {
		t.Fatalf("missing detection count: %q", bottom)
	}
	if !strings.Contains(bottom, fmt.Sprintf("%s%s%s=1", colorWhite(), "d1", colorReset)) {
		t.Fatalf("missing per-drone detections: %q", bottom)
	}
	if !strings.Contains(bottom, "trend=[1]") {
		t.Fatalf("missing detection trend: %q", bottom)
	}
}

func TestUpdateViewportHeightClampsToZero(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, map[string]string{})
	m.height = 0
	m.headerHeight = 0
	m.updateViewportHeight()
	if m.vp.Height < 0 {
		t.Fatalf("viewport height should be non-negative, got %d", m.vp.Height)
	}
}

func TestRefreshViewportNoPanicWithZeroHeight(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, map[string]string{})
	m.height = 0
	m.headerHeight = 0
	m.logs = []string{"log line"}
	m.updateViewportHeight()
	m.vp.Width = 10
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("refreshViewport panicked: %v", r)
		}
	}()
	m.refreshViewport()
}

func TestRenderEnemiesShowsStatus(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, map[string]string{})
	m.enemies = []enemy.Enemy{{ID: "e1", Type: enemy.EnemyVehicle, Position: telemetry.Position{}, Status: enemy.EnemyActive}}
	out := m.renderEnemies()
	if !strings.Contains(out, "status=active") {
		t.Fatalf("expected status, got %q", out)
	}
}

func TestEnemyEditAndRemove(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, map[string]string{})
	m.enemies = []enemy.Enemy{{ID: "e1", Type: enemy.EnemyVehicle, Position: telemetry.Position{}, Status: enemy.EnemyActive}}
	mi, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'E'}})
	m = mi.(tuiModel)
	if !m.editEnemyDialog {
		t.Fatalf("expected edit dialog")
	}
	m.editEnemyInput.SetValue("e1,neutralized")
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mi.(tuiModel)
	if m.enemies[0].Status != enemy.EnemyNeutralized {
		t.Fatalf("status not updated: %+v", m.enemies[0])
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'E'}})
	m = mi.(tuiModel)
	m.editEnemyInput.SetValue("e1,delete")
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mi.(tuiModel)
	if len(m.enemies) != 0 {
		t.Fatalf("enemy not removed")
	}
}

func TestHelpToggle(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, map[string]string{})
	mi, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mi.(tuiModel)
	if !m.help {
		t.Fatalf("help not enabled")
	}
	view := m.View()
	if !strings.Contains(view, "Key Bindings:") {
		t.Fatalf("help view missing: %q", view)
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mi.(tuiModel)
	if m.help {
		t.Fatalf("help not toggled off")
	}
}

func TestToggleSections(t *testing.T) {
	cfg := &config.SimulationConfig{Missions: []config.Mission{{ID: "m1", Name: "M1"}}}
	m := newTUIModel(cfg, map[string]string{"m1": colorBlue})
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	m = mi.(tuiModel)
	if !strings.Contains(m.renderHeader(), "Missions") {
		t.Fatalf("expected missions in header")
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = mi.(tuiModel)
	if strings.Contains(m.renderHeader(), "Missions") {
		t.Fatalf("missions not hidden")
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = mi.(tuiModel)
	if strings.Contains(m.View(), "Enemies:") {
		t.Fatalf("enemies section not hidden")
	}
	// toggle enemies back on
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = mi.(tuiModel)
	if !strings.Contains(m.View(), "Enemies:") {
		t.Fatalf("enemies section not shown")
	}
}

func TestMapViewRendering(t *testing.T) {
	cfg := &config.SimulationConfig{Missions: []config.Mission{{ID: "m1"}}}
	m := newTUIModel(cfg, map[string]string{"m1": colorGreen})
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 10})
	m = mi.(tuiModel)
	row := telemetry.TelemetryRow{DroneID: "d1", MissionID: "m1", Lat: 0, Lon: 0, HeadingDeg: 0, Alt: 50, Battery: 80}
	mi, _ = m.Update(telemetryMsg{TelemetryRow: row})
	m = mi.(tuiModel)
	m.enemies = []enemy.Enemy{{ID: "e1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: 0, Lon: 0.1}, Status: enemy.EnemyActive}}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = mi.(tuiModel)
	if !m.showMap {
		t.Fatalf("map view not enabled")
	}
	view := m.View()
	if !strings.Contains(view, bgGreen+colorGreen+"↑"+colorReset) {
		t.Fatalf("missing drone marker: %q", view)
	}
	if !strings.Contains(view, colorRed+"X"+colorReset) {
		t.Fatalf("missing enemy marker: %q", view)
	}
}

func TestEnemyStatusMarkers(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 10})
	m = mi.(tuiModel)
	m.dronePositions["d1"] = telemetry.Position{Lat: 0, Lon: 0}
	m.droneHeadings["d1"] = 0
	m.droneBatteries["d1"] = 100
	m.enemies = []enemy.Enemy{
		{ID: "e1", Position: telemetry.Position{Lat: 0, Lon: 0.1}, Status: enemy.EnemyActive},
		{ID: "e2", Position: telemetry.Position{Lat: 0.1, Lon: 0}, Status: enemy.EnemyNeutralized},
	}
	out := m.renderMap()
	if !strings.Contains(out, colorRed+"X"+colorReset) {
		t.Fatalf("active enemy marker missing: %q", out)
	}
	if !strings.Contains(out, colorYellow+"x"+colorReset) {
		t.Fatalf("neutralized enemy marker missing: %q", out)
	}
}

func TestMapLayerToggle(t *testing.T) {
	cfg := &config.SimulationConfig{Missions: []config.Mission{{ID: "m1", Region: config.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}}}}
	m := newTUIModel(cfg, map[string]string{"m1": colorGreen})
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = mi.(tuiModel)
	row := telemetry.TelemetryRow{DroneID: "d1", MissionID: "m1", Lat: 0, Lon: 0, HeadingDeg: 0, Alt: 50, Battery: 80}
	mi, _ = m.Update(telemetryMsg{TelemetryRow: row})
	m = mi.(tuiModel)
	m.enemies = []enemy.Enemy{{ID: "e1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: 0, Lon: 0.1}, Status: enemy.EnemyActive}}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = mi.(tuiModel)
	out := m.renderMap()
	if !strings.Contains(out, bgGreen+colorGreen+"↑"+colorReset) {
		t.Fatalf("expected drone marker: %q", out)
	}
	if strings.Count(out, colorRed+"X"+colorReset) < 2 {
		t.Fatalf("expected enemy marker: %q", out)
	}
	if !strings.Contains(out, colorGreen+"o"+colorReset) {
		t.Fatalf("expected mission zone: %q", out)
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = mi.(tuiModel)
	if strings.Contains(m.renderMap(), bgGreen+colorGreen+"↑"+colorReset) {
		t.Fatalf("drone layer not toggled off")
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = mi.(tuiModel)
	if strings.Count(m.renderMap(), colorRed+"X"+colorReset) > 1 {
		t.Fatalf("enemy layer not toggled off")
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = mi.(tuiModel)
	if strings.Contains(m.renderMap(), colorGreen+"o"+colorReset) {
		t.Fatalf("mission zone layer not toggled off")
	}
}

func TestMapZoomPan(t *testing.T) {
	cfg := &config.SimulationConfig{Missions: []config.Mission{{ID: "m1", Region: config.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}}}}
	m := newTUIModel(cfg, map[string]string{"m1": colorGreen})
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = mi.(tuiModel)
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = mi.(tuiModel)
	initial := m.renderMap()
	lines := strings.Split(initial, "\n")
	var maxLat1, minLat1, minLon1, maxLon1 float64
	fmt.Sscanf(strings.Trim(lines[1], "│"), "lat %f..%f lon %f..%f", &maxLat1, &minLat1, &minLon1, &maxLon1)
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	m = mi.(tuiModel)
	zoomed := m.renderMap()
	lines = strings.Split(zoomed, "\n")
	var maxLat2, minLat2, minLon2, maxLon2 float64
	fmt.Sscanf(strings.Trim(lines[1], "│"), "lat %f..%f lon %f..%f", &maxLat2, &minLat2, &minLon2, &maxLon2)
	if (maxLat2 - minLat2) >= (maxLat1 - minLat1) {
		t.Fatalf("expected zoom in to reduce lat span")
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = mi.(tuiModel)
	panned := m.renderMap()
	lines = strings.Split(panned, "\n")
	var maxLat3, minLat3, minLon3, maxLon3 float64
	fmt.Sscanf(strings.Trim(lines[1], "│"), "lat %f..%f lon %f..%f", &maxLat3, &minLat3, &minLon3, &maxLon3)
	if minLon3 <= minLon2 {
		t.Fatalf("expected pan right to increase min lon")
	}
}
