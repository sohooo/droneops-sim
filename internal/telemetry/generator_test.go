package telemetry

import (
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