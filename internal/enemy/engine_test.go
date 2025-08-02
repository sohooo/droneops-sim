package enemy

import (
	"testing"

	"math/rand"

	"droneops-sim/internal/telemetry"
)

func TestEngine_EvasiveManeuver(t *testing.T) {
	eng := &Engine{regions: []telemetry.Region{{CenterLat: 0, CenterLon: 0, RadiusKM: 1}}, Enemies: []*Enemy{{ID: "e", Type: EnemyVehicle, Position: telemetry.Position{Lat: 0, Lon: 0}, Confidence: 100, Region: telemetry.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}}}}
	drone := &telemetry.Drone{Position: telemetry.Position{Lat: 0, Lon: 0}}
	rand.Seed(1) // avoid decoy spawn
	eng.Step([]*telemetry.Drone{drone})
	if distance(eng.Enemies[0].Position, drone.Position) == 0 {
		t.Fatalf("expected enemy to move away from drone")
	}
}

func TestEngine_SpawnMultipleRegions(t *testing.T) {
	regions := []telemetry.Region{{CenterLat: 0, CenterLon: 0, RadiusKM: 1}, {CenterLat: 1, CenterLon: 1, RadiusKM: 1}}
	eng := NewEngine(1, regions)
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
	eng := &Engine{regions: []telemetry.Region{{CenterLat: 0, CenterLon: 0, RadiusKM: 1}}, Enemies: []*Enemy{{ID: "e", Type: EnemyVehicle, Position: telemetry.Position{Lat: 0, Lon: 0}, Confidence: 100, Region: telemetry.Region{CenterLat: 0, CenterLon: 0, RadiusKM: 1}}}}
	drone := &telemetry.Drone{Position: telemetry.Position{Lat: 0, Lon: 0}}
	for i := 0; i < 100; i++ {
		eng.Enemies[0].Position = telemetry.Position{Lat: 0, Lon: 0}
		eng.Step([]*telemetry.Drone{drone})
		if len(eng.Enemies) > 1 {
			decoy := eng.Enemies[1]
			if decoy.Type != EnemyDecoy {
				t.Fatalf("expected decoy type, got %s", decoy.Type)
			}
			return
		}
	}
	t.Fatalf("expected decoy to spawn")
}
