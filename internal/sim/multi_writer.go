package sim

import (
	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

// MultiWriter fan-outs telemetry and detection rows to multiple writers.
type MultiWriter struct {
	telewriters []TelemetryWriter
	detwriters  []DetectionWriter
}

// NewMultiWriter creates a new MultiWriter.
func NewMultiWriter(tws []TelemetryWriter, dws []DetectionWriter) *MultiWriter {
	return &MultiWriter{telewriters: tws, detwriters: dws}
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
