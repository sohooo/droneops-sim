package telemetry

import (
	"testing"
	"time"
)

func TestGenerateTelemetry_BatteryDrainAndStatus(t *testing.T) {
	drone := &Drone{
		ID:       "test-drone",
		Model:    "small-fpv",
		Position: Position{Lat: 48.2, Lon: 16.4, Alt: 100},
		Battery:  100,
		Status:   StatusOK,
	}
	gen := NewGenerator("cluster-01")

	// Run multiple ticks to drain battery
	for i := 0; i < 200; i++ {
		_ = gen.GenerateTelemetry(drone)
	}

	if drone.Battery <= 0 {
		t.Logf("Battery drained fully after simulation: %f", drone.Battery)
	} else {
		t.Errorf("Expected battery to drain significantly, got %f", drone.Battery)
	}

	if drone.Status != StatusFailure && drone.Battery <= 5 {
		t.Errorf("Expected drone to be in failure state when battery low, got %s", drone.Status)
	}
}

func TestGenerateTelemetry_PositionChanges(t *testing.T) {
	drone := &Drone{
		ID:       "test-drone",
		Model:    "medium-uav",
		Position: Position{Lat: 48.2, Lon: 16.4, Alt: 100},
		Battery:  100,
		Status:   StatusOK,
	}
	gen := NewGenerator("cluster-02")

	firstLat := drone.Position.Lat
	firstLon := drone.Position.Lon

	row := gen.GenerateTelemetry(drone)
	if row.Lat == firstLat && row.Lon == firstLon {
		t.Errorf("Expected drone position to change, but it did not.")
	}
	if row.Timestamp.After(time.Now()) {
		t.Errorf("Timestamp should not be in the future")
	}
}