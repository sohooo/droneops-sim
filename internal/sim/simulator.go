// Simulator orchestrating drones and telemetry ticks
package sim

import (
	"log"
	"time"

	"droneops-sim/internal/config"
	"droneops-sim/internal/telemetry"
)

// TelemetryWriter is an interface to support different output writers.
type TelemetryWriter interface {
	Write(telemetry.TelemetryRow) error
}

// Optional: Writers can also support batch mode
type batchWriter interface {
	WriteBatch([]telemetry.TelemetryRow) error
}

// Simulator orchestrates fleet telemetry generation and writing.
type Simulator struct {
	clusterID    string
	fleets       []DroneFleet
	teleGen      *telemetry.Generator
	writer       TelemetryWriter
	tickInterval time.Duration
}

// DroneFleet holds runtime drones for one fleet.
type DroneFleet struct {
	Model  string
	Drones []*telemetry.Drone
}

// NewSimulator initializes drones from fleet config.
func NewSimulator(clusterID string, cfg *config.SimulationConfig, writer TelemetryWriter, tickInterval time.Duration) *Simulator {
	sim := &Simulator{
		clusterID:    clusterID,
		teleGen:      telemetry.NewGenerator(clusterID),
		writer:       writer,
		tickInterval: tickInterval,
	}

	// Initialize fleets
	for _, fleet := range cfg.Fleets {
		f := DroneFleet{Model: fleet.Model}
		for i := 0; i < fleet.Count; i++ {
			drone := &telemetry.Drone{
				ID:       generateDroneID(fleet.Name, i),
				Model:    fleet.Model,
				Position: telemetry.Position{Lat: cfg.Zones[0].CenterLat, Lon: cfg.Zones[0].CenterLon, Alt: 100},
				Battery:  100,
				Status:   telemetry.StatusOK,
			}
			f.Drones = append(f.Drones, drone)
		}
		sim.fleets = append(sim.fleets, f)
	}
	return sim
}

// Run starts the simulation loop (blocking until stop signal)
func (s *Simulator) Run(stop <-chan struct{}) {
	log.Printf("[Simulator] starting with tick interval %s", s.tickInterval)
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-stop:
			log.Println("[Simulator] stopping...")
			return
		}
	}
}

// tick generates telemetry and writes it.
func (s *Simulator) tick() {
	var batch []telemetry.TelemetryRow
	for _, fleet := range s.fleets {
		for _, drone := range fleet.Drones {
			batch = append(batch, s.teleGen.GenerateTelemetry(drone))
		}
	}

	// Batch support if writer implements WriteBatch
	if bw, ok := s.writer.(batchWriter); ok {
		if err := bw.WriteBatch(batch); err != nil {
			log.Printf("[Simulator] batch write failed: %v", err)
		}
	} else {
		for _, row := range batch {
			if err := s.writer.Write(row); err != nil {
				log.Printf("[Simulator] write failed for drone %s: %v", row.DroneID, err)
			}
		}
	}
}

func generateDroneID(fleetName string, index int) string {
	return fleetName + "-" + time.Now().Format("150405") + "-" + string(rune('A'+index))
}
