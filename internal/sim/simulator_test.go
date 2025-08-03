package sim

import (
	"context"
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
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "test", Region: config.Region{Name: "region-1", CenterLat: 48.2, CenterLon: 16.4, RadiusKM: 50}}},
		Fleets: []config.Fleet{
			{Name: "fleet-1", Model: "small-fpv", Count: 3, MovementPattern: "patrol", HomeRegion: "region-1", MissionID: "m1"},
		},
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster-test", cfg, writer, dWriter, 1*time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	// Run one tick manually
	sim.tick(context.Background())

	if len(writer.Rows) != 3 {
		t.Errorf("Expected telemetry for 3 drones, got %d", len(writer.Rows))
	}
	for _, row := range writer.Rows {
		if row.DroneID == "" || row.ClusterID == "" {
			t.Errorf("Telemetry row has missing IDs: %+v", row)
		}
		if row.MissionID != "m1" {
			t.Errorf("expected mission ID m1, got %s", row.MissionID)
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
	sim := NewSimulator("cluster", cfg, writer, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	sim.tick(context.Background())

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
	sim := NewSimulator("cluster", cfg, writer, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	drone := sim.fleets[0].Drones[0]
	start := drone.Battery
	sim.tick(context.Background())

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
	sim := NewSimulator("cluster", cfg, writer, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	sim.tick(context.Background())

	if len(writer.Rows) != 0 {
		t.Fatalf("expected no telemetry due to dropout, got %d rows", len(writer.Rows))
	}
}

func TestUpdateDroneDropout(t *testing.T) {
	rand.Seed(1)
	cfg := &config.SimulationConfig{
		Zones: []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}},
		Fleets: []config.Fleet{
			{Name: "f1", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "zone"},
		},
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	drone := sim.fleets[0].Drones[0]
	drone.DropoutRate = 1
	if _, ok := sim.updateDrone(drone); ok {
		t.Fatalf("expected updateDrone to indicate dropout")
	}
}

func TestInjectChaosAltersBattery(t *testing.T) {
	rand.Seed(1)
	cfg := &config.SimulationConfig{
		Zones: []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}},
		Fleets: []config.Fleet{
			{Name: "f1", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "zone"},
		},
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	drone := sim.fleets[0].Drones[0]
	row := sim.teleGen.GenerateTelemetry(drone)
	before := drone.Battery
	sim.injectChaos(drone, &row)
	if drone.Battery >= before {
		t.Fatalf("expected battery to decrease")
	}
	if row.Battery != drone.Battery {
		t.Fatalf("telemetry row battery mismatch")
	}
}

func TestProcessDetectionsReturnsDetection(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones: []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}},
		Fleets: []config.Fleet{
			{Name: "f1", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, &MockDetectionWriter{}, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	drone := sim.fleets[0].Drones[0]
	en := &enemy.Enemy{ID: "e1", Type: enemy.EnemyDrone, Position: drone.Position}
	sim.enemyEng = &enemy.Engine{Enemies: []*enemy.Enemy{en}}
	dets := sim.processDetections(&sim.fleets[0], drone)
	if len(dets) != 1 {
		t.Fatalf("expected one detection, got %d", len(dets))
	}
	if sim.droneAssignments[drone.ID] != en.ID {
		t.Fatalf("expected drone assigned to enemy")
	}
}

func TestSimulator_DetectsEnemy(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 0.1}},
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 1, MovementPattern: "loiter", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster-test", cfg, writer, dWriter, 1*time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat, Lon: drone.Position.Lon, Alt: 0}},
	}

	sim.tick(context.Background())

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
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 1, MovementPattern: "loiter", HomeRegion: "zone"},
		},
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster-test", cfg, writer, dWriter, 1*time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-far", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat + 0.02, Lon: drone.Position.Lon + 0.02, Alt: 0}},
	}

	sim.tick(context.Background())

	if len(dWriter.Detections) != 0 {
		t.Fatalf("expected no detections, got %d", len(dWriter.Detections))
	}
}

func TestSimulator_NoPanicWithNilDetectionWriter(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 0.1}},
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 1, MovementPattern: "loiter", HomeRegion: "zone"},
		},
	}
	writer := &MockWriter{}
	sim := NewSimulator("cluster-test", cfg, writer, nil, 1*time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat, Lon: drone.Position.Lon, Alt: 0}},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("tick panicked with nil detection writer: %v", r)
		}
	}()

	sim.tick(context.Background())
}

