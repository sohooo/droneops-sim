// Simulator orchestrating drones and telemetry ticks
package sim

import (
	"log"
	"math/rand"
	"sync"
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
	chaosMode    bool
	cfg          *config.SimulationConfig
	mu           sync.Mutex
}

// DroneFleet holds runtime drones for one fleet.
type DroneFleet struct {
	Name   string
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
		cfg:          cfg,
	}

	// Check if zones are defined
	if len(cfg.Zones) == 0 {
		log.Panic("No zones defined in the configuration")
	}

	// Initialize fleets
	for _, fleet := range cfg.Fleets {
		f := DroneFleet{Name: fleet.Name, Model: fleet.Model}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, fleet := range s.fleets {
		for _, drone := range fleet.Drones {
			row := s.teleGen.GenerateTelemetry(drone)
			if s.chaosMode {
				if rand.Float64() < 0.1 {
					row.Status = telemetry.StatusFailure
					drone.Status = telemetry.StatusFailure
				}
				extra := rand.Float64() * 5
				drone.Battery -= extra
				if drone.Battery < 0 {
					drone.Battery = 0
				}
				row.Battery = drone.Battery
			}
			batch = append(batch, row)
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

// ToggleChaos flips chaos mode on or off and returns the new state.
func (s *Simulator) ToggleChaos() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chaosMode = !s.chaosMode
	return s.chaosMode
}

// Chaos returns whether chaos mode is active.
func (s *Simulator) Chaos() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.chaosMode
}

// LaunchSwarm adds a new fleet of drones of the given model and count.
func (s *Simulator) LaunchSwarm(model string, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	region := s.cfg.Zones[0]
	fleetName := model + "-" + time.Now().Format("150405")
	f := DroneFleet{Name: fleetName, Model: model}
	for i := 0; i < count; i++ {
		drone := &telemetry.Drone{
			ID:       generateDroneID(fleetName, i),
			Model:    model,
			Position: telemetry.Position{Lat: region.CenterLat, Lon: region.CenterLon, Alt: 100},
			Battery:  100,
			Status:   telemetry.StatusOK,
		}
		f.Drones = append(f.Drones, drone)
	}
	s.fleets = append(s.fleets, f)
}

// FleetHealth summarizes status counts per fleet.
type FleetHealth struct {
	Name       string `json:"name"`
	Total      int    `json:"total"`
	LowBattery int    `json:"low_battery"`
	Failed     int    `json:"failed"`
}

// Health returns aggregated health information for all fleets.
func (s *Simulator) Health() []FleetHealth {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []FleetHealth
	for _, f := range s.fleets {
		h := FleetHealth{Name: f.Name, Total: len(f.Drones)}
		for _, d := range f.Drones {
			switch d.Status {
			case telemetry.StatusFailure:
				h.Failed++
			case telemetry.StatusLowBattery:
				h.LowBattery++
			}
		}
		result = append(result, h)
	}
	return result
}

// GetConfig returns the simulation configuration.
func (s *Simulator) GetConfig() *config.SimulationConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func generateDroneID(fleetName string, index int) string {
	return fleetName + "-" + time.Now().Format("150405") + "-" + string(rune('A'+index))
}
