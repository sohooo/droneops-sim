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
	if _, ok := p.msgs[4].(detectionMsg); !ok {
		t.Fatalf("expected detectionMsg for detection")
	}
	e := telemetry.SwarmEventRow{EventType: "test", Timestamp: time.Unix(0, 0).UTC()}
	if err := w.WriteSwarmEvent(e); err != nil {
		t.Fatalf("swarm: %v", err)
	}
	if _, ok := p.msgs[5].(swarmMsg); !ok {
		t.Fatalf("expected swarmMsg for swarm event")
	}
}

func TestRenderDetectionsAndSwarmEvents(t *testing.T) {
	cfg := &config.SimulationConfig{}
	m := newTUIModel(cfg, nil)
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 40})
	m = mi.(tuiModel)
	mi, _ = m.Update(detectionMsg{line: "det1"})
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
		mi, _ = m.Update(detectionMsg{line: fmt.Sprintf("det%d", i)})
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
	mi, _ = m.Update(detectionMsg{line: "d1"})
	m = mi.(tuiModel)
	mi, _ = m.Update(detectionMsg{line: "d2"})
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
	mi, _ = m.Update(detectionMsg{line: "d3"})
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
