// Simulator orchestrating drones and telemetry ticks
package sim

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
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

// MapDrone is used for the 3D map data response.
type MapDrone struct {
	ID        string   `json:"id"`
	Lat       float64  `json:"lat"`
	Lon       float64  `json:"lon"`
	Alt       float64  `json:"alt"`
	Battery   float64  `json:"battery"`
	FollowLat *float64 `json:"follow_lat,omitempty"`
	FollowLon *float64 `json:"follow_lon,omitempty"`
	FollowAlt *float64 `json:"follow_alt,omitempty"`
}

// MapEnemy represents an enemy entity for the 3D map.
type MapEnemy struct {
	ID   string          `json:"id"`
	Type enemy.EnemyType `json:"type"`
	Lat  float64         `json:"lat"`
	Lon  float64         `json:"lon"`
	Alt  float64         `json:"alt"`
}

// MapData aggregates drone and enemy positions for the map view.
type MapData struct {
	Drones  []MapDrone `json:"drones"`
	Enemies []MapEnemy `json:"enemies"`
}

// Simulator orchestrates fleet telemetry generation and writing.
type Simulator struct {
	clusterID          string
	fleets             []DroneFleet
	teleGen            *telemetry.Generator
	writer             TelemetryWriter
	detectionWriter    DetectionWriter
	enemyEng           *enemy.Engine
	tickInterval       time.Duration
	chaosMode          bool
	cfg                *config.SimulationConfig
	followConfidence   float64
	detectionRadiusM   float64
	sensorNoise        float64
	terrainOcclusion   float64
	weatherImpact      float64
	swarmResponses     map[string]int
	missionCriticality int
	mu                 sync.Mutex
}

// DroneFleet holds runtime drones for one fleet.
type DroneFleet struct {
	Name   string
	Model  string
	Drones []*telemetry.Drone
}