func TestSimulator_EnemyCountConfig(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:      []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets:     []config.Fleet{{Name: "f", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "zone"}},
		EnemyCount: 5,
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
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
	sim := NewSimulator("cluster", cfg, writer, dWriter, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	rand.Seed(1)

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{{ID: "e", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat + 0.002, Lon: drone.Position.Lon, Alt: 0}}}

	sim.tick(context.Background())

	if len(dWriter.Detections) == 0 {
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
	sim := NewSimulator("cluster", cfg, writer, dWriter, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	rand.Seed(1)

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{{ID: "e", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat + 0.006, Lon: drone.Position.Lon, Alt: 0}}}

	sim.tick(context.Background())

	if len(dWriter.Detections) != 0 {
		t.Fatalf("expected no detections outside custom radius, got %d", len(dWriter.Detections))
	}
}

func TestSimulator_SwarmFollowHighConfidence(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 48.0, CenterLon: 16.0, RadiusKM: 0.1}},
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
		SwarmResponses:   map[string]int{"patrol": 1},
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster", cfg, writer, dWriter, 1*time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat, Lon: drone.Position.Lon, Alt: 0}},
	}

	sim.tick(context.Background())

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
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
		SwarmResponses:   map[string]int{"patrol": 1},
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster", cfg, writer, dWriter, 1*time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat + 0.007, Lon: drone.Position.Lon, Alt: 0}},
	}

	sim.tick(context.Background())

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
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
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
	sim := NewSimulator("cluster", cfg, writer, dWriter, 1*time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	drone := sim.fleets[0].Drones[0]
	sim.enemyEng.Enemies = []*enemy.Enemy{
		{ID: "enemy-1", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat, Lon: drone.Position.Lon, Alt: 0}},
	}

	sim.tick(context.Background())

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
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "point-to-point", HomeRegion: "zone"},
		},
		FollowConfidence: 50,
		SwarmResponses:   map[string]int{"point-to-point": 0},
	}
	writer := &MockWriter{}
	dWriter := &MockDetectionWriter{}
	sim := NewSimulator("cluster", cfg, writer, dWriter, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

	drone := sim.fleets[0].Drones[0]
	sim.fleets[0].Drones[1].Position.Lat += 0.05
	sim.enemyEng.Enemies = []*enemy.Enemy{{ID: "e", Type: enemy.EnemyPerson, Position: telemetry.Position{Lat: drone.Position.Lat + 0.001, Lon: drone.Position.Lon}}}

	sim.tick(context.Background())

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

func TestSimulator_PredictiveInterception(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:            []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets:           []config.Fleet{{Name: "f", Model: "small-fpv", Count: 3, MovementPattern: "patrol", HomeRegion: "zone"}},
		FollowConfidence: 50,
		SwarmResponses:   map[string]int{"patrol": 2},
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, &MockDetectionWriter{}, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	fleet := &sim.fleets[0]
	detecting := fleet.Drones[0]
	prev := telemetry.Position{Lat: 0, Lon: 0}
	curr := telemetry.Position{Lat: 0.001, Lon: 0}
	en := &enemy.Enemy{ID: "e1", Type: enemy.EnemyVehicle, Position: curr}
	sim.enemyPrevPositions[en.ID] = prev

	sim.assignFollower(fleet, detecting, en, 100)

	followers := 0
	var t1, t2 *telemetry.Position
	for _, d := range fleet.Drones {
		if d == detecting {
			continue
		}
		if d.FollowTarget != nil {
			followers++
			if t1 == nil {
				t1 = d.FollowTarget
			} else {
				t2 = d.FollowTarget
			}
		}
	}
	if followers != 2 {
		t.Fatalf("expected two drones to receive intercept targets, got %d", followers)
	}
	if t1.Lat == t2.Lat && t1.Lon == t2.Lon {
		t.Errorf("expected followers to flank with different targets")
	}
	if t1.Lat <= en.Position.Lat || t2.Lat <= en.Position.Lat {
		t.Errorf("followers should aim ahead of enemy")
	}
}

func TestSimulator_PatrolResponseCount(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 0}},
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 3, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		SwarmResponses: map[string]int{"patrol": 1},
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

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
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 3, MovementPattern: "loiter", HomeRegion: "zone"},
		},
		SwarmResponses: map[string]int{"loiter": 2},
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

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
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 10}}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 4, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		SwarmResponses:     map[string]int{"patrol": 1},
		MissionCriticality: "high",
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

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
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })

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

func TestSimulator_CommunicationLossPreventsAssignment(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones: []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		SwarmResponses:    map[string]int{"patrol": 1},
		CommunicationLoss: 1.0,
		BandwidthLimit:    10,
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	detecting := sim.fleets[0].Drones[0]
	en := &enemy.Enemy{ID: "e", Type: enemy.EnemyPerson, Position: telemetry.Position{Lat: 0, Lon: 0}}
	sim.assignFollower(&sim.fleets[0], detecting, en, 80)
	for _, d := range sim.fleets[0].Drones {
		if d.FollowTarget != nil {
			t.Fatalf("expected no followers due to comm loss")
		}
	}
}

