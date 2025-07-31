package sim

import (
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

func TestSimulator_DetectsEnemy(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 5}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 1, MovementPattern: "loiter", HomeRegion: "zone"},
		},
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster-test", cfg, writer, dWriter, 1*time.Second)

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat, Lon: drone.Position.Lon, Alt: 0}},
	}

	sim.tick()

	if len(dWriter.Detections) == 0 {
		t.Fatalf("expected enemy detection event")
	}
	det := dWriter.Detections[0]
	if det.ClusterID != "cluster-test" || det.DroneID != drone.ID || det.EnemyID == "" {
		t.Errorf("unexpected detection row: %+v", det)
	}
}

func TestSimulator_NoDetectionOutsideRange(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 5}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 1, MovementPattern: "loiter", HomeRegion: "zone"},
		},
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster-test", cfg, writer, dWriter, 1*time.Second)

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-far", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat + 0.02, Lon: drone.Position.Lon + 0.02, Alt: 0}},
	}

	sim.tick()

	if len(dWriter.Detections) != 0 {
		t.Fatalf("expected no detections, got %d", len(dWriter.Detections))
	}
}

func TestSimulator_NoPanicWithNilDetectionWriter(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 5}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 1, MovementPattern: "loiter", HomeRegion: "zone"},
		},
	}
	writer := &MockWriter{}
	sim := NewSimulator("cluster-test", cfg, writer, nil, 1*time.Second)

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat, Lon: drone.Position.Lon, Alt: 0}},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("tick panicked with nil detection writer: %v", r)
		}
	}()

	sim.tick()
}
