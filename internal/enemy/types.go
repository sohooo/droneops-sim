package enemy

import (
	"time"

	"droneops-sim/internal/telemetry"
)

// EnemyType represents the type of enemy object.
type EnemyType string

const (
	EnemyVehicle EnemyType = "vehicle"
	EnemyPerson  EnemyType = "person"
	EnemyDrone   EnemyType = "drone"
	EnemyDecoy   EnemyType = "decoy"
)

// Enemy represents one simulated enemy entity.
type Enemy struct {
	ID         string
	Type       EnemyType
	Position   telemetry.Position
	Confidence float64
	Region     telemetry.Region
}

// DetectionRow describes a drone enemy detection event.
type DetectionRow struct {
	ClusterID  string    `json:"cluster_id"`
	DroneID    string    `json:"drone_id"`
	EnemyID    string    `json:"enemy_id"`
	EnemyType  EnemyType `json:"enemy_type"`
	Lat        float64   `json:"lat"`
	Lon        float64   `json:"lon"`
	Alt        float64   `json:"alt"`
	Confidence float64   `json:"confidence"`
	Timestamp  time.Time `json:"ts"`
}