func TestSimulator_FollowerFailover(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones: []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets: []config.Fleet{
			{Name: "fleet", Model: "small-fpv", Count: 3, MovementPattern: "patrol", HomeRegion: "zone"},
		},
		SwarmResponses:    map[string]int{"patrol": 1},
		CommunicationLoss: 0,
		BandwidthLimit:    10,
	}
	sim := NewSimulator("cluster", cfg, &MockWriter{}, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	detecting := sim.fleets[0].Drones[0]
	en := &enemy.Enemy{ID: "e", Type: enemy.EnemyPerson, Position: telemetry.Position{Lat: 0, Lon: 0}}
	sim.assignFollower(&sim.fleets[0], detecting, en, 80)
	var follower *telemetry.Drone
	for _, d := range sim.fleets[0].Drones {
		if d.FollowTarget != nil && d != detecting {
			follower = d
		}
	}
	if follower == nil {
		t.Fatalf("expected a follower to be assigned")
	}
	follower.Status = telemetry.StatusFailure
	sim.reassignFollowers()
	if follower.FollowTarget != nil {
		t.Errorf("failed follower should have no target")
	}
	var replacement *telemetry.Drone
	for _, d := range sim.fleets[0].Drones {
		if d.FollowTarget != nil && d != follower {
			replacement = d
		}
	}
	if replacement == nil {
		t.Fatalf("expected replacement follower to be assigned")
	}
}

