package sim

import (
	"math"
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
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 0.1}},
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
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 0.1}},
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
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 0.1}},
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

func TestSimulator_EnemyCountConfig(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:      []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets:     []config.Fleet{{Name: "f", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "zone"}},
		EnemyCount: 5,
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second)
	if len(sim.enemyEng.Enemies) != 5 {
		t.Fatalf("expected 5 enemies, got %d", len(sim.enemyEng.Enemies))
	}
}

func TestSimulator_CustomDetectionRadius(t *testing.T) {
	rand.Seed(1)
	cfg := &config.SimulationConfig{
		Zones:            []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets:           []config.Fleet{{Name: "f", Model: "small-fpv", Count: 1, MovementPattern: "loiter", HomeRegion: "zone"}},
		DetectionRadiusM: 500,
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster", cfg, writer, dWriter, time.Second)
	rand.Seed(1)

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{{ID: "e", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat + 0.002, Lon: drone.Position.Lon, Alt: 0}}}

	sim.tick()

	if len(dWriter.Detections) != 1 {
		t.Fatalf("expected detection within custom radius, got %d", len(dWriter.Detections))
	}
}

func TestSimulator_NoDetectionOutsideCustomRadius(t *testing.T) {
	rand.Seed(1)
	cfg := &config.SimulationConfig{
		Zones:            []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets:           []config.Fleet{{Name: "f", Model: "small-fpv", Count: 1, MovementPattern: "loiter", HomeRegion: "zone"}},
		DetectionRadiusM: 500,
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster", cfg, writer, dWriter, time.Second)
	rand.Seed(1)

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{{ID: "e", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat + 0.006, Lon: drone.Position.Lon, Alt: 0}}}

	sim.tick()

	if len(dWriter.Detections) != 0 {
		t.Fatalf("expected no detections outside custom radius, got %d", len(dWriter.Detections))
	}
}

func TestSimulator_SwarmFollowHighConfidence(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 0.1}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
		SwarmResponses:   map[string]int{"patrol": 1},
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
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 0.1}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
		SwarmResponses:   map[string]int{"patrol": 1},
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

func TestSimulator_DetectionFactorsReduceConfidence(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 0.1}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 1, MovementPattern: "loiter", HomeRegion: "zone"},
		},
		DetectionRadiusM: 1000,
		FollowConfidence: 75,
		SensorNoise:      0,
		TerrainOcclusion: 0.3,
		WeatherImpact:    0.4,
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
	det := dWriter.Detections[0]
	dist := distanceMeters(drone.Position.Lat, drone.Position.Lon, det.Lat, det.Lon)
	expected := 100 * (1 - dist/cfg.DetectionRadiusM) * (1 - cfg.TerrainOcclusion) * (1 - cfg.WeatherImpact)
	if math.Abs(det.Confidence-expected) > 0.01 {
		t.Errorf("expected confidence %.2f, got %.2f", expected, det.Confidence)
	}
	if drone.FollowTarget != nil {
		t.Fatalf("expected no follow target due to reduced confidence")
	}
}

func TestSimulator_PointToPointDetectorFollows(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 0.1}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "point-to-point", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
		SwarmResponses:   map[string]int{"point-to-point": 0},
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster", cfg, writer, dWriter, time.Second)

	drone := sim.fleets[0].Drones[0]
	sim.fleets[0].Drones[1].Position.Lat += 0.05
	sim.enemyEng.Enemies = []*enemy.Enemy{{ID: "e", Type: enemy.EnemyPerson, Position: telemetry.Position{Lat: drone.Position.Lat + 0.001, Lon: drone.Position.Lon}}}

	sim.tick()

	if drone.FollowTarget == nil {
		t.Fatalf("detecting drone should follow target")
	}
	followers := 0
	for _, d := range sim.fleets[0].Drones {
		if d.FollowTarget != nil {
			followers++
		}
	}
	if followers != 1 {
		t.Fatalf("expected only detecting drone to follow, got %d", followers)
	}
}

