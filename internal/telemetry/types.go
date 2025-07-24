// Telemetry structs with greptime tags
package telemetry

import "time"

// TelemetryRow represents one record written to GreptimeDB.
type TelemetryRow struct {
	ClusterID    string    `greptime:"tag;column:cluster_id;type:string"`
	DroneID      string    `greptime:"tag;column:drone_id;type:string"`
	Lat          float64   `greptime:"field;column:lat;type:float64"`
	Lon          float64   `greptime:"field;column:lon;type:float64"`
	Alt          float64   `greptime:"field;column:alt;type:float64"`
	Battery      float64   `greptime:"field;column:battery;type:float64"`
	Status       string    `greptime:"field;column:status;type:string"`
	Timestamp    time.Time `greptime:"timestamp;column:ts;type:timestamp;precision:millisecond"`
}

// TableName returns the target table name for ORM mapping.
func (TelemetryRow) TableName() string {
	return "drone_telemetry"
}

// Drone holds runtime state for a simulated drone.
type Drone struct {
	ID       string
	Model    string
	Position Position
	Battery  float64
	Status   string
}

// Position holds latitude, longitude, and altitude.
type Position struct {
	Lat float64
	Lon float64
	Alt float64
}

// Drone status constants.
const (
	StatusOK         = "ok"
	StatusLowBattery = "low_battery"
	StatusFailure    = "failed"
)