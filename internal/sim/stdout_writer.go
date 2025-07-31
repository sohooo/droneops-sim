// Writer implementation printing telemetry to STDOUT
package sim

import (
	"encoding/json"
	"fmt"

	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

// StdoutWriter prints telemetry rows to STDOUT.
type StdoutWriter struct{}

// Write outputs a single telemetry row.
func (w *StdoutWriter) Write(row telemetry.TelemetryRow) error {
	data, _ := json.Marshal(row)
	fmt.Println(string(data))
	return nil
}

// WriteBatch outputs multiple telemetry rows.
func (w *StdoutWriter) WriteBatch(rows []telemetry.TelemetryRow) error {
	for _, r := range rows {
		_ = w.Write(r)
	}
	return nil
}

// WriteDetection prints an enemy detection event to STDOUT.
func (w *StdoutWriter) WriteDetection(d enemy.DetectionRow) error {
	data, _ := json.Marshal(d)
	fmt.Println(string(data))
	return nil
}

// WriteDetections prints multiple enemy detections.
func (w *StdoutWriter) WriteDetections(rows []enemy.DetectionRow) error {
	for _, d := range rows {
		_ = w.WriteDetection(d)
	}
	return nil
}
