// Writer implementation printing telemetry to STDOUT
package sim

import (
	"encoding/json"
	"fmt"

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