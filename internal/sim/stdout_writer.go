package sim

import (
	"os"

	"golang.org/x/term"

	"droneops-sim/internal/config"
)

// NewStdoutWriters selects a stdout writer based on terminal capability.
// It returns both telemetry and detection writers.
func NewStdoutWriters(cfg *config.SimulationConfig) (TelemetryWriter, DetectionWriter) {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		w := NewTUIWriter(cfg)
		return w, w
	}
	w := NewJSONStdoutWriter()
	return w, w
}
