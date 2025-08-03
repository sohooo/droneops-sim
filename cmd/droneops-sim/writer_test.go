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
	tw, dw, cleanup, err := newWriters(true, "")
	if err != nil {
		t.Fatalf("newWriters returned error: %v", err)
	}
	cleanup()
	if _, ok := tw.(*sim.StdoutWriter); !ok {
		t.Fatalf("expected *sim.StdoutWriter, got %T", tw)
	}
	if _, ok := dw.(*sim.StdoutWriter); !ok {
		t.Fatalf("expected *sim.StdoutWriter, got %T", dw)
	}
}

func TestNewWritersGreptimeFallback(t *testing.T) {
	t.Setenv("GREPTIMEDB_ENDPOINT", "")
	tw, dw, cleanup, err := newWriters(false, "")
	if err != nil {
		t.Fatalf("newWriters returned error: %v", err)
	}
	cleanup()
	if _, ok := tw.(*sim.StdoutWriter); !ok {
		t.Fatalf("expected *sim.StdoutWriter, got %T", tw)
	}
	if _, ok := dw.(*sim.StdoutWriter); !ok {
		t.Fatalf("expected *sim.StdoutWriter, got %T", dw)
	}
}

func TestNewWritersLogFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "telemetry.log")
	tw, _, cleanup, err := newWriters(true, path)
	if err != nil {
		t.Fatalf("newWriters returned error: %v", err)
	}
	defer cleanup()
	if _, ok := tw.(*sim.MultiWriter); !ok {
		t.Fatalf("expected *sim.MultiWriter, got %T", tw)
	}
	row := telemetry.TelemetryRow{ClusterID: "c1", DroneID: "d1", Timestamp: time.Now()}
	if err := tw.Write(row); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected log file to be non-empty")
	}
}
