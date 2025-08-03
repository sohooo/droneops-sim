package telemetry

import "time"

const (
	SwarmEventAssignment      = "assignment"
	SwarmEventUnassignment    = "unassignment"
	SwarmEventFormationChange = "formation_change"
)

// SwarmEventRow represents a swarm coordination event.
type SwarmEventRow struct {
	ClusterID string    `json:"cluster_id"`
	EventType string    `json:"event_type"`
	DroneIDs  []string  `json:"drone_ids"`
	EnemyID   string    `json:"enemy_id,omitempty"`
	Timestamp time.Time `json:"ts"`
}
