package sim

import "droneops-sim/internal/telemetry"

// MissionWriter handles mission metadata rows.
type MissionWriter interface {
	WriteMission(telemetry.MissionRow) error
}

// Optional: Mission writers may support batch mode.
type batchMissionWriter interface {
	WriteMissions([]telemetry.MissionRow) error
}
