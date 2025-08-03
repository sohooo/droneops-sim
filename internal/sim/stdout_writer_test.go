package sim

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"droneops-sim/internal/config"
	"droneops-sim/internal/telemetry"
)

func TestStdoutWriterJSONFallback(t *testing.T) {
	buf := &bytes.Buffer{}
	w := &StdoutWriter{out: buf, colorize: false, missionColors: make(map[string]string)}
	row := telemetry.TelemetryRow{ClusterID: "c1", DroneID: "d1", Timestamp: time.Unix(0, 0)}
	if err := w.Write(row); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(buf.String()), "{") {
		t.Fatalf("expected JSON output, got %q", buf.String())
	}
}

func TestStdoutWriterColorized(t *testing.T) {
	cfg := &config.SimulationConfig{FollowConfidence: 50, MissionCriticality: "medium", DetectionRadiusM: 100, Missions: []config.Mission{{ID: "m1", Name: "Alpha", Objective: "test"}}}
	buf := &bytes.Buffer{}
	w := &StdoutWriter{cfg: cfg, colorize: true, out: buf, missionColors: make(map[string]string)}
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