// NewSimulator initializes drones from fleet config.
func NewSimulator(clusterID string, cfg *config.SimulationConfig, writer TelemetryWriter, dWriter DetectionWriter, tickInterval time.Duration) *Simulator {
	radius := cfg.DetectionRadiusM
	if radius <= 0 {
		radius = 1000
	}
	sNoise := cfg.SensorNoise
	if sNoise < 0 {
		sNoise = 0
	}
	terrain := cfg.TerrainOcclusion
	if terrain < 0 {
		terrain = 0
	} else if terrain > 1 {
		terrain = 1
	}
	weather := cfg.WeatherImpact
	if weather < 0 {
		weather = 0
	} else if weather > 1 {
		weather = 1
	}
	crit := 0
	switch strings.ToLower(cfg.MissionCriticality) {
	case "medium":
		crit = 1
	case "high":
		crit = 2
	}
	sim := &Simulator{
		clusterID:          clusterID,
		teleGen:            telemetry.NewGenerator(clusterID),
		writer:             writer,
		detectionWriter:    dWriter,
		tickInterval:       tickInterval,
		cfg:                cfg,
		followConfidence:   cfg.FollowConfidence,
		detectionRadiusM:   radius,
		sensorNoise:        sNoise,
		terrainOcclusion:   terrain,
		weatherImpact:      weather,
		swarmResponses:     cfg.SwarmResponses,
		missionCriticality: crit,
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
				ID:              generateDroneID(fleet.Name, i),
				Model:           fleet.Model,
				Position:        telemetry.Position{Lat: cfg.Zones[0].CenterLat, Lon: cfg.Zones[0].CenterLon, Alt: 100},
				Battery:         100,
				Status:          telemetry.StatusOK,
				MovementPattern: fleet.MovementPattern,
				HomeRegion: telemetry.Region{
					Name:      cfg.Zones[0].Name,
					CenterLat: cfg.Zones[0].CenterLat,
					CenterLon: cfg.Zones[0].CenterLon,
					RadiusKM:  cfg.Zones[0].RadiusKM,
				},
				SensorErrorRate:    fleet.Behavior.SensorErrorRate,
				DropoutRate:        fleet.Behavior.DropoutRate,
				BatteryAnomalyRate: fleet.Behavior.BatteryAnomalyRate,
			}
			f.Drones = append(f.Drones, drone)
		}
		sim.fleets = append(sim.fleets, f)
	}

	// Initialize enemy engine across all zones
	count := cfg.EnemyCount
	if count <= 0 {
		count = 3
	}
	regions := make([]telemetry.Region, len(cfg.Zones))
	for i, z := range cfg.Zones {
		regions[i] = telemetry.Region{
			Name:      z.Name,
			CenterLat: z.CenterLat,
			CenterLon: z.CenterLon,
			RadiusKM:  z.RadiusKM,
		}
	}
	sim.enemyEng = enemy.NewEngine(count, regions)

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

	var allDrones []*telemetry.Drone
	for _, f := range s.fleets {
		allDrones = append(allDrones, f.Drones...)
	}
	if s.enemyEng != nil {
		s.enemyEng.Step(allDrones)
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
					if dist <= s.detectionRadiusM {
						conf := 100 * (1 - dist/s.detectionRadiusM)
						conf *= 1 - s.terrainOcclusion
						conf *= 1 - s.weatherImpact
						if s.sensorNoise > 0 {
							conf += rand.NormFloat64() * s.sensorNoise * conf
						}
						if conf < 0 {
							conf = 0
						} else if conf > 100 {
							conf = 100
						}
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
							s.assignFollower(&fleet, drone, en, conf)
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

func (s *Simulator) assignFollower(fleet *DroneFleet, detecting *telemetry.Drone, en *enemy.Enemy, conf float64) {
	target := en.Position
	count, ok := s.swarmResponses[detecting.MovementPattern]
	if !ok {
		count = 0
	}
	if count >= 0 {
		if conf > 90 {
			count++
		}
		switch en.Type {
		case enemy.EnemyVehicle, enemy.EnemyDrone:
			count++
		case enemy.EnemyDecoy:
			if count > 0 {
				count--
			}
		}
		count += s.missionCriticality
	}
	if count == 0 {
		cp := target
		detecting.FollowTarget = &cp
		s.rebalanceFormation(fleet)
		return
	}
	if count < 0 {
		for _, d := range fleet.Drones {
			if d.FollowTarget == nil {
				cp := target
				d.FollowTarget = &cp
			}
		}
		s.rebalanceFormation(fleet)
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
	if assigned == 0 {
		cp := target
		detecting.FollowTarget = &cp
	}
	s.rebalanceFormation(fleet)
}

func (s *Simulator) rebalanceFormation(fleet *DroneFleet) {
	var remaining []*telemetry.Drone
	for _, d := range fleet.Drones {
		if d.FollowTarget == nil {
			remaining = append(remaining, d)
		}
	}
	n := len(remaining)
	if n == 0 {
		return
	}
	region := remaining[0].HomeRegion
	radius := region.RadiusKM * 1000 * 0.5
	for i, d := range remaining {
		angle := float64(i) / float64(n) * 2 * math.Pi
		deltaLat := (radius * math.Cos(angle)) / 111000
		deltaLon := (radius * math.Sin(angle)) / (111000 * math.Cos(region.CenterLat*math.Pi/180))
		d.HomeRegion.CenterLat = region.CenterLat + deltaLat
		d.HomeRegion.CenterLon = region.CenterLon + deltaLon
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

// MapSnapshot returns simplified drone and enemy data for the 3D map.
func (s *Simulator) MapSnapshot() MapData {
	s.mu.Lock()
	defer s.mu.Unlock()
	var drones []MapDrone
	for _, fleet := range s.fleets {
		for _, d := range fleet.Drones {
			md := MapDrone{
				ID:      d.ID,
				Lat:     d.Position.Lat,
				Lon:     d.Position.Lon,
				Alt:     d.Position.Alt,
				Battery: d.Battery,
			}
			if d.FollowTarget != nil {
				md.FollowLat = &d.FollowTarget.Lat
				md.FollowLon = &d.FollowTarget.Lon
				md.FollowAlt = &d.FollowTarget.Alt
			}
			drones = append(drones, md)
		}
	}
	var enemies []MapEnemy
	if s.enemyEng != nil {
		for _, e := range s.enemyEng.Enemies {
			enemies = append(enemies, MapEnemy{
				ID:   e.ID,
				Type: e.Type,
				Lat:  e.Position.Lat,
				Lon:  e.Position.Lon,
				Alt:  e.Position.Alt,
			})
		}
	}
	return MapData{Drones: drones, Enemies: enemies}
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
