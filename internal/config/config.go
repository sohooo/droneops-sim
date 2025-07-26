// YAML config loader with CUE validation integration
package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Behavior defines dynamic properties of a drone model/fleet
type Behavior struct {
	BatteryDrainRate float64 `yaml:"battery_drain_rate"`
	FailureRate      float64 `yaml:"failure_rate"`
	SpeedMinKmh      float64 `yaml:"speed_min_kmh"`
	SpeedMaxKmh      float64 `yaml:"speed_max_kmh"`
}

// Region defines an operational region
type Region struct {
	Name      string  `yaml:"name"`
	CenterLat float64 `yaml:"center_lat"`
	CenterLon float64 `yaml:"center_lon"`
	RadiusKM  float64 `yaml:"radius_km"`
}

// Fleet defines a fleet of drones of the same model and behavior
type Fleet struct {
	Name            string   `yaml:"name"`
	Model           string   `yaml:"model"`
	Count           int      `yaml:"count"`
	MovementPattern string   `yaml:"movement_pattern"`
	HomeRegion      string   `yaml:"home_region"`
	Behavior        Behavior `yaml:"behavior"`
}

// Mission describes a named mission that operates within a zone
type Mission struct {
	Name        string `yaml:"name"`
	Zone        string `yaml:"zone"`
	Description string `yaml:"description"`
}

// SimulationConfig is the root configuration for zones, missions, and fleets
type SimulationConfig struct {
	Zones    []Region  `yaml:"zones"`
	Missions []Mission `yaml:"missions"`
	Fleets   []Fleet   `yaml:"fleets"`
}

// Load loads YAML config and validates it against a CUE schema
func Load(configPath, cueSchemaPath string) (*SimulationConfig, error) {
	// Validate with CUE first
	if err := ValidateWithCue(configPath, cueSchemaPath); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg SimulationConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	log.Printf("Loaded configuration: %+v", cfg)

	return &cfg, nil
}
