package main

import (
	"os"

	"droneops-sim/internal/sim"
)

// newWriters sets up telemetry and detection writers based on flags and env vars.
// It returns the writers and a cleanup function to close any resources.
func newWriters(printOnly bool, logFile string) (sim.TelemetryWriter, sim.DetectionWriter, func(), error) {
	cleanup := func() {}

	writer, detectWriter, err := baseWriters(printOnly)
	if err != nil {
		return nil, nil, nil, err
	}
	if logFile == "" {
		return writer, detectWriter, cleanup, nil
	}

	fw, err := sim.NewFileWriter(logFile, logFile+".detections")
	if err != nil {
		return nil, nil, nil, err
	}
	mw := sim.NewMultiWriter([]sim.TelemetryWriter{writer, fw}, []sim.DetectionWriter{detectWriter, fw})
	cleanup = func() { fw.Close() }
	return mw, mw, cleanup, nil
}

// baseWriters chooses the underlying writers based on printOnly flag and env vars.
func baseWriters(printOnly bool) (sim.TelemetryWriter, sim.DetectionWriter, error) {
	if printOnly || os.Getenv("GREPTIMEDB_ENDPOINT") == "" {
		sw := &sim.StdoutWriter{}
		return sw, sw, nil
	}

	endpoint := os.Getenv("GREPTIMEDB_ENDPOINT")
	table := os.Getenv("GREPTIMEDB_TABLE")
	detTable := os.Getenv("ENEMY_DETECTION_TABLE")
	w, err := sim.NewGreptimeDBWriter(endpoint, "public", table, detTable)
	if err != nil {
		return nil, nil, err
	}
	return w, w, nil
}

// newTelemetryWriter creates a telemetry writer without detection handling.
func newTelemetryWriter(printOnly bool) (sim.TelemetryWriter, error) {
	w, _, _, err := newWriters(printOnly, "")
	return w, err
}
