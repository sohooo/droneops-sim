package sim

import "droneops-sim/internal/telemetry"

// StateWriter handles simulation state rows.
type StateWriter interface {
	WriteState(telemetry.SimulationStateRow) error
}

// Optional: writers may support batch mode for state rows.
type batchStateWriter interface {
	WriteStates([]telemetry.SimulationStateRow) error
}
