package sim

import (
	"testing"
	"time"

	"droneops-sim/internal/config"
	"droneops-sim/internal/telemetry"
)

// MockWriter collects telemetry rows for validation
type MockWriter struct {
	Rows []telemetry.TelemetryRow
}

func (w *MockWriter) Write(row telemetry.TelemetryRow) error {
	w.Rows = append(w.Rows, row)
	return nil
}

func TestSimulator_TickGeneratesTelemetry(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "region-1", CenterLat: 48.2, CenterLon: 16.4, RadiusKM: 50}},
		Missions: []config.Mission{{Name: "m1", Zone: "region-1", Description: "test"}},
		Fleets: []config.Fleet{
			{Name: "fleet-1", Model: "small-fpv", Count: 3, MovementPattern: "patrol", HomeRegion: "region-1"},
		},
	}
	writer := &MockWriter{}
	sim := NewSimulator("cluster-test", cfg, writer, 1*time.Second)

	// Run one tick manually
	sim.tick()

	if len(writer.Rows) != 3 {
		t.Errorf("Expected telemetry for 3 drones, got %d", len(writer.Rows))
	}
	for _, row := range writer.Rows {
		if row.DroneID == "" || row.ClusterID == "" {
			t.Errorf("Telemetry row has missing IDs: %+v", row)
		}
	}
}

func TestSimulator_Dropout(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones: []config.Region{{Name: "region-1", CenterLat: 0, CenterLon: 0, RadiusKM: 10}},
		Fleets: []config.Fleet{
			{Name: "fleet-1", Model: "small-fpv", Count: 1, Behavior: config.Behavior{DropoutRate: 1}},
		},
	}
	writer := &MockWriter{}
	sim := NewSimulator("cluster", cfg, writer, time.Second)
	sim.tick()
	if len(writer.Rows) != 0 {
		t.Errorf("expected no rows due to dropout, got %d", len(writer.Rows))
	}
}
