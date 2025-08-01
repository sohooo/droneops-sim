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

func TestSimulator_DetectsEnemy(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 5}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 1, MovementPattern: "loiter", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
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
	if drone.FollowTarget == nil {
		t.Errorf("expected drone to receive follow target")
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

func TestSimulator_SwarmFollowHighConfidence(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 5}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster", cfg, writer, dWriter, 1*time.Second)

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat, Lon: drone.Position.Lon, Alt: 0}},
	}

	sim.tick()

	if len(dWriter.Detections) == 0 {
		t.Fatalf("expected enemy detection event")
	}

	found := false
	for _, d := range sim.fleets[0].Drones {
		if d.FollowTarget != nil {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected at least one drone to have follow target")
	}
}

func TestSimulator_SwarmNoFollowBelowConfidence(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 5}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster", cfg, writer, dWriter, 1*time.Second)

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat + 0.007, Lon: drone.Position.Lon, Alt: 0}},
	}

	sim.tick()

	if len(dWriter.Detections) == 0 {
		t.Fatalf("expected enemy detection event")
	}

	for _, d := range sim.fleets[0].Drones {
		if d.FollowTarget != nil {
			t.Fatalf("expected no drone to have follow target due to low confidence")
		}
	}
}
