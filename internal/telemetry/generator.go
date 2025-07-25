package telemetry

import (
	"math"
	"math/rand"
	"time"
)

// Generator simulates telemetry for a fleet of drones.
type Generator struct {
	ClusterID string
}

// NewGenerator creates a new telemetry generator for a given cluster.
func NewGenerator(clusterID string) *Generator {
	return &Generator{ClusterID: clusterID}
}

// GenerateTelemetry updates a drone's state and returns a TelemetryRow ready for DB write.
func (g *Generator) GenerateTelemetry(drone *Drone) TelemetryRow {
	// Movement
	drone.Position = randomWalk(drone.Position, drone.Model)

	// Battery drain
	drone.Battery -= batteryDrain(drone.Model)
	if drone.Battery < 0 {
		drone.Battery = 0
	}

	// Status
	if drone.Battery <= 5 {
		drone.Status = StatusFailure
	} else if drone.Battery <= 20 {
		drone.Status = StatusLowBattery
	} else {
		drone.Status = StatusOK
	}

	return TelemetryRow{
		ClusterID:  g.ClusterID,
		DroneID:    drone.ID,
		Lat:        drone.Position.Lat,
		Lon:        drone.Position.Lon,
		Alt:        drone.Position.Alt,
		Battery:    drone.Battery,
		Status:     drone.Status,
		SyncedFrom: "",
		SyncedID:   "",
		SyncedAt:   time.Time{},
		Timestamp:  time.Now().UTC(),
	}
}

// randomWalk moves the drone in a pseudo-random direction, speed depends on model.
func randomWalk(pos Position, model string) Position {
	var speedMin, speedMax float64
	switch model {
	case "small-fpv":
		speedMin, speedMax = 15, 30
	case "medium-uav":
		speedMin, speedMax = 25, 50
	case "large-uav":
		speedMin, speedMax = 20, 40
	default:
		speedMin, speedMax = 15, 25
	}

	heading := rand.Float64() * 2 * math.Pi
	speed := rand.Float64()*(speedMax-speedMin) + speedMin // m/s

	// Convert speed to lat/lon deltas
	deltaLat := (speed * math.Cos(heading)) / 111000
	deltaLon := (speed * math.Sin(heading)) / (111000 * math.Cos(pos.Lat*math.Pi/180))
	altDelta := rand.Float64()*2 - 1 // ±1m

	return Position{
		Lat: pos.Lat + deltaLat,
		Lon: pos.Lon + deltaLon,
		Alt: math.Max(0, pos.Alt+altDelta),
	}
}

// batteryDrain returns battery consumption per tick based on model.
func batteryDrain(model string) float64 {
	switch model {
	case "small-fpv":
		return 0.5
	case "medium-uav":
		return 0.3
	case "large-uav":
		return 0.2
	default:
		return 0.4
	}
}