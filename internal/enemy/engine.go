package enemy

import (
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"droneops-sim/internal/telemetry"
)

const nearDroneDistThreshold = 0.005 // degrees, ~500m

// Engine maintains and updates simulated enemy entities.
type Engine struct {
	regions            []telemetry.Region
	Enemies            []*Enemy
	rand               *rand.Rand
	randFloat          func() float64
	MaxDecoysPerParent int
	DecoyLifespan      time.Duration
}

// NewEngine creates an engine with a given number of enemies per region.
func NewEngine(count int, regions []telemetry.Region, r *rand.Rand) *Engine {
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	e := &Engine{regions: regions, rand: r, randFloat: r.Float64}
	for _, reg := range regions {
		for i := 0; i < count; i++ {
			en := &Enemy{
				ID:         uuid.New().String(),
				Type:       randomType(r),
				Position:   randomPosition(r, reg),
				Confidence: 100,
				Region:     reg,
				Status:     EnemyActive,
			}
			e.Enemies = append(e.Enemies, en)
		}
	}
	return e
}

func randomType(r *rand.Rand) EnemyType {
	types := []EnemyType{EnemyVehicle, EnemyPerson, EnemyDrone}
	return types[r.Intn(len(types))]
}

func randomPosition(r *rand.Rand, region telemetry.Region) telemetry.Position {
	angle := r.Float64() * 2 * math.Pi
	dist := r.Float64() * region.RadiusKM * 1000
	dLat := (dist * math.Cos(angle)) / 111000
	dLon := (dist * math.Sin(angle)) / (111000 * math.Cos(region.CenterLat*math.Pi/180))
	return telemetry.Position{Lat: region.CenterLat + dLat, Lon: region.CenterLon + dLon, Alt: 0}
}

func randomStep(r *rand.Rand, pos telemetry.Position) telemetry.Position {
	dLat := r.Float64()*0.001 - 0.0005
	dLon := r.Float64()*0.001 - 0.0005
	return telemetry.Position{Lat: pos.Lat + dLat, Lon: pos.Lon + dLon, Alt: pos.Alt}
}

func distance(a, b telemetry.Position) float64 {
	dLat := a.Lat - b.Lat
	dLon := a.Lon - b.Lon
	return math.Sqrt(dLat*dLat + dLon*dLon)
}

func moveAway(r *rand.Rand, pos, threat telemetry.Position) telemetry.Position {
	vecLat := pos.Lat - threat.Lat
	vecLon := pos.Lon - threat.Lon
	norm := math.Sqrt(vecLat*vecLat + vecLon*vecLon)
	if norm == 0 {
		return randomStep(r, pos)
	}
	factor := 0.001 / norm
	return telemetry.Position{Lat: pos.Lat + vecLat*factor, Lon: pos.Lon + vecLon*factor, Alt: pos.Alt}
}

func moveTowards(pos, target telemetry.Position) telemetry.Position {
	vecLat := target.Lat - pos.Lat
	vecLon := target.Lon - pos.Lon
	norm := math.Sqrt(vecLat*vecLat + vecLon*vecLon)
	if norm == 0 {
		return pos
	}
	factor := 0.001 / norm
	return telemetry.Position{Lat: pos.Lat + vecLat*factor, Lon: pos.Lon + vecLon*factor, Alt: pos.Alt}
}

func (e *Engine) spawnDecoy(parent *Enemy) {
	decoy := &Enemy{
		ID:         uuid.New().String(),
		Type:       EnemyDecoy,
		Position:   randomStep(e.rand, parent.Position),
		Confidence: parent.Confidence * 0.5,
		Region:     parent.Region,
		Status:     EnemyActive,
		ParentID:   parent.ID,
	}
	if e.DecoyLifespan > 0 {
		decoy.ExpiresAt = time.Now().Add(e.DecoyLifespan)
	}
	e.Enemies = append(e.Enemies, decoy)
}

func nearestDrone(pos telemetry.Position, drones []*telemetry.Drone) (*telemetry.Drone, float64) {
	var closest *telemetry.Drone
	min := math.MaxFloat64
	for _, d := range drones {
		dist := distance(pos, d.Position)
		if dist < min {
			min = dist
			closest = d
		}
	}
	return closest, min
}

func nearestEnemy(cur *Enemy, enemies []*Enemy) (*Enemy, float64) {
	var closest *Enemy
	min := math.MaxFloat64
	for _, e := range enemies {
		if e == cur {
			continue
		}
		dist := distance(cur.Position, e.Position)
		if dist < min {
			min = dist
			closest = e
		}
	}
	return closest, min
}

func (e *Engine) respondToNearbyDrone(en *Enemy, drones []*telemetry.Drone) bool {
	nearest, dist := nearestDrone(en.Position, drones)
	if nearest != nil && dist < nearDroneDistThreshold {
		en.Position = moveAway(e.rand, en.Position, nearest.Position)
		if e.randFloat() < 0.3 {
			count := 0
			if e.MaxDecoysPerParent > 0 {
				for _, other := range e.Enemies {
					if other.Type == EnemyDecoy && other.ParentID == en.ID {
						count++
					}
				}
			}
			if e.MaxDecoysPerParent == 0 || count < e.MaxDecoysPerParent {
				e.spawnDecoy(en)
			}
		}
		return true
	}
	return false
}

func (e *Engine) pursueAnotherEnemy(en *Enemy) bool {
	if e.randFloat() < 0.1 && len(e.Enemies) > 1 {
		other, _ := nearestEnemy(en, e.Enemies)
		if other != nil {
			en.Position = moveTowards(en.Position, other.Position)
			return true
		}
	}
	return false
}

func (e *Engine) handleRegionBounds(en *Enemy) {
	if en.Region.RadiusKM > 0 {
		center := telemetry.Position{Lat: en.Region.CenterLat, Lon: en.Region.CenterLon}
		if distance(en.Position, center) > en.Region.RadiusKM/111 {
			en.Position = randomPosition(e.rand, en.Region)
		}
	}
}

// Step updates enemies based on drone positions and tactics.
func (e *Engine) Step(drones []*telemetry.Drone) {
	if e.randFloat == nil {
		e.randFloat = e.rand.Float64
	}
	now := time.Now()
	filtered := e.Enemies[:0]
	for _, en := range e.Enemies {
		if en.Type == EnemyDecoy && !en.ExpiresAt.IsZero() && now.After(en.ExpiresAt) {
			continue
		}
		filtered = append(filtered, en)
	}
	e.Enemies = filtered
	for _, en := range e.Enemies {
		handled := e.respondToNearbyDrone(en, drones)
		if !handled {
			handled = e.pursueAnotherEnemy(en)
		}
		if !handled {
			en.Position = randomStep(e.rand, en.Position)
		}
		e.handleRegionBounds(en)
	}
}
