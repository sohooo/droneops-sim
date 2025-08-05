package sim

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"droneops-sim/internal/config"
	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

func TestJSONStdoutWriter(t *testing.T) {
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
	dRow := enemy.DetectionRow{ClusterID: "c1", DroneID: "d1", EnemyID: "e1", Timestamp: ts}
	sRow := telemetry.SwarmEventRow{ClusterID: "c1", EventType: telemetry.SwarmEventAssignment, DroneIDs: []string{"d1"}, EnemyID: "e1", Timestamp: ts}
	stRow := telemetry.SimulationStateRow{ClusterID: "c1", MessagesSent: 1, ChaosMode: true, Timestamp: ts}
	mRow := telemetry.MissionRow{ID: "m1", Name: "Mission", Objective: "Obj", Description: "Desc", Region: telemetry.Region{Name: "R", CenterLat: 1, CenterLon: 2, RadiusKM: 3}}

	cases := []struct {
		name   string
		write  func(*JSONStdoutWriter) error
		decode func([]byte) error
	}{
		{
			name:  "telemetry",
			write: func(w *JSONStdoutWriter) error { return w.Write(tRow) },
			decode: func(b []byte) error {
				var got telemetry.TelemetryRow
				if err := json.Unmarshal(b, &got); err != nil {
					return err
				}
				if got.SpeedMPS != tRow.SpeedMPS || got.HeadingDeg != tRow.HeadingDeg || got.PreviousPosition != prev {
					t.Fatalf("unexpected telemetry row: %#v", got)
				}
				return nil
			},
		},
		{
			name:  "detection",
			write: func(w *JSONStdoutWriter) error { return w.WriteDetection(dRow) },
			decode: func(b []byte) error {
				var got enemy.DetectionRow
				if err := json.Unmarshal(b, &got); err != nil {
					return err
				}
				if got.EnemyID != dRow.EnemyID {
					t.Fatalf("unexpected detection row: %#v", got)
				}
				return nil
			},
		},
		{
			name:  "swarm",
			write: func(w *JSONStdoutWriter) error { return w.WriteSwarmEvent(sRow) },
			decode: func(b []byte) error {
				var got telemetry.SwarmEventRow
				if err := json.Unmarshal(b, &got); err != nil {
					return err
				}
				if got.EventType != sRow.EventType || got.EnemyID != sRow.EnemyID {
					t.Fatalf("unexpected swarm row: %#v", got)
				}
				return nil
			},
		},
		{
			name:  "state",
			write: func(w *JSONStdoutWriter) error { return w.WriteState(stRow) },
			decode: func(b []byte) error {
				var got telemetry.SimulationStateRow
				if err := json.Unmarshal(b, &got); err != nil {
					return err
				}
				if got.MessagesSent != stRow.MessagesSent || got.ChaosMode != stRow.ChaosMode {
					t.Fatalf("unexpected state row: %#v", got)
				}
				return nil
			},
		},
		{
			name:  "mission",
			write: func(w *JSONStdoutWriter) error { return w.WriteMission(mRow) },
			decode: func(b []byte) error {
				var got telemetry.MissionRow
				if err := json.Unmarshal(b, &got); err != nil {
					return err
				}
				if got.ID != mRow.ID || got.Region.Name != mRow.Region.Name {
					t.Fatalf("unexpected mission row: %#v", got)
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			w := &JSONStdoutWriter{out: buf}
			if err := tc.write(w); err != nil {
				t.Fatalf("write failed: %v", err)
			}
			line := strings.TrimSpace(buf.String())
			if !strings.HasPrefix(line, "{") {
				t.Fatalf("expected JSON output, got %q", line)
			}
			if err := tc.decode([]byte(line)); err != nil {
				t.Fatalf("decode failed: %v", err)
			}
		})
	}
}

func TestColorStdoutWriter(t *testing.T) {
	cfg := &config.SimulationConfig{FollowConfidence: 50, MissionCriticality: "medium", DetectionRadiusM: 100, Missions: []config.Mission{{ID: "m1", Name: "Alpha", Objective: "test"}}}
	buf := &bytes.Buffer{}
	w := &ColorStdoutWriter{cfg: cfg, out: buf, missionColors: make(map[string]string)}
	row := telemetry.TelemetryRow{ClusterID: "c1", DroneID: "d1", MissionID: "m1", Lat: 1, Lon: 2, Alt: 3, Battery: 4, Status: telemetry.StatusOK, Timestamp: time.Unix(0, 0)}
	if err := w.Write(row); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Simulation Configuration:") || !strings.Contains(output, "Missions:") {
		t.Fatalf("overview not printed: %q", output)
	}
	if !strings.Contains(output, "\x1b[") {
		t.Fatalf("expected color codes in output: %q", output)
	}

	buf.Reset()
	if err := w.Write(row); err != nil {
		t.Fatalf("second write failed: %v", err)
	}
	if strings.Contains(buf.String(), "Simulation Configuration:") {
		t.Fatalf("overview printed more than once")
	}
}
