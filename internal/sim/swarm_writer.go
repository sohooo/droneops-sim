package sim

import "droneops-sim/internal/telemetry"

// SwarmEventWriter handles swarm coordination events.
type SwarmEventWriter interface {
	WriteSwarmEvent(telemetry.SwarmEventRow) error
}

// Optional: writers may support batch mode for swarm events.
type batchSwarmEventWriter interface {
	WriteSwarmEvents([]telemetry.SwarmEventRow) error
}
