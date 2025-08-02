package scenario

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Scenario defines a mission scenario with ordered phases and an overall description.
type Scenario struct {
	Name        string  `yaml:"name,omitempty"`
	Description string  `yaml:"description,omitempty"`
	Phases      []Phase `yaml:"phases"`
}

// Phase describes a stage in the mission with objectives and triggers for transitions.
type Phase struct {
	Name            string           `yaml:"name"`
	Description     string           `yaml:"description,omitempty"`
	EnemyObjectives []EnemyObjective `yaml:"enemy_objectives,omitempty"`
	Triggers        []Trigger        `yaml:"triggers,omitempty"`
}

// EnemyObjective declares a dynamic behaviour for an enemy entity during a phase.
type EnemyObjective struct {
	ID     string `yaml:"id"`
	Action string `yaml:"action"`
	Target string `yaml:"target,omitempty"`
}

// Trigger moves the scenario to another phase based on an event.
type Trigger struct {
	Event string `yaml:"event"`
	Value int    `yaml:"value"`
	Next  string `yaml:"next"`
}

// Event represents a runtime occurrence that may advance the scenario.
type Event struct {
	Type  string
	Value int
}

// Load reads a YAML scenario definition from disk.
func Load(path string) (*Scenario, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scenario: %w", err)
	}
	var s Scenario
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse scenario: %w", err)
	}
	return &s, nil
}

// NextPhase returns the name of the next phase given the current phase and event.
// If no trigger matches, ok will be false.
func (s *Scenario) NextPhase(current string, ev Event) (next string, ok bool) {
	for _, p := range s.Phases {
		if p.Name != current {
			continue
		}
		for _, tr := range p.Triggers {
			if tr.Event == ev.Type && ev.Value >= tr.Value {
				return tr.Next, true
			}
		}
	}
	return "", false
}
