package sim

import (
	"math/rand"
	"testing"
	"time"

	"droneops-sim/internal/config"
	"droneops-sim/internal/enemy"
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

type MockDetectionWriter struct {
	Detections []enemy.DetectionRow
}

func (w *MockDetectionWriter) WriteDetection(d enemy.DetectionRow) error {
	w.Detections = append(w.Detections, d)
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
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster-test", cfg, writer, dWriter, 1*time.Second)

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

func TestSimulator_SensorErrorRate(t *testing.T) {
	rand.Seed(1)
	cfg := &config.SimulationConfig{
		Zones: []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}},
		Fleets: []config.Fleet{
			{Name: "f1", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "zone",
				Behavior: config.Behavior{SensorErrorRate: 1}},
		},
	}
	writer := &MockWriter{}
	sim := NewSimulator("cluster", cfg, writer, nil, time.Second)

	sim.tick()

	if len(writer.Rows) != 1 {
		t.Fatalf("expected 1 telemetry row, got %d", len(writer.Rows))
	}

	row := writer.Rows[0]
	drone := sim.fleets[0].Drones[0]
	if row.Lat == drone.Position.Lat && row.Lon == drone.Position.Lon {
		t.Errorf("expected coordinates to deviate due to sensor error")
	}
}

func TestSimulator_BatteryAnomalyRate(t *testing.T) {
	rand.Seed(1)
	cfg := &config.SimulationConfig{
		Zones: []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}},
		Fleets: []config.Fleet{
			{Name: "f1", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "zone",
				Behavior: config.Behavior{BatteryAnomalyRate: 1}},
		},
	}
	writer := &MockWriter{}
	sim := NewSimulator("cluster", cfg, writer, nil, time.Second)

	drone := sim.fleets[0].Drones[0]
	start := drone.Battery
	sim.tick()

	if len(writer.Rows) != 1 {
		t.Fatalf("expected 1 telemetry row, got %d", len(writer.Rows))
	}

	if start-drone.Battery < 10 {
		t.Errorf("expected battery drop of at least 10, got %.2f", start-drone.Battery)
	}
	if writer.Rows[0].Battery != drone.Battery {
		t.Errorf("row battery should match drone battery")
	}
}

func TestSimulator_DropoutRate(t *testing.T) {
	rand.Seed(1)
	cfg := &config.SimulationConfig{
		Zones: []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}},
		Fleets: []config.Fleet{
			{Name: "f1", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "zone",
				Behavior: config.Behavior{DropoutRate: 1}},
		},
	}
	writer := &MockWriter{}
	sim := NewSimulator("cluster", cfg, writer, nil, time.Second)

	sim.tick()

	if len(writer.Rows) != 0 {
		t.Fatalf("expected no telemetry due to dropout, got %d rows", len(writer.Rows))
	}
}