func TestCleanupFollowers_RemovesInvalid(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:  []config.Region{{Name: "z", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets: []config.Fleet{{Name: "f", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "z"}},
	}
	sim := NewSimulator("c", cfg, nil, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	d := sim.fleets[0].Drones[0]
	pos := telemetry.Position{Lat: 0, Lon: 0}
	d.FollowTarget = &pos
	sim.droneAssignments[d.ID] = "e"
	sim.enemyFollowers["e"] = []string{d.ID}
	d.Status = telemetry.StatusFailure
	active := sim.cleanupFollowers("e", sim.enemyFollowers["e"])
	if len(active) != 0 {
		t.Fatalf("expected no active followers, got %d", len(active))
	}
	if d.FollowTarget != nil {
		t.Fatalf("expected FollowTarget cleared")
	}
	if _, ok := sim.droneAssignments[d.ID]; ok {
		t.Fatalf("expected assignment removed")
	}
}

func TestApplyAssignments_SetsTargets(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:  []config.Region{{Name: "z", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets: []config.Fleet{{Name: "f", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "z"}},
	}
	sim := NewSimulator("c", cfg, nil, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	d1 := sim.fleets[0].Drones[0]
	d2 := sim.fleets[0].Drones[1]
	en := &enemy.Enemy{ID: "e", Position: telemetry.Position{Lat: 1, Lon: 1}}
	cands := []*telemetry.Drone{d1, d2}
	sim.applyAssignments("e", en, cands)
	if d1.FollowTarget == nil || d2.FollowTarget == nil {
		t.Fatalf("expected targets assigned")
	}
	if sim.droneAssignments[d1.ID] != "e" || sim.droneAssignments[d2.ID] != "e" {
		t.Fatalf("expected assignments recorded")
	}
	if len(sim.enemyFollowers["e"]) != 2 {
		t.Fatalf("expected follower records")
	}
}

func TestSelectCandidates_ReservesAssignments(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:             []config.Region{{Name: "z", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets:            []config.Fleet{{Name: "f", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "z"}},
		CommunicationLoss: 0,
		BandwidthLimit:    10,
	}
	sim := NewSimulator("c", cfg, nil, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	cands := sim.selectCandidates(1)
	if len(cands) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(cands))
	}
	if _, ok := sim.droneAssignments[cands[0].ID]; !ok {
		t.Fatalf("expected assignment reserved")
	}
}

func TestObserverTools(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:  []config.Region{{Name: "r1", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets: []config.Fleet{{Name: "f1", Model: "small-fpv", Count: 1}},
	}
	sim := NewSimulator("cluster", cfg, nil, nil, 1, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	sim.ObserverInjectCommand("test")
	events := sim.ObserverEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev, ok := sim.ObserverStep(0)
	if !ok || ev.Type != "command" {
		t.Fatalf("unexpected step result: %+v", ev)
	}
	droneID := sim.TelemetrySnapshot()[0].DroneID
	sim.ObserverSetPerspective(droneID)
	if sim.ObserverPerspective() != droneID {
		t.Fatalf("expected perspective %s, got %s", droneID, sim.ObserverPerspective())
	}
}

func TestSimulator_ToggleChaos(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:  []config.Region{{Name: "z", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets: []config.Fleet{{Name: "f", Model: "small-fpv", Count: 1}},
	}
	sim := NewSimulator("c", cfg, nil, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	if sim.Chaos() {
		t.Fatalf("expected chaos disabled")
	}
	if !sim.ToggleChaos() || !sim.Chaos() {
		t.Fatalf("chaos should be enabled after toggle")
	}
	if sim.ToggleChaos() || sim.Chaos() {
		t.Fatalf("chaos should be disabled after second toggle")
	}
}

func TestAssignFollowerDeterministic(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:          []config.Region{{Name: "zone", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets:         []config.Fleet{{Name: "fleet", Model: "small-fpv", Count: 2, MovementPattern: "patrol", HomeRegion: "zone"}},
		SwarmResponses: map[string]int{"patrol": 1},
	}
	r1 := rand.New(rand.NewSource(1))
	r2 := rand.New(rand.NewSource(1))
	sim1 := NewSimulator("c1", cfg, &MockWriter{}, nil, time.Second, r1, func() time.Time { return time.Unix(0, 0).UTC() })
	sim2 := NewSimulator("c2", cfg, &MockWriter{}, nil, time.Second, r2, func() time.Time { return time.Unix(0, 0).UTC() })
	detecting1 := sim1.fleets[0].Drones[0]
	detecting2 := sim2.fleets[0].Drones[0]
	en := &enemy.Enemy{ID: "e", Type: enemy.EnemyPerson, Position: telemetry.Position{Lat: 0, Lon: 0}}
	sim1.assignFollower(&sim1.fleets[0], detecting1, en, 80)
	sim2.assignFollower(&sim2.fleets[0], detecting2, en, 80)
	var f1Idx, f2Idx int = -1, -1
	for i, d := range sim1.fleets[0].Drones {
		if d != detecting1 && d.FollowTarget != nil {
			f1Idx = i
		}
	}
	for i, d := range sim2.fleets[0].Drones {
		if d != detecting2 && d.FollowTarget != nil {
			f2Idx = i
		}
	}
	if f1Idx != f2Idx || f1Idx == -1 {
		t.Fatalf("expected same follower index, got %d and %d", f1Idx, f2Idx)
	}
	t1 := sim1.fleets[0].Drones[f1Idx].FollowTarget
	t2 := sim2.fleets[0].Drones[f2Idx].FollowTarget
	if t1 == nil || t2 == nil || t1.Lat != t2.Lat || t1.Lon != t2.Lon || t1.Alt != t2.Alt {
		t.Fatalf("expected identical follow targets")
	}
}
func TestProcessDetectionsPopulatesFields(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:  []config.Region{{Name: "z", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets: []config.Fleet{{Name: "f", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "z"}},
	}
	sim := NewSimulator("c", cfg, &MockWriter{}, &MockDetectionWriter{}, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	drone := sim.fleets[0].Drones[0]
	en := &enemy.Enemy{ID: "e", Type: enemy.EnemyVehicle, Position: telemetry.Position{Lat: drone.Position.Lat, Lon: drone.Position.Lon + 0.001}}
	sim.enemyEng = &enemy.Engine{Enemies: []*enemy.Enemy{en}}
	sim.enemyPrevPositions = map[string]telemetry.Position{"e": {Lat: drone.Position.Lat, Lon: drone.Position.Lon}}
	dets := sim.processDetections(&sim.fleets[0], drone)
	if len(dets) != 1 {
		t.Fatalf("expected detection")
	}
	det := dets[0]
	if det.DroneLat != drone.Position.Lat || det.DroneLon != drone.Position.Lon {
		t.Fatalf("missing drone coords: %+v", det)
	}
	expDist := distanceMeters(drone.Position.Lat, drone.Position.Lon, en.Position.Lat, en.Position.Lon)
	if math.Abs(det.DistanceM-expDist) > 0.1 {
		t.Fatalf("distance mismatch: got %f want %f", det.DistanceM, expDist)
	}
	expBearing := bearingDegrees(drone.Position.Lat, drone.Position.Lon, en.Position.Lat, en.Position.Lon)
	if math.Abs(det.BearingDeg-expBearing) > 0.1 {
		t.Fatalf("bearing mismatch: got %f want %f", det.BearingDeg, expBearing)
	}
	expVel := distanceMeters(drone.Position.Lat, drone.Position.Lon, en.Position.Lat, en.Position.Lon) / sim.tickInterval.Seconds()
	if math.Abs(det.EnemyVelMS-expVel) > 0.1 {
		t.Fatalf("velocity mismatch: got %f want %f", det.EnemyVelMS, expVel)
	}
}
