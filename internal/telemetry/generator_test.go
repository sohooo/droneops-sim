package telemetry

import (
	"math"
	"testing"
	"time"
)

func TestGenerateTelemetry(t *testing.T) {
	gen := NewGenerator("cluster-1")
	drone := &Drone{
		ID:       "drone-001",
		Model:    "small-fpv",
		Position: Position{Lat: 48.2082, Lon: 16.3738, Alt: 100},
		Battery:  50,
		Status:   StatusOK,
	}

	row := gen.GenerateTelemetry(drone)

	if row.ClusterID != "cluster-1" {
		t.Errorf("expected cluster-1, got %s", row.ClusterID)
	}
	if row.DroneID != "drone-001" {
		t.Errorf("expected drone-001, got %s", row.DroneID)
	}
	if row.SyncedFrom != "" || row.SyncedID != "" || !row.SyncedAt.IsZero() {
		t.Errorf("expected unsynced defaults, got %+v", row)
	}
	if time.Since(row.Timestamp) > 1*time.Second {
		t.Errorf("timestamp too old: %v", row.Timestamp)
	}
	// Check that position changed (movement simulated)
	if row.Lat == 48.2082 && row.Lon == 16.3738 {
		t.Errorf("expected position to change")
	}
	// Battery should decrease
	if row.Battery >= 50 {
		t.Errorf("expected battery decrease, got %f", row.Battery)
	}
}

func TestPatrolMovement(t *testing.T) {
	region := Region{
		Name:      "test-region",
		CenterLat: 48.2082,
		CenterLon: 16.3738,
		RadiusKM:  1,
	}
	pos := Position{Lat: 48.2082, Lon: 16.3738, Alt: 100}
	newPos := patrolMovement(pos, region)

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
	pos := Position{Lat: 48.2082, Lon: 16.3738, Alt: 100}
	newPos := pointToPointMovement(pos, waypoints)

	closestWaypoint := findClosestWaypoint(newPos, waypoints)
	distanceToWaypoint := calculateDistance(newPos.Lat, newPos.Lon, closestWaypoint.Lat, closestWaypoint.Lon)
	if distanceToWaypoint > calculateDistance(pos.Lat, pos.Lon, closestWaypoint.Lat, closestWaypoint.Lon) {
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
	pos := Position{Lat: 48.2082, Lon: 16.3738, Alt: 100}
	newPos := loiterMovement(pos, region)

	distance := calculateDistance(region.CenterLat, region.CenterLon, newPos.Lat, newPos.Lon)
	if distance > 10 { // Loiter movement should stay within 10 meters
		t.Errorf("Loiter movement exceeded allowed range: got %f, expected <= 10", distance)
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
