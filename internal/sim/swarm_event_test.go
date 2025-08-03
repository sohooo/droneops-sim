package sim

import (
	"math/rand"
	"testing"
	"time"

	"droneops-sim/internal/config"
	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

type mockSwarmWriter struct {
	events []telemetry.SwarmEventRow
}

func (m *mockSwarmWriter) Write(row telemetry.TelemetryRow) error { return nil }
func (m *mockSwarmWriter) WriteSwarmEvent(e telemetry.SwarmEventRow) error {
	m.events = append(m.events, e)
	return nil
}

// Ensure mock satisfies interfaces
var _ TelemetryWriter = (*mockSwarmWriter)(nil)
var _ SwarmEventWriter = (*mockSwarmWriter)(nil)

func TestSimulator_AssignFollowerEmitsEvents(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "region-1", CenterLat: 0, CenterLon: 0, RadiusKM: 10}},
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "region-1", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
		Fleets:   []config.Fleet{{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "region-1", MissionID: "m1"}},
	}
	writer := &mockSwarmWriter{}
	sim := NewSimulator("c1", cfg, writer, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	fleet := &sim.fleets[0]
	en := &enemy.Enemy{ID: "e1", Position: telemetry.Position{}}

	sim.assignFollower(fleet, fleet.Drones[0], en, 95)

	var hasAssign, hasFormation bool
	for _, e := range writer.events {
		switch e.EventType {
		case telemetry.SwarmEventAssignment:
			hasAssign = true
		case telemetry.SwarmEventFormationChange:
			hasFormation = true
		}
	}
	if !hasAssign || !hasFormation {
		t.Fatalf("expected assignment and formation events, got %#v", writer.events)
	}
}
