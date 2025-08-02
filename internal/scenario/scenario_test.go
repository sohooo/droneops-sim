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
	if len(sc.Phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(sc.Phases))
	}
	if sc.Phases[0].EnemyObjectives[0].Action != "patrol" {
		t.Fatalf("unexpected objective action %s", sc.Phases[0].EnemyObjectives[0].Action)
	}
}
