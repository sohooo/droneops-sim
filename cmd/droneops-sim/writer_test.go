package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"droneops-sim/internal/sim"
	"droneops-sim/internal/telemetry"
)

func TestNewWritersPrintOnly(t *testing.T) {
	tw, dw, mw, cleanup, err := newWriters(nil, true, "", true, true, true)
	if err != nil {
		t.Fatalf("newWriters returned error: %v", err)
	}
	cleanup()
	if _, ok := tw.(*sim.JSONStdoutWriter); !ok {
		t.Fatalf("expected *sim.JSONStdoutWriter, got %T", tw)
	}
	if _, ok := dw.(*sim.JSONStdoutWriter); !ok {
		t.Fatalf("expected *sim.JSONStdoutWriter, got %T", dw)
	}
	if _, ok := mw.(*sim.JSONStdoutWriter); !ok {
		t.Fatalf("expected mission writer *sim.JSONStdoutWriter, got %T", mw)
	}
}

func TestNewWritersGreptimeFallback(t *testing.T) {
	t.Setenv("GREPTIMEDB_ENDPOINT", "")
	tw, dw, mw, cleanup, err := newWriters(nil, false, "", true, true, true)
	if err != nil {
		t.Fatalf("newWriters returned error: %v", err)
	}
	cleanup()
	if _, ok := tw.(*sim.JSONStdoutWriter); !ok {
		t.Fatalf("expected *sim.JSONStdoutWriter, got %T", tw)
	}
	if _, ok := dw.(*sim.JSONStdoutWriter); !ok {
		t.Fatalf("expected *sim.JSONStdoutWriter, got %T", dw)
	}
	if _, ok := mw.(*sim.JSONStdoutWriter); !ok {
		t.Fatalf("expected mission writer *sim.JSONStdoutWriter, got %T", mw)
	}
}

func TestNewWritersLogFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "telemetry.log")
	tw, _, mw, cleanup, err := newWriters(nil, true, path, true, true, true)
	if err != nil {
		t.Fatalf("newWriters returned error: %v", err)
	}
	defer cleanup()
	if _, ok := tw.(*sim.MultiWriter); !ok {
		t.Fatalf("expected *sim.MultiWriter, got %T", tw)
	}
	if _, ok := mw.(*sim.MultiWriter); !ok {
		t.Fatalf("expected mission writer *sim.MultiWriter, got %T", mw)
	}
	row := telemetry.TelemetryRow{ClusterID: "c1", DroneID: "d1", Timestamp: time.Now()}
	if err := tw.Write(row); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	sw, ok := tw.(sim.StateWriter)
	if !ok {
		t.Fatalf("telemetry writer does not implement StateWriter")
	}
	st := telemetry.SimulationStateRow{ClusterID: "c1", CommunicationLoss: 0.1, MessagesSent: 1, SensorNoise: 0.2, WeatherImpact: 0.3, ChaosMode: true, Timestamp: time.Now()}
	if err := sw.WriteState(st); err != nil {
		t.Fatalf("write state failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected log file to be non-empty")
	}
	stateInfo, err := os.Stat(path + ".state")
	if err != nil {
		t.Fatalf("stat state failed: %v", err)
	}
	if stateInfo.Size() == 0 {
		t.Fatalf("expected state file to be non-empty")
	}
}

func TestNewWritersDisableDetections(t *testing.T) {
	tw, dw, mw, cleanup, err := newWriters(nil, true, "", false, true, true)
	if err != nil {
		t.Fatalf("newWriters returned error: %v", err)
	}
	cleanup()
	if dw != nil {
		t.Fatalf("expected detection writer to be nil when disabled")
	}
	if _, ok := tw.(*sim.JSONStdoutWriter); !ok {
		t.Fatalf("expected *sim.JSONStdoutWriter, got %T", tw)
	}
	if _, ok := mw.(*sim.JSONStdoutWriter); !ok {
		t.Fatalf("expected mission writer *sim.JSONStdoutWriter, got %T", mw)
	}
}
