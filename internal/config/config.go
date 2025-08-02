// YAML config loader with CUE validation integration
package config

import (
	"fmt"
	"log"
	"os"

	"cuelang.org/go/cue/cuecontext"
	"gopkg.in/yaml.v3"
)

// Behavior defines dynamic properties of a drone model/fleet
type Behavior struct {
	BatteryDrainRate   float64 `yaml:"battery_drain_rate"`
	FailureRate        float64 `yaml:"failure_rate"`
	SpeedMinKmh        float64 `yaml:"speed_min_kmh"`
	SpeedMaxKmh        float64 `yaml:"speed_max_kmh"`
	SensorErrorRate    float64 `yaml:"sensor_error_rate"`
	DropoutRate        float64 `yaml:"dropout_rate"`
	BatteryAnomalyRate float64 `yaml:"battery_anomaly_rate"`
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
	Zones              []Region       `yaml:"zones"`
	Missions           []Mission      `yaml:"missions"`
	Fleets             []Fleet        `yaml:"fleets"`
	EnemyCount         int            `yaml:"enemy_count"`
	DetectionRadiusM   float64        `yaml:"detection_radius_m"`
	SensorNoise        float64        `yaml:"sensor_noise"`
	TerrainOcclusion   float64        `yaml:"terrain_occlusion"`
	WeatherImpact      float64        `yaml:"weather_impact"`
	FollowConfidence   float64        `yaml:"follow_confidence"`
	SwarmResponses     map[string]int `yaml:"swarm_responses"`
	MissionCriticality string         `yaml:"mission_criticality"`
	CommunicationLoss  float64        `yaml:"communication_loss"`
	BandwidthLimit     int            `yaml:"bandwidth_limit"`
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

// ValidateWithCue validates a YAML configuration file using a CUE schema file.
func ValidateWithCue(configFile, cueFile string) error {
	ctx := cuecontext.New()

	// Read YAML config
	yamlBytes, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("cannot read YAML config: %w", err)
	}
	var configData map[string]interface{}
	if err := yaml.Unmarshal(yamlBytes, &configData); err != nil {
		return fmt.Errorf("cannot unmarshal YAML config: %w", err)
	}
	configVal := ctx.CompileBytes(yamlBytes)

	// Read CUE schema
	schemaBytes, err := os.ReadFile(cueFile)
	if err != nil {
		return fmt.Errorf("cannot read CUE schema: %w", err)
	}
	schemaVal := ctx.CompileBytes(schemaBytes)

	// Validate config against schema
	if err := schemaVal.Subsume(configVal); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}
