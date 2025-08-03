package enemy

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDetectionRowJSON(t *testing.T) {
	d := DetectionRow{
		ClusterID:  "c",
		DroneID:    "d",
		EnemyID:    "e",
		EnemyType:  EnemyVehicle,
		Lat:        1,
		Lon:        2,
		Alt:        3,
		DroneLat:   4,
		DroneLon:   5,
		DroneAlt:   6,
		DistanceM:  7,
		BearingDeg: 8,
		EnemyVelMS: 9,
		Confidence: 10,
		Timestamp:  time.Unix(0, 0).UTC(),
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"drone_lat", "drone_lon", "drone_alt", "distance_m", "bearing_deg", "enemy_velocity_mps"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing %s in json: %s", key, string(data))
		}
	}
}
