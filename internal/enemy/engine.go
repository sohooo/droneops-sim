package enemy

import (
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"droneops-sim/internal/telemetry"
)

// Engine maintains and updates simulated enemy entities.
type Engine struct {
	region  telemetry.Region
	Enemies []*Enemy
}

// NewEngine creates an engine with a given number of enemies in the region.
func NewEngine(count int, region telemetry.Region) *Engine {
	e := &Engine{region: region}
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < count; i++ {
		en := &Enemy{
			ID:         uuid.New().String(),
			Type:       randomType(),
			Position:   randomPosition(region),
			Confidence: 100,
		}
		e.Enemies = append(e.Enemies, en)
	}
	return e
}

func randomType() EnemyType {
	types := []EnemyType{EnemyVehicle, EnemyPerson, EnemyDrone}
	return types[rand.Intn(len(types))]
}

func randomPosition(region telemetry.Region) telemetry.Position {
	angle := rand.Float64() * 2 * math.Pi
	r := rand.Float64() * region.RadiusKM * 1000
	dLat := (r * math.Cos(angle)) / 111000
	dLon := (r * math.Sin(angle)) / (111000 * math.Cos(region.CenterLat*math.Pi/180))
	return telemetry.Position{Lat: region.CenterLat + dLat, Lon: region.CenterLon + dLon, Alt: 0}
}

func randomStep(pos telemetry.Position) telemetry.Position {
	dLat := rand.Float64()*0.001 - 0.0005
	dLon := rand.Float64()*0.001 - 0.0005
	return telemetry.Position{Lat: pos.Lat + dLat, Lon: pos.Lon + dLon, Alt: pos.Alt}
}

// Step moves all enemies slightly using a random walk.
func (e *Engine) Step() {
	for _, en := range e.Enemies {
		en.Position = randomStep(en.Position)
	}
}
