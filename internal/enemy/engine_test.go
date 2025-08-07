package enemy

import (
	"math/rand"
	"testing"
	"time"

	"droneops-sim/internal/telemetry"
)

func TestEngine_EvasiveManeuver(t *testing.T) {
	eng := &Engine{
		regions:   []telemetry.Region{{CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Enemies:   []*Enemy{{ID: "e", Type: EnemyVehicle, Position: telemetry.Position{Lat: 0, Lon: 0}, Confidence: 100, Region: telemetry.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}}},
		rand:      rand.New(rand.NewSource(1)),
		randFloat: func() float64 { return 0.9 },
	}
	drone := &telemetry.Drone{Position: telemetry.Position{Lat: 0, Lon: 0}}
	eng.Step([]*telemetry.Drone{drone})
	if distance(eng.Enemies[0].Position, drone.Position) == 0 {
		t.Fatalf("expected enemy to move away from drone")
	}
}

func TestEngine_SpawnMultipleRegions(t *testing.T) {
	regions := []telemetry.Region{{CenterLat: 0, CenterLon: 0, RadiusKM: 1}, {CenterLat: 1, CenterLon: 1, RadiusKM: 1}}
	eng := NewEngine(1, regions, rand.New(rand.NewSource(1)))
	if len(eng.Enemies) != 2 {
		t.Fatalf("expected 2 enemies, got %d", len(eng.Enemies))
	}
	foundA, foundB := false, false
	for _, e := range eng.Enemies {
		if distance(e.Position, telemetry.Position{Lat: regions[0].CenterLat, Lon: regions[0].CenterLon}) < regions[0].RadiusKM/111 {
			foundA = true
		}
		if distance(e.Position, telemetry.Position{Lat: regions[1].CenterLat, Lon: regions[1].CenterLon}) < regions[1].RadiusKM/111 {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("enemies not spawned in all regions")
	}
}

func TestEngine_SpawnDecoy(t *testing.T) {
	eng := &Engine{
		regions:   []telemetry.Region{{CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Enemies:   []*Enemy{{ID: "e", Type: EnemyVehicle, Position: telemetry.Position{Lat: 0, Lon: 0}, Confidence: 100, Region: telemetry.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}}},
		rand:      rand.New(rand.NewSource(1)),
		randFloat: func() float64 { return 0.1 },
	}
	drone := &telemetry.Drone{Position: telemetry.Position{Lat: 0, Lon: 0}}
	eng.Step([]*telemetry.Drone{drone})
	if len(eng.Enemies) != 2 {
		t.Fatalf("expected decoy to spawn")
	}
	decoy := eng.Enemies[1]
	if decoy.Type != EnemyDecoy {
		t.Fatalf("expected decoy type, got %s", decoy.Type)
	}
}

func TestEngine_PursueEnemy(t *testing.T) {
	region := telemetry.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}
	e1 := &Enemy{ID: "e1", Type: EnemyPerson, Position: telemetry.Position{Lat: 0, Lon: 0}, Region: region}
	e2 := &Enemy{ID: "e2", Type: EnemyVehicle, Position: telemetry.Position{Lat: 0.001, Lon: 0}, Region: region}
	eng := &Engine{regions: []telemetry.Region{region}, Enemies: []*Enemy{e1, e2}, rand: rand.New(rand.NewSource(1)), randFloat: func() float64 { return 0 }}
	before := distance(e1.Position, e2.Position)
	eng.Step(nil)
	after := distance(e1.Position, e2.Position)
	if after >= before {
		t.Fatalf("expected enemy to move towards another enemy")
	}
}

func TestEngine_HandleRegionBounds(t *testing.T) {
	region := telemetry.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}
	en := &Enemy{ID: "e", Type: EnemyPerson, Position: telemetry.Position{Lat: 0.02, Lon: 0.02}, Region: region}
	eng := &Engine{regions: []telemetry.Region{region}, Enemies: []*Enemy{en}, rand: rand.New(rand.NewSource(1)), randFloat: rand.Float64}
	eng.Step(nil)
	center := telemetry.Position{Lat: region.CenterLat, Lon: region.CenterLon}
	if distance(en.Position, center) > region.RadiusKM/111 {
		t.Fatalf("expected enemy to be within region bounds")
	}
}

func TestEngine_DeterministicStep(t *testing.T) {
	region := telemetry.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}
	r1 := rand.New(rand.NewSource(1))
	r2 := rand.New(rand.NewSource(1))
	eng1 := NewEngine(1, []telemetry.Region{region}, r1)
	eng2 := NewEngine(1, []telemetry.Region{region}, r2)
	eng1.Step(nil)
	eng2.Step(nil)
	p1 := eng1.Enemies[0].Position
	p2 := eng2.Enemies[0].Position
	if p1.Lat != p2.Lat || p1.Lon != p2.Lon || p1.Alt != p2.Alt {
		t.Fatalf("expected deterministic positions, got %#v and %#v", p1, p2)
	}
}

func TestEngine_DecoyCap(t *testing.T) {
	region := telemetry.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}
	parent := &Enemy{ID: "p", Type: EnemyVehicle, Position: telemetry.Position{Lat: 0, Lon: 0}, Region: region}
	eng := &Engine{
		regions:            []telemetry.Region{region},
		Enemies:            []*Enemy{parent},
		rand:               rand.New(rand.NewSource(1)),
		randFloat:          func() float64 { return 0 },
		MaxDecoysPerParent: 1,
	}
	drone := &telemetry.Drone{Position: telemetry.Position{Lat: 0, Lon: 0}}
	eng.Step([]*telemetry.Drone{drone})
	eng.Step([]*telemetry.Drone{drone})
	count := 0
	for _, e := range eng.Enemies {
		if e.Type == EnemyDecoy && e.ParentID == parent.ID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 decoy, got %d", count)
	}
}

func TestEngine_DecoyExpiration(t *testing.T) {
	region := telemetry.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}
	parent := &Enemy{ID: "p", Type: EnemyVehicle, Position: telemetry.Position{Lat: 0, Lon: 0}, Region: region}
	eng := &Engine{
		regions:       []telemetry.Region{region},
		Enemies:       []*Enemy{parent},
		rand:          rand.New(rand.NewSource(1)),
		randFloat:     func() float64 { return 0 },
		DecoyLifespan: time.Hour,
	}
	drone := &telemetry.Drone{Position: telemetry.Position{Lat: 0, Lon: 0}}
	eng.Step([]*telemetry.Drone{drone})
	if len(eng.Enemies) != 2 {
		t.Fatalf("expected decoy to spawn")
	}
	eng.Enemies[1].ExpiresAt = time.Now().Add(-time.Second)
	eng.Step(nil)
	for _, e := range eng.Enemies {
		if e.Type == EnemyDecoy {
			t.Fatalf("expected decoy to expire")
		}
	}
}
