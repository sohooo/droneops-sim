package sim

import (
	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

// MultiWriter fan-outs telemetry and detection rows to multiple writers.
type MultiWriter struct {
	telewriters  []TelemetryWriter
	detwriters   []DetectionWriter
	swarmwriters []SwarmEventWriter
}

// NewMultiWriter creates a new MultiWriter.
func NewMultiWriter(tws []TelemetryWriter, dws []DetectionWriter, sws []SwarmEventWriter) *MultiWriter {
	return &MultiWriter{telewriters: tws, detwriters: dws, swarmwriters: sws}
}

// Write sends a telemetry row to all writers.
func (mw *MultiWriter) Write(row telemetry.TelemetryRow) error {
	for _, w := range mw.telewriters {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// WriteBatch sends multiple telemetry rows to all writers, using batch if supported.
func (mw *MultiWriter) WriteBatch(rows []telemetry.TelemetryRow) error {
	for _, w := range mw.telewriters {
		if bw, ok := w.(batchWriter); ok {
			if err := bw.WriteBatch(rows); err != nil {
				return err
			}
			continue
		}
		for _, r := range rows {
			if err := w.Write(r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteDetection sends a detection row to all detection writers.
func (mw *MultiWriter) WriteDetection(row enemy.DetectionRow) error {
	for _, w := range mw.detwriters {
		if err := w.WriteDetection(row); err != nil {
			return err
		}
	}
	return nil
}

// WriteDetections sends multiple detections to all detection writers, using batch if supported.
func (mw *MultiWriter) WriteDetections(rows []enemy.DetectionRow) error {
	for _, w := range mw.detwriters {
		if bw, ok := w.(batchDetectionWriter); ok {
			if err := bw.WriteDetections(rows); err != nil {
				return err
			}
			continue
		}
		for _, r := range rows {
			if err := w.WriteDetection(r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteSwarmEvent sends a swarm event row to all swarm writers.
func (mw *MultiWriter) WriteSwarmEvent(row telemetry.SwarmEventRow) error {
	for _, w := range mw.swarmwriters {
		if err := w.WriteSwarmEvent(row); err != nil {
			return err
		}
	}
	return nil
}

// WriteSwarmEvents sends multiple swarm events, using batch mode if supported.
func (mw *MultiWriter) WriteSwarmEvents(rows []telemetry.SwarmEventRow) error {
	for _, w := range mw.swarmwriters {
		if bw, ok := w.(batchSwarmEventWriter); ok {
			if err := bw.WriteSwarmEvents(rows); err != nil {
				return err
			}
			continue
		}
		for _, r := range rows {
			if err := w.WriteSwarmEvent(r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteState sends a simulation state row to all telemetry writers that support it.
func (mw *MultiWriter) WriteState(row telemetry.SimulationStateRow) error {
	for _, w := range mw.telewriters {
		if sw, ok := w.(StateWriter); ok {
			if err := sw.WriteState(row); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteStates sends multiple simulation state rows using batch mode if supported.
func (mw *MultiWriter) WriteStates(rows []telemetry.SimulationStateRow) error {
	for _, w := range mw.telewriters {
		if bw, ok := w.(batchStateWriter); ok {
			if err := bw.WriteStates(rows); err != nil {
				return err
			}
			continue
		}
		if sw, ok := w.(StateWriter); ok {
			for _, r := range rows {
				if err := sw.WriteState(r); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// SetAdminStatus forwards admin UI status to underlying writers that support it.
func (mw *MultiWriter) SetAdminStatus(active bool) {
	for _, w := range mw.telewriters {
		if aw, ok := w.(AdminStatusWriter); ok {
			aw.SetAdminStatus(active)
		}
	}
}

// Close closes underlying writers that support it.
func (mw *MultiWriter) Close() error {
	for _, w := range mw.telewriters {
		if c, ok := w.(interface{ Close() error }); ok {
			_ = c.Close()
		}
	}
	return nil
}
