package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	tmpFile := "test-simulation.yaml"
	defer os.Remove(tmpFile)
	yaml := `
zones:
  - name: region-x
    center_lat: 48.2
    center_lon: 16.4
    radius_km: 50
missions:
  - id: test
    name: test-mission
    objective: test
    description: test
    region:
      name: region-x
      center_lat: 48.2
      center_lon: 16.4
      radius_km: 50
fleets:
  - name: fleet-x
    model: small-fpv
    count: 2
    movement_pattern: patrol
    home_region: region-x
    mission_id: test
`
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	cfg, err := Load(tmpFile, "../../schemas/simulation.cue")
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if len(cfg.Fleets) != 1 || cfg.Fleets[0].Name != "fleet-x" {
		t.Errorf("Unexpected fleet data: %+v", cfg.Fleets)
	}
	if cfg.Fleets[0].MissionID != "test" {
		t.Errorf("expected mission_id 'test', got %s", cfg.Fleets[0].MissionID)
	}
	if len(cfg.Missions) != 1 || cfg.Missions[0].ID != "test" {
		t.Errorf("Unexpected mission data: %+v", cfg.Missions)
	}
}

func TestValidateWithCue_InvalidPath(t *testing.T) {
	err := ValidateWithCue("non-existent.yaml", "../../schemas/simulation.cue")
	if err == nil {
		t.Fatalf("expected error for missing YAML file")
	}
}

func TestValidateWithCue_InvalidConfig(t *testing.T) {
	tmpFile := "invalid-config.yaml"
	defer os.Remove(tmpFile)
	yaml := `
zones:
  - name: region-x
    center_lat: 48.2
    center_lon: 16.4
    radius_km: -5
missions: []
fleets: []
`
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	err := ValidateWithCue(tmpFile, "../../schemas/simulation.cue")
	if err == nil {
		t.Fatalf("expected validation error for invalid config")
	}
}
