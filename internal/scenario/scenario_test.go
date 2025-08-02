package scenario

import "testing"

func TestScenarioTransition(t *testing.T) {
	s := Scenario{
		Phases: []Phase{{
			Name:     "patrol",
			Triggers: []Trigger{{Event: "time_elapsed", Value: 10, Next: "attack"}},
		}, {
			Name: "attack",
		}},
	}

	next, ok := s.NextPhase("patrol", Event{Type: "time_elapsed", Value: 10})
	if !ok || next != "attack" {
		t.Fatalf("expected transition to attack, got %s", next)
	}
}

func TestLoadScenario(t *testing.T) {
	sc, err := Load("testdata/simple.yaml")
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if sc.Name != "example" {
		t.Fatalf("unexpected name %s", sc.Name)
	}
	if sc.Description != "basic test scenario" {
		t.Fatalf("unexpected description %s", sc.Description)
	}
	if len(sc.Phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(sc.Phases))
	}
	if sc.Phases[0].EnemyObjectives[0].Action != "patrol" {
		t.Fatalf("unexpected objective action %s", sc.Phases[0].EnemyObjectives[0].Action)
	}
}

func TestBuiltInArcs(t *testing.T) {
	arcs := BuiltIn()
	names := []string{"escort", "search-and-rescue", "defensive-stand"}
	phases := []string{"setup", "escalation", "climax", "resolution"}
	for _, n := range names {
		arc, ok := arcs[n]
		if !ok {
			t.Fatalf("arc %s not found", n)
		}
		if arc.Description == "" {
			t.Fatalf("arc %s missing description", n)
		}
		if len(arc.Phases) != len(phases) {
			t.Fatalf("arc %s expected %d phases, got %d", n, len(phases), len(arc.Phases))
		}
		for i, ph := range phases {
			if arc.Phases[i].Name != ph {
				t.Fatalf("arc %s phase %d expected %s got %s", n, i, ph, arc.Phases[i].Name)
			}
		}
	}
}
