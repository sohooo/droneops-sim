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
	DroneLat   float64   `json:"drone_lat"`
	DroneLon   float64   `json:"drone_lon"`
	DroneAlt   float64   `json:"drone_alt"`
	DistanceM  float64   `json:"distance_m"`
	BearingDeg float64   `json:"bearing_deg"`
	EnemyVelMS float64   `json:"enemy_velocity_mps"`
	Confidence float64   `json:"confidence"`
	Timestamp  time.Time `json:"ts"`
}
