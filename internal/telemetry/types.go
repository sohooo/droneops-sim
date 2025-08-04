// Telemetry structs with greptime tags
package telemetry

import (
	"os"
	"time"
)

// MissionRow represents one mission record for telemetry.
type MissionRow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Objective   string `json:"objective"`
	Description string `json:"description"`
	Region      Region `json:"region"`
}

// TelemetryRow represents one telemetry record for GreptimeDB.
type TelemetryRow struct {
	ClusterID        string    `json:"cluster_id"`        // TAG
	DroneID          string    `json:"drone_id"`          // TAG
	MissionID        string    `json:"mission_id"`        // Added field for mission association
	Lat              float64   `json:"lat"`               // FIELD
	Lon              float64   `json:"lon"`               // FIELD
	Alt              float64   `json:"alt"`               // FIELD
	Battery          float64   `json:"battery"`           // FIELD
	Status           string    `json:"status"`            // FIELD
	Follow           bool      `json:"follow"`            // FIELD indicates active follow mode
	MovementPattern  string    `json:"movement_pattern"`  // FIELD movement pattern
	SpeedMPS         float64   `json:"speed_mps"`         // FIELD speed in meters/second
	HeadingDeg       float64   `json:"heading_deg"`       // FIELD heading in degrees
	PreviousPosition Position  `json:"previous_position"` // FIELD previous position
	SyncedFrom       string    `json:"synced_from"`       // Added by sync process
	SyncedID         string    `json:"synced_id"`         // Added by sync process
	SyncedAt         time.Time `json:"synced_at"`         // Added by sync process
	Timestamp        time.Time `json:"ts"`                // TIME INDEX
}

// TelemetryTableName holds the table name used when writing to GreptimeDB.
// It defaults to "drone_telemetry" but can be overridden via the
// GREPTIMEDB_TABLE environment variable.
var TelemetryTableName = func() string {
	if env := os.Getenv("GREPTIMEDB_TABLE"); env != "" {
		return env
	}
	return "drone_telemetry"
}()

func (TelemetryRow) TableName() string {
	return TelemetryTableName
}

// Drone holds runtime state for a simulated drone.
type Drone struct {
	ID                 string     // Drone ID
	Model              string     // Drone model
	MissionID          string     // Associated mission ID
	Position           Position   // Current position
	Battery            float64    // Battery level
	Status             string     // Current status
	MovementPattern    string     // Movement pattern: patrol, point-to-point, loiter
	HomeRegion         Region     // Home region for patrol and loiter
	Waypoints          []Position // Waypoints for point-to-point movement
	FollowTarget       *Position  // If set, drone will move toward this target
	SensorErrorRate    float64
	DropoutRate        float64
	BatteryAnomalyRate float64
}

// Position holds latitude, longitude, and altitude.
type Position struct {
	Lat float64 // Latitude
	Lon float64 // Longitude
	Alt float64 // Altitude
}

// Region defines an operational region.
type Region struct {
	Name      string  // Name of the region
	CenterLat float64 // Latitude of the region center
	CenterLon float64 // Longitude of the region center
	RadiusKM  float64 // Radius of the region in kilometers
}

// Drone status constants.
const (
	StatusOK         = "ok"
	StatusLowBattery = "low_battery"
	StatusFailure    = "failed"
)

// Battery status thresholds in percentage.
const (
	BatteryFailureThreshold = 5.0  // Battery at or below this is a failure
	BatteryLowThreshold     = 20.0 // Battery at or below this is low
)
