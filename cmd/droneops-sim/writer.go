package main

import (
	"os"

	"droneops-sim/internal/config"
	"droneops-sim/internal/sim"
)

// newWriters sets up telemetry and detection writers based on flags and env vars.
// It returns the writers and a cleanup function to close any resources.
func newWriters(cfg *config.SimulationConfig, printOnly bool, logFile string) (sim.TelemetryWriter, sim.DetectionWriter, func(), error) {
	cleanup := func() {}

	writer, detectWriter, err := baseWriters(cfg, printOnly)
	if err != nil {
		return nil, nil, nil, err
	}
	if logFile == "" {
		return writer, detectWriter, cleanup, nil
	}

	fw, err := sim.NewFileWriter(logFile, logFile+".detections", logFile+".swarm", logFile+".state")
	if err != nil {
		return nil, nil, nil, err
	}
	sws := []sim.SwarmEventWriter{fw}
	if sw, ok := writer.(sim.SwarmEventWriter); ok {
		sws = append(sws, sw)
	}
	mw := sim.NewMultiWriter([]sim.TelemetryWriter{writer, fw}, []sim.DetectionWriter{detectWriter, fw}, sws)
	cleanup = func() { fw.Close() }
	return mw, mw, cleanup, nil
}

// baseWriters chooses the underlying writers based on printOnly flag and env vars.
func baseWriters(cfg *config.SimulationConfig, printOnly bool) (sim.TelemetryWriter, sim.DetectionWriter, error) {
	if printOnly || os.Getenv("GREPTIMEDB_ENDPOINT") == "" {
		tw, dw := sim.NewStdoutWriters(cfg)
		return tw, dw, nil
	}

	endpoint := os.Getenv("GREPTIMEDB_ENDPOINT")
	table := os.Getenv("GREPTIMEDB_TABLE")
	detTable := os.Getenv("ENEMY_DETECTION_TABLE")
	swarmTable := os.Getenv("SWARM_EVENT_TABLE")
	stateTable := os.Getenv("SIMULATION_STATE_TABLE")
	w, err := sim.NewGreptimeDBWriter(endpoint, "public", table, detTable, swarmTable, stateTable)
	if err != nil {
		return nil, nil, err
	}
	return w, w, nil
}

// newTelemetryWriter creates a telemetry writer without detection handling.
func newTelemetryWriter(cfg *config.SimulationConfig, printOnly bool) (sim.TelemetryWriter, error) {
	w, _, _, err := newWriters(cfg, printOnly, "")
	return w, err
}
