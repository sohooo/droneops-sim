// Simulator orchestrating drones and telemetry ticks
package sim

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"droneops-sim/internal/config"
	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"

	"github.com/google/uuid"
)

// TelemetryWriter is an interface to support different output writers.
type TelemetryWriter interface {
	Write(telemetry.TelemetryRow) error
}

// DetectionWriter handles enemy detection events.
type DetectionWriter interface {
	WriteDetection(enemy.DetectionRow) error
}

// Optional: Detection writers may support batch mode
type batchDetectionWriter interface {
	WriteDetections([]enemy.DetectionRow) error
}

// Optional: Writers can also support batch mode
type batchWriter interface {
	WriteBatch([]telemetry.TelemetryRow) error
}

// Simulator orchestrates fleet telemetry generation and writing.
type Simulator struct {
	clusterID        string
	fleets           []DroneFleet
	teleGen          *telemetry.Generator
	writer           TelemetryWriter
	detectionWriter  DetectionWriter
	enemyEng         *enemy.Engine
	tickInterval     time.Duration
	chaosMode        bool
	cfg              *config.SimulationConfig
	followConfidence float64
	swarmResponses   map[string]int
	mu               sync.Mutex
}

// DroneFleet holds runtime drones for one fleet.
type DroneFleet struct {
	Name   string
	Model  string
	Drones []*telemetry.Drone
}

// NewSimulator initializes drones from fleet config.
func NewSimulator(clusterID string, cfg *config.SimulationConfig, writer TelemetryWriter, dWriter DetectionWriter, tickInterval time.Duration) *Simulator {
	sim := &Simulator{
		clusterID:        clusterID,
		teleGen:          telemetry.NewGenerator(clusterID),
		writer:           writer,
		detectionWriter:  dWriter,
		tickInterval:     tickInterval,
		cfg:              cfg,
		followConfidence: cfg.FollowConfidence,
		swarmResponses:   cfg.SwarmResponses,
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
				ID:                 generateDroneID(fleet.Name, i),
				Model:              fleet.Model,
				Position:           telemetry.Position{Lat: cfg.Zones[0].CenterLat, Lon: cfg.Zones[0].CenterLon, Alt: 100},
				Battery:            100,
				Status:             telemetry.StatusOK,
				SensorErrorRate:    fleet.Behavior.SensorErrorRate,
				DropoutRate:        fleet.Behavior.DropoutRate,
				BatteryAnomalyRate: fleet.Behavior.BatteryAnomalyRate,
			}
			f.Drones = append(f.Drones, drone)
		}
		sim.fleets = append(sim.fleets, f)
	}

	// Initialize enemy engine with a few entities in the first zone
	sim.enemyEng = enemy.NewEngine(3, telemetry.Region{
		Name:      cfg.Zones[0].Name,
		CenterLat: cfg.Zones[0].CenterLat,
		CenterLon: cfg.Zones[0].CenterLon,
		RadiusKM:  cfg.Zones[0].RadiusKM,
	})

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
	var detections []enemy.DetectionRow

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.enemyEng != nil {
		s.enemyEng.Step()
	}

	for _, fleet := range s.fleets {
		for _, drone := range fleet.Drones {
			row := s.teleGen.GenerateTelemetry(drone)
			if rand.Float64() < drone.SensorErrorRate {
				row.Lat += rand.Float64()*0.01 - 0.005
				row.Lon += rand.Float64()*0.01 - 0.005
			}
			if rand.Float64() < drone.BatteryAnomalyRate {
				drop := rand.Float64()*20 + 10
				drone.Battery -= drop
				if drone.Battery < 0 {
					drone.Battery = 0
				}
				row.Battery = drone.Battery
			}
			if rand.Float64() < drone.DropoutRate {
				continue
			}
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

			if s.enemyEng != nil {
				for _, en := range s.enemyEng.Enemies {
					dist := distanceMeters(drone.Position.Lat, drone.Position.Lon, en.Position.Lat, en.Position.Lon)
					if dist <= 1000 {
						conf := 100 * (1 - dist/1000)
						d := enemy.DetectionRow{
							ClusterID:  s.clusterID,
							DroneID:    drone.ID,
							EnemyID:    en.ID,
							EnemyType:  en.Type,
							Lat:        en.Position.Lat,
							Lon:        en.Position.Lon,
							Alt:        en.Position.Alt,
							Confidence: conf,
							Timestamp:  time.Now().UTC(),
						}
						detections = append(detections, d)
						if conf >= s.followConfidence {
							s.assignFollower(&fleet, drone, en)
						}
					}
				}
			}
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

	// Write enemy detections if any
	if len(detections) > 0 && s.detectionWriter != nil {
		if bw, ok := s.detectionWriter.(batchDetectionWriter); ok {
			if err := bw.WriteDetections(detections); err != nil {
				log.Printf("[Simulator] detection batch write failed: %v", err)
			}
		} else {
			for _, d := range detections {
				if err := s.detectionWriter.WriteDetection(d); err != nil {
					log.Printf("[Simulator] detection write failed: %v", err)
				}
			}
		}
	}
}

func (s *Simulator) assignFollower(fleet *DroneFleet, detecting *telemetry.Drone, en *enemy.Enemy) {
	target := en.Position
	count, ok := s.swarmResponses[detecting.MovementPattern]
	if !ok {
		count = 0
	}
	if count == 0 {
		cp := target
		detecting.FollowTarget = &cp
		return
	}
	if count < 0 {
		for _, d := range fleet.Drones {
			if d.FollowTarget == nil {
				cp := target
				d.FollowTarget = &cp
			}
		}
		return
	}
	assigned := 0
	for _, d := range fleet.Drones {
		if d == detecting {
			continue
		}
		if d.FollowTarget == nil {
			cp := target
			d.FollowTarget = &cp
			assigned++
			if assigned >= count {
				break
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
	fleetName := model // Use model name directly as fleet name
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

// TelemetrySnapshot returns the latest state for all drones.
func (s *Simulator) TelemetrySnapshot() []telemetry.TelemetryRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rows []telemetry.TelemetryRow
	for _, fleet := range s.fleets {
		for _, drone := range fleet.Drones {
			rows = append(rows, telemetry.TelemetryRow{
				ClusterID: s.clusterID,
				DroneID:   drone.ID,
				Lat:       drone.Position.Lat,
				Lon:       drone.Position.Lon,
				Alt:       drone.Position.Alt,
				Battery:   drone.Battery,
				Status:    drone.Status,
				Follow:    drone.FollowTarget != nil,
				Timestamp: time.Now().UTC(),
			})
		}
	}
	return rows
}

func generateDroneID(fleetName string, index int) string {
	// Include the drone's index along with a UUID to guarantee uniqueness
	id := uuid.New().String()
	return fmt.Sprintf("%s-%d-%s", fleetName, index, id)
}

// distanceMeters calculates the haversine distance between two lat/lon points.
func distanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}
