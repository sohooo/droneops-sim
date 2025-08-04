package main

import (
	"os"

	"droneops-sim/internal/config"
	"droneops-sim/internal/sim"
)

// newWriters sets up telemetry and detection writers based on flags and env vars.
// It returns the writers and a cleanup function to close any resources.
func newWriters(cfg *config.SimulationConfig, printOnly bool, logFile string, enableDetections, enableSwarm, enableState bool) (sim.TelemetryWriter, sim.DetectionWriter, func(), error) {
	cleanup := func() {}

	writer, detectWriter, err := baseWriters(cfg, printOnly)
	if err != nil {
		return nil, nil, nil, err
	}
	if !enableDetections {
		detectWriter = nil
	}
	if logFile == "" {
		return writer, detectWriter, cleanup, nil
	}

	detPath := ""
	if enableDetections {
		detPath = logFile + ".detections"
	}
	swarmPath := ""
	if enableSwarm {
		swarmPath = logFile + ".swarm"
	}
	statePath := ""
	if enableState {
		statePath = logFile + ".state"
	}
	fw, err := sim.NewFileWriter(logFile, detPath, swarmPath, statePath)
	if err != nil {
		return nil, nil, nil, err
	}
	tws := []sim.TelemetryWriter{writer, fw}
	dws := []sim.DetectionWriter{}
	if detectWriter != nil {
		dws = append(dws, detectWriter)
	}
	if enableDetections {
		dws = append(dws, fw)
	}
	sws := []sim.SwarmEventWriter{}
	if enableSwarm {
		if sw, ok := writer.(sim.SwarmEventWriter); ok {
			sws = append(sws, sw)
		}
		sws = append(sws, fw)
	}
	mw := sim.NewMultiWriter(tws, dws, sws)
	cleanup = func() { fw.Close() }
	if enableDetections {
		return mw, mw, cleanup, nil
	}
	return mw, nil, cleanup, nil
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
	w, _, _, err := newWriters(cfg, printOnly, "", true, true, true)
	return w, err
}
