package telemetry

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestGenerateTelemetry(t *testing.T) {
	fixed := time.Unix(0, 0).UTC()
	gen := NewGenerator("cluster-1", rand.New(rand.NewSource(1)), func() time.Time { return fixed })
	drone := &Drone{
		ID:        "drone-001",
		Model:     "small-fpv",
		MissionID: "m1",
		Position:  Position{Lat: 48.2082, Lon: 16.3738, Alt: 100},
		Battery:   50,
		Status:    StatusOK,
	}

	prev := drone.Position
	row := gen.GenerateTelemetry(drone, prev, time.Second)

	if row.ClusterID != "cluster-1" {
		t.Errorf("expected cluster-1, got %s", row.ClusterID)
	}
	if row.DroneID != "drone-001" {
		t.Errorf("expected drone-001, got %s", row.DroneID)
	}
	if row.MissionID != "m1" {
		t.Errorf("expected mission ID m1, got %s", row.MissionID)
	}
	if row.SyncedFrom != "" || row.SyncedID != "" || !row.SyncedAt.IsZero() {
		t.Errorf("expected unsynced defaults, got %+v", row)
	}
	if !row.Timestamp.Equal(fixed) {
		t.Errorf("unexpected timestamp: %v", row.Timestamp)
	}
	// Check that position changed (movement simulated)
	if row.Lat == 48.2082 && row.Lon == 16.3738 {
		t.Errorf("expected position to change")
	}
	// Battery should decrease
	if row.Battery >= 50 {
		t.Errorf("expected battery decrease, got %f", row.Battery)
	}
	if row.PreviousPosition != prev {
		t.Errorf("expected previous position to be set")
	}
}

// Updated tests to use MovementStrategy implementations.
func TestPatrolMovement(t *testing.T) {
	region := Region{
		Name:      "test-region",
		CenterLat: 48.2082,
		CenterLon: 16.3738,
		RadiusKM:  1,
	}
	drone := &Drone{
		Position: Position{Lat: 48.2082, Lon: 16.3738, Alt: 100},
	}
	strategy := PatrolMovement{}
	newPos := strategy.Move(drone, region, nil, rand.New(rand.NewSource(1)))
	distance := calculateDistance(region.CenterLat, region.CenterLon, newPos.Lat, newPos.Lon)
	if distance > region.RadiusKM*1000 {
		t.Errorf("Patrol movement exceeded region radius: got %f, expected <= %f", distance, region.RadiusKM*1000)
	}
}

func TestPointToPointMovement(t *testing.T) {
	waypoints := []Position{
		{Lat: 48.2083, Lon: 16.3740},
		{Lat: 48.2085, Lon: 16.3750},
	}
	drone := &Drone{
		Position: Position{Lat: 48.2082, Lon: 16.3738, Alt: 100},
	}
	strategy := PointToPointMovement{}
	newPos := strategy.Move(drone, Region{}, waypoints, rand.New(rand.NewSource(1)))

	closestWaypoint := findClosestWaypoint(newPos, waypoints)
	distanceToWaypoint := calculateDistance(newPos.Lat, newPos.Lon, closestWaypoint.Lat, closestWaypoint.Lon)
	if distanceToWaypoint > calculateDistance(drone.Position.Lat, drone.Position.Lon, closestWaypoint.Lat, closestWaypoint.Lon) {
		t.Errorf("Point-to-point movement did not move closer to waypoint")
	}
}

func TestLoiterMovement(t *testing.T) {
	region := Region{
		Name:      "test-region",
		CenterLat: 48.2082,
		CenterLon: 16.3738,
		RadiusKM:  1,
	}
	drone := &Drone{
		Position: Position{Lat: 48.2082, Lon: 16.3738, Alt: 100},
	}
	strategy := LoiterMovement{}
	newPos := strategy.Move(drone, region, nil, rand.New(rand.NewSource(1)))
	distance := calculateDistance(region.CenterLat, region.CenterLon, newPos.Lat, newPos.Lon)
	if distance > 10 {
		t.Errorf("Loiter movement exceeded allowed range: got %f, expected <= 10", distance)
	}
}

func TestRandomWalkMovement(t *testing.T) {
	drone := &Drone{
		Model:    "medium-uav",
		Position: Position{Lat: 48.2082, Lon: 16.3738, Alt: 10},
	}
	strategy := RandomWalkMovement{}
	newPos := strategy.Move(drone, Region{}, nil, rand.New(rand.NewSource(1)))
	if newPos.Alt < 0 {
		t.Errorf("altitude should not be negative: %f", newPos.Alt)
	}
	if newPos == drone.Position {
		t.Errorf("expected drone position to change")
	}
}

func TestSpeedAndHeadingPatrol(t *testing.T) { testSpeedAndHeading(t, "patrol", nil) }
func TestSpeedAndHeadingPointToPoint(t *testing.T) {
	wps := []Position{{Lat: 48.2083, Lon: 16.3740}}
	testSpeedAndHeading(t, "point-to-point", wps)
}
func TestSpeedAndHeadingLoiter(t *testing.T) { testSpeedAndHeading(t, "loiter", nil) }
func TestSpeedAndHeadingRandom(t *testing.T) { testSpeedAndHeading(t, "random", nil) }

func testSpeedAndHeading(t *testing.T, pattern string, wps []Position) {
	gen := NewGenerator("c1", rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	drone := &Drone{ID: "d1", Model: "small-fpv", MovementPattern: pattern, Position: Position{Lat: 48.0, Lon: 16.0, Alt: 100}, Waypoints: wps}
	prev := drone.Position
	row := gen.GenerateTelemetry(drone, prev, time.Second)
	expSpeed := calculateDistance(prev.Lat, prev.Lon, row.Lat, row.Lon)
	expHeading := bearingDegrees(prev.Lat, prev.Lon, row.Lat, row.Lon)
	if math.Abs(row.SpeedMPS-expSpeed) > 1e-6 {
		t.Errorf("speed mismatch: got %.6f want %.6f", row.SpeedMPS, expSpeed)
	}
	if math.Abs(row.HeadingDeg-expHeading) > 1e-6 {
		t.Errorf("heading mismatch: got %.6f want %.6f", row.HeadingDeg, expHeading)
	}
}

func TestBatteryDrain(t *testing.T) {
	cases := map[string]float64{
		"small-fpv":  0.5,
		"medium-uav": 0.3,
		"large-uav":  0.2,
		"other":      0.4,
	}
	for model, want := range cases {
		if got := batteryDrain(model); got != want {
			t.Errorf("batteryDrain(%s)=%f, want %f", model, got, want)
		}
	}
}

func TestTelemetryRowTableName(t *testing.T) {
	orig := TelemetryTableName
	TelemetryTableName = "custom"
	defer func() { TelemetryTableName = orig }()
	if (TelemetryRow{}).TableName() != "custom" {
		t.Errorf("expected custom table name, got %s", (TelemetryRow{}).TableName())
	}
}

func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000 // Earth radius in meters
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

func findClosestWaypoint(pos Position, waypoints []Position) Position {
	closest := waypoints[0]
	minDistance := calculateDistance(pos.Lat, pos.Lon, closest.Lat, closest.Lon)
	for _, waypoint := range waypoints {
		distance := calculateDistance(pos.Lat, pos.Lon, waypoint.Lat, waypoint.Lon)
		if distance < minDistance {
			closest = waypoint
			minDistance = distance
		}
	}
	return closest
}
