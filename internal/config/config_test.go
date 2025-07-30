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
  - name: test-mission
    zone: region-x
    description: test
fleets:
  - name: fleet-x
    model: small-fpv
    count: 2
    movement_pattern: patrol
    home_region: region-x
`
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// This test skips full CUE validation for speed, uses ValidateWithCue = no-op or mocked.
	cfg, err := Load(tmpFile, "../../schemas/simulation.cue")
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if len(cfg.Fleets) != 1 || cfg.Fleets[0].Name != "fleet-x" {
		t.Errorf("Unexpected fleet data: %+v", cfg.Fleets)
	}
}

func TestValidateWithCue_InvalidPath(t *testing.T) {
	err := ValidateWithCue("non-existent.yaml", "../../schemas/simulation.cue")
	if err == nil {
		t.Fatalf("expected error for missing YAML file")
	}
}