func TestSimulator_PatrolResponseCount(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 0}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 3, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		SwarmResponses: map[string]int{"patrol": 1},
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second)

	detecting := sim.fleets[0].Drones[0]
	en := &enemy.Enemy{ID: "e", Type: enemy.EnemyPerson, Position: telemetry.Position{Lat: 0, Lon: 0}}

	sim.assignFollower(&sim.fleets[0], detecting, en, 80)

	followers := 0
	for _, d := range sim.fleets[0].Drones {
		if d.FollowTarget != nil {
			followers++
		}
	}
	if followers != 1 {
		t.Fatalf("expected one drone to follow, got %d", followers)
	}
	if detecting.FollowTarget != nil {
		t.Errorf("detecting drone should remain on patrol")
	}
}

func TestSimulator_LoiterResponseCount(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 0}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 3, MovementPattern: "loiter", HomeRegion: "zone"},
		},
		SwarmResponses: map[string]int{"loiter": 2},
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second)

	detecting := sim.fleets[0].Drones[0]
	en := &enemy.Enemy{ID: "e", Type: enemy.EnemyPerson, Position: telemetry.Position{Lat: 0, Lon: 0}}

	sim.assignFollower(&sim.fleets[0], detecting, en, 80)

	followers := 0
	for _, d := range sim.fleets[0].Drones {
		if d.FollowTarget != nil {
			followers++
		}
	}
	if followers != 2 {
		t.Fatalf("expected two drones to follow, got %d", followers)
	}
	if detecting.FollowTarget != nil {
		t.Errorf("detecting drone should remain on loiter path")
	}
}

func TestSimulator_ThreatAdaptiveFollowers(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 0}},
		Missions: []config.Mission{{Name: "m1", Zone: "zone", Description: ""}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 4, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		SwarmResponses:     map[string]int{"patrol": 1},
		MissionCriticality: "high",
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second)

	detecting := sim.fleets[0].Drones[0]
	en := &enemy.Enemy{ID: "e", Type: enemy.EnemyDrone, Position: telemetry.Position{Lat: 0, Lon: 0}}

	sim.assignFollower(&sim.fleets[0], detecting, en, 95)

	followers := 0
	for _, d := range sim.fleets[0].Drones {
		if d.FollowTarget != nil {
			followers++
		}
	}
	if followers != 3 {
		t.Fatalf("expected 3 drones to follow under high threat, got %d", followers)
	}
}

func TestSimulator_RebalanceFormation(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones: []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 4, MovementPattern: "patrol", HomeRegion: "zone"},
		},
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second)

	// First drone breaks formation to follow an enemy
	en := &enemy.Enemy{ID: "e", Type: enemy.EnemyPerson, Position: telemetry.Position{Lat: 0.001, Lon: 0}}
	sim.fleets[0].Drones[0].FollowTarget = &en.Position

	orig := sim.fleets[0].Drones[1].HomeRegion
	sim.rebalanceFormation(&sim.fleets[0])

	var centers []telemetry.Position
	for i, d := range sim.fleets[0].Drones {
		if i == 0 {
			continue
		}
		if d.FollowTarget != nil {
			t.Fatalf("drone %d should not be following", i)
		}
		centers = append(centers, telemetry.Position{Lat: d.HomeRegion.CenterLat, Lon: d.HomeRegion.CenterLon})
	}
	if len(centers) != 3 {
		t.Fatalf("expected 3 drones to be reassigned, got %d", len(centers))
	}

	region := orig
	radius := region.RadiusKM * 1000 * 0.5
	for i, c := range centers {
		angle := float64(i) / float64(len(centers)) * 2 * math.Pi
		expLat := region.CenterLat + (radius*math.Cos(angle))/111000
		expLon := region.CenterLon + (radius*math.Sin(angle))/(111000*math.Cos(region.CenterLat*math.Pi/180))
		if math.Abs(c.Lat-expLat) > 1e-6 || math.Abs(c.Lon-expLon) > 1e-6 {
			t.Errorf("drone %d center (%.6f, %.6f), expected (%.6f, %.6f)", i, c.Lat, c.Lon, expLat, expLon)
		}
	}
}
