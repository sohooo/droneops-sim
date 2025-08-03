package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

func TestFileWriter(t *testing.T) {
	dir := t.TempDir()
	telePath := filepath.Join(dir, "telemetry.json")
	detPath := filepath.Join(dir, "detections.json")
	swarmPath := filepath.Join(dir, "swarm.json")
	fw, err := NewFileWriter(telePath, detPath, swarmPath)
	if err != nil {
		t.Fatalf("NewFileWriter: %v", err)
	}
	defer fw.Close()

	tRow := telemetry.TelemetryRow{ClusterID: "c1", DroneID: "d1", Timestamp: time.Unix(0, 0)}
	if err := fw.Write(tRow); err != nil {
		t.Fatalf("Write telemetry: %v", err)
	}
	dRow := enemy.DetectionRow{
		ClusterID:  "c1",
		DroneID:    "d1",
		EnemyID:    "e1",
		EnemyType:  enemy.EnemyVehicle,
		Lat:        1,
		Lon:        2,
		Alt:        3,
		DroneLat:   4,
		DroneLon:   5,
		DroneAlt:   6,
		DistanceM:  7,
		BearingDeg: 8,
		EnemyVelMS: 9,
		Confidence: 50,
		Timestamp:  time.Unix(0, 0),
	}
	if err := fw.WriteDetection(dRow); err != nil {
		t.Fatalf("Write detection: %v", err)
	}
	sRow := telemetry.SwarmEventRow{ClusterID: "c1", EventType: telemetry.SwarmEventAssignment, DroneIDs: []string{"d1"}, EnemyID: "e1", Timestamp: time.Unix(0, 0)}
	if err := fw.WriteSwarmEvent(sRow); err != nil {
		t.Fatalf("Write swarm: %v", err)
	}

	fw.Close()

	// Verify telemetry file
	tData, err := os.ReadFile(telePath)
	if err != nil {
		t.Fatalf("read tele: %v", err)
	}
	var gotT telemetry.TelemetryRow
	if err := json.Unmarshal(tData, &gotT); err != nil {
		t.Fatalf("decode tele: %v", err)
	}
	if gotT.DroneID != tRow.DroneID {
		t.Fatalf("unexpected telemetry row: %+v", gotT)
	}

	dData, err := os.ReadFile(detPath)
	if err != nil {
		t.Fatalf("read det: %v", err)
	}
	var gotD enemy.DetectionRow
	if err := json.Unmarshal(dData, &gotD); err != nil {
		t.Fatalf("decode det: %v", err)
	}
	if gotD.EnemyID != dRow.EnemyID || gotD.DistanceM != dRow.DistanceM {
		t.Fatalf("unexpected detection row: %+v", gotD)
	}

	sData, err := os.ReadFile(swarmPath)
	if err != nil {
		t.Fatalf("read swarm: %v", err)
	}
	var gotS telemetry.SwarmEventRow
	if err := json.Unmarshal(sData, &gotS); err != nil {
		t.Fatalf("decode swarm: %v", err)
	}
	if gotS.EventType != sRow.EventType || gotS.EnemyID != sRow.EnemyID {
		t.Fatalf("unexpected swarm row: %+v", gotS)
	}
}
