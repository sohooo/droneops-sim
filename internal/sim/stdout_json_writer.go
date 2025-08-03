package sim

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

// JSONStdoutWriter prints telemetry and detections as JSON to STDOUT.
type JSONStdoutWriter struct {
	out io.Writer
}

// NewJSONStdoutWriter creates a JSONStdoutWriter writing to os.Stdout.
func NewJSONStdoutWriter() *JSONStdoutWriter {
	return &JSONStdoutWriter{out: os.Stdout}
}

// Write outputs a telemetry row in JSON format.
func (w *JSONStdoutWriter) Write(row telemetry.TelemetryRow) error {
	data, _ := json.Marshal(row)
	fmt.Fprintln(w.out, string(data))
	return nil
}

// WriteBatch outputs multiple telemetry rows in JSON format.
func (w *JSONStdoutWriter) WriteBatch(rows []telemetry.TelemetryRow) error {
	for _, r := range rows {
		_ = w.Write(r)
	}
	return nil
}

// WriteDetection outputs an enemy detection event in JSON format.
func (w *JSONStdoutWriter) WriteDetection(d enemy.DetectionRow) error {
	data, _ := json.Marshal(d)
	fmt.Fprintln(w.out, string(data))
	return nil
}

// WriteDetections outputs multiple enemy detections in JSON format.
func (w *JSONStdoutWriter) WriteDetections(rows []enemy.DetectionRow) error {
	for _, d := range rows {
		_ = w.WriteDetection(d)
	}
	return nil
}
