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
	var strategy MovementStrategy

	// Select movement strategy based on drone's movement pattern
	switch drone.MovementPattern {
	case "patrol":
		strategy = PatrolMovement{}
	case "point-to-point":
		strategy = PointToPointMovement{}
	case "loiter":
		strategy = LoiterMovement{}
	default:
		strategy = RandomWalkMovement{} // Implement RandomWalkMovement similarly
	}

	// Update drone's position using the selected strategy
	drone.Position = strategy.Move(drone, drone.HomeRegion, drone.Waypoints)

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

// MovementStrategy defines the interface for drone movement.
type MovementStrategy interface {
	Move(drone *Drone, region Region, waypoints []Position) Position
}

// PatrolMovement implements circular movement around the home region's center.
type PatrolMovement struct{}

func (p PatrolMovement) Move(drone *Drone, region Region, waypoints []Position) Position {
	radius := region.RadiusKM * 1000 * 0.99 // Scale radius slightly to ensure position stays within bounds
	angle := rand.Float64() * 2 * math.Pi
	deltaLat := (radius * math.Cos(angle)) / 111000
	deltaLon := (radius * math.Sin(angle)) / (111000 * math.Cos(region.CenterLat*math.Pi/180))
	return Position{
		Lat: region.CenterLat + deltaLat,
		Lon: region.CenterLon + deltaLon,
		Alt: drone.Position.Alt,
	}
}

// PointToPointMovement implements movement between predefined waypoints.
type PointToPointMovement struct{}

func (p PointToPointMovement) Move(drone *Drone, region Region, waypoints []Position) Position {
	if len(waypoints) == 0 {
		return drone.Position
	}
	target := waypoints[rand.Intn(len(waypoints))]
	deltaLat := (target.Lat - drone.Position.Lat) / 10 // Gradual movement
	deltaLon := (target.Lon - drone.Position.Lon) / 10
	return Position{
		Lat: drone.Position.Lat + deltaLat,
		Lon: drone.Position.Lon + deltaLon,
		Alt: drone.Position.Alt,
	}
}

// LoiterMovement implements hovering near the home region's center.
type LoiterMovement struct{}

func (l LoiterMovement) Move(drone *Drone, region Region, waypoints []Position) Position {
	deltaLat := rand.Float64()*0.0001 - 0.00005 // Small random movement
	deltaLon := rand.Float64()*0.0001 - 0.00005
	return Position{
		Lat: region.CenterLat + deltaLat,
		Lon: region.CenterLon + deltaLon,
		Alt: drone.Position.Alt,
	}
}

// RandomWalkMovement implements random movement within the region.
type RandomWalkMovement struct{}

func (r RandomWalkMovement) Move(drone *Drone, region Region, waypoints []Position) Position {
	var speedMin, speedMax float64 // speed range, in meters
	switch drone.Model {
	case "small-fpv":
		speedMin, speedMax = 15, 30
	case "medium-uav":
		speedMin, speedMax = 25, 50
	case "large-uav":
		speedMin, speedMax = 20, 40
	default:
		speedMin, speedMax = 15, 25
	}

	// Random heading (direction) in radians (0 to 2π)
	heading := rand.Float64() * 2 * math.Pi

	// Random speed within the range for the drone model
	speed := rand.Float64()*(speedMax-speedMin) + speedMin // m/s

	// Convert speed and heading into latitude and longitude deltas
	deltaLat := (speed * math.Cos(heading)) / 111000
	deltaLon := (speed * math.Sin(heading)) / (111000 * math.Cos(drone.Position.Lat*math.Pi/180))

	// Altitude delta: random change between -1m and +1m
	altDelta := rand.Float64()*2 - 1 // ±1m

	// Return the new position, ensuring altitude is non-negative
	return Position{
		Lat: drone.Position.Lat + deltaLat,
		Lon: drone.Position.Lon + deltaLon,
		Alt: math.Max(0, drone.Position.Alt+altDelta),
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
