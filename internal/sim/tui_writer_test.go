package sim

import (
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
	st := telemetry.SimulationStateRow{MessagesSent: 1}
	if err := w.WriteState(st); err != nil {
		t.Fatalf("state: %v", err)
	}
	if _, ok := p.msgs[1].(stateMsg); !ok {
		t.Fatalf("expected stateMsg, got %T", p.msgs[1])
	}
	w.SetAdminStatus(true)
	if _, ok := p.msgs[2].(adminMsg); !ok {
		t.Fatalf("expected adminMsg, got %T", p.msgs[2])
	}
	d := enemy.DetectionRow{DroneID: "d", EnemyID: "e", Timestamp: time.Unix(0, 0).UTC()}
	if err := w.WriteDetection(d); err != nil {
		t.Fatalf("detect: %v", err)
	}
	if _, ok := p.msgs[3].(logMsg); !ok {
		t.Fatalf("expected logMsg for detection")
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
	if strings.TrimSpace(lines[1]) == "" {
		t.Fatalf("expected wrapped content on second line")
	}
	if strings.Count(m.header, "\n") <= strings.Count(before, "\n") {
		t.Fatalf("expected mission description to wrap")
	}
}
