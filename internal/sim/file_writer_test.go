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
	ts := time.Unix(0, 0).UTC()
	prev := telemetry.Position{Lat: 1, Lon: 2, Alt: 3}
	tRow := telemetry.TelemetryRow{
		ClusterID:        "c1",
		DroneID:          "d1",
		Lat:              4,
		Lon:              5,
		Alt:              6,
		MovementPattern:  "patrol",
		SpeedMPS:         7,
		HeadingDeg:       8,
		PreviousPosition: prev,
		Timestamp:        ts,
	}
	dRow := enemy.DetectionRow{ClusterID: "c1", DroneID: "d1", EnemyID: "e1", DistanceM: 10, Timestamp: ts}
	sRow := telemetry.SwarmEventRow{ClusterID: "c1", EventType: telemetry.SwarmEventAssignment, DroneIDs: []string{"d1"}, EnemyID: "e1", Timestamp: ts}
	stRow := telemetry.SimulationStateRow{ClusterID: "c1", MessagesSent: 1, ChaosMode: true, Timestamp: ts}

	cases := []struct {
		name   string
		path   string
		write  func(*FileWriter) error
		decode func([]byte)
	}{
		{
			name:  "telemetry",
			path:  filepath.Join(dir, "telemetry.json"),
			write: func(fw *FileWriter) error { return fw.Write(tRow) },
			decode: func(b []byte) {
				var got telemetry.TelemetryRow
				if err := json.Unmarshal(b, &got); err != nil {
					t.Fatalf("decode telemetry: %v", err)
				}
				if got.SpeedMPS != tRow.SpeedMPS || got.HeadingDeg != tRow.HeadingDeg || got.PreviousPosition != prev {
					t.Fatalf("unexpected telemetry: %#v", got)
				}
			},
		},
		{
			name:  "detection",
			path:  filepath.Join(dir, "detections.json"),
			write: func(fw *FileWriter) error { return fw.WriteDetection(dRow) },
			decode: func(b []byte) {
				var got enemy.DetectionRow
				if err := json.Unmarshal(b, &got); err != nil {
					t.Fatalf("decode detection: %v", err)
				}
				if got.EnemyID != dRow.EnemyID || got.DistanceM != dRow.DistanceM {
					t.Fatalf("unexpected detection: %#v", got)
				}
			},
		},
		{
			name:  "swarm",
			path:  filepath.Join(dir, "swarm.json"),
			write: func(fw *FileWriter) error { return fw.WriteSwarmEvent(sRow) },
			decode: func(b []byte) {
				var got telemetry.SwarmEventRow
				if err := json.Unmarshal(b, &got); err != nil {
					t.Fatalf("decode swarm: %v", err)
				}
				if got.EventType != sRow.EventType || got.EnemyID != sRow.EnemyID {
					t.Fatalf("unexpected swarm: %#v", got)
				}
			},
		},
		{
			name:  "state",
			path:  filepath.Join(dir, "state.json"),
			write: func(fw *FileWriter) error { return fw.WriteState(stRow) },
			decode: func(b []byte) {
				var got telemetry.SimulationStateRow
				if err := json.Unmarshal(b, &got); err != nil {
					t.Fatalf("decode state: %v", err)
				}
				if got.MessagesSent != stRow.MessagesSent || got.ChaosMode != stRow.ChaosMode {
					t.Fatalf("unexpected state: %#v", got)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tele := filepath.Join(dir, tc.name+"_tele.json")
			var det, swarm, state string
			switch tc.name {
			case "telemetry":
				tele = tc.path
			case "detection":
				det = tc.path
			case "swarm":
				swarm = tc.path
			case "state":
				state = tc.path
			}
			fw, err := NewFileWriter(tele, det, swarm, state)
			if err != nil {
				t.Fatalf("NewFileWriter: %v", err)
			}
			if err := tc.write(fw); err != nil {
				t.Fatalf("write: %v", err)
			}
			fw.Close()
			data, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			tc.decode(data)
		})
	}
}
