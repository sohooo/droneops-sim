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

const (
	sensorErrorMaxOffset   = 0.005 // degrees (~500m)
	interceptLateralMeters = 50.0  // lateral spacing between intercept points
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

// MapMission represents a mission region for annotations on the map.
type MapMission struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	RadiusKM float64 `json:"radius_km"`
}

// MapData aggregates drone, enemy, and mission positions for the map view.
type MapData struct {
	Drones   []MapDrone   `json:"drones"`
	Enemies  []MapEnemy   `json:"enemies"`
	Missions []MapMission `json:"missions"`
}

// ObserverEvent represents a mission event used by analyst tools.
type ObserverEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Details   string    `json:"details,omitempty"`
}

// Simulator orchestrates fleet telemetry generation and writing.
type Simulator struct {
	clusterID            string
	fleets               []DroneFleet
	teleGen              *telemetry.Generator
	writer               TelemetryWriter
	detectionWriter      DetectionWriter
	enemyEng             *enemy.Engine
	tickInterval         time.Duration
	chaosMode            bool
	cfg                  *config.SimulationConfig
	followConfidence     float64
	detectionRadiusM     float64
	sensorNoise          float64
	terrainOcclusion     float64
	weatherImpact        float64
	swarmResponses       map[string]int
	missionCriticality   int
	enemyPrevPositions   map[string]telemetry.Position
	commLoss             float64
	bandwidthLimit       int
	messagesSent         int
	enemyFollowers       map[string][]string
	droneAssignments     map[string]string
	enemyFollowerTargets map[string]int
	enemyObjects         map[string]*enemy.Enemy
	droneIndex           map[string]*telemetry.Drone
	droneFleet           map[string]*DroneFleet
	observerEvents       []ObserverEvent
	observerIdx          int
	observerPerspective  string
	mu                   sync.Mutex
	rand                 *rand.Rand
	now                  func() time.Time
}

// DroneFleet holds runtime drones for one fleet.
type DroneFleet struct {
	Name   string
	Model  string
	Drones []*telemetry.Drone
}

// NewSimulator initializes drones from fleet config.
func NewSimulator(clusterID string, cfg *config.SimulationConfig, writer TelemetryWriter, dWriter DetectionWriter, tickInterval time.Duration, r *rand.Rand, now func() time.Time) *Simulator {
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	if now == nil {
		now = time.Now
	}
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
		clusterID:            clusterID,
		teleGen:              telemetry.NewGenerator(clusterID, r, now),
		writer:               writer,
		detectionWriter:      dWriter,
		tickInterval:         tickInterval,
		cfg:                  cfg,
		followConfidence:     cfg.FollowConfidence,
		detectionRadiusM:     radius,
		sensorNoise:          sNoise,
		terrainOcclusion:     terrain,
		weatherImpact:        weather,
		swarmResponses:       cfg.SwarmResponses,
		missionCriticality:   crit,
		enemyPrevPositions:   make(map[string]telemetry.Position),
		commLoss:             cfg.CommunicationLoss,
		bandwidthLimit:       cfg.BandwidthLimit,
		enemyFollowers:       make(map[string][]string),
		droneAssignments:     make(map[string]string),
		enemyFollowerTargets: make(map[string]int),
		enemyObjects:         make(map[string]*enemy.Enemy),
		droneIndex:           make(map[string]*telemetry.Drone),
		droneFleet:           make(map[string]*DroneFleet),
		rand:                 r,
		now:                  now,
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

	for i := range sim.fleets {
		f := &sim.fleets[i]
		for _, d := range f.Drones {
			sim.droneIndex[d.ID] = d
			sim.droneFleet[d.ID] = f
		}
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
	sim.enemyEng = enemy.NewEngine(count, regions, r)

	return sim
}

// ToggleChaos flips chaos mode on or off and returns the new state.
func (s *Simulator) ToggleChaos() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chaosMode = !s.chaosMode
	s.logObserverEvent("toggle_chaos", fmt.Sprintf("enabled=%v", s.chaosMode))
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
	s.logObserverEvent("launch_swarm", fmt.Sprintf("model=%s count=%d", model, count))
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
				Timestamp: s.now().UTC(),
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
	var missions []MapMission
	if s.cfg != nil {
		for _, m := range s.cfg.Missions {
			missions = append(missions, MapMission{
				ID:       m.ID,
				Name:     m.Name,
				Lat:      m.Region.CenterLat,
				Lon:      m.Region.CenterLon,
				RadiusKM: m.Region.RadiusKM,
			})
		}
	}
	return MapData{Drones: drones, Enemies: enemies, Missions: missions}
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
