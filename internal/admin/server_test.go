package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"droneops-sim/internal/config"
	"droneops-sim/internal/sim"
	"droneops-sim/internal/telemetry"
)

func TestHandleToggleChaos(t *testing.T) {
	// Setup simulator and server
	cfg := &config.SimulationConfig{
		Zones:  []config.Region{{Name: "region-1", CenterLat: 48.2, CenterLon: 16.4, RadiusKM: 50}},
		Fleets: []config.Fleet{{Name: "fleet-1", Model: "small-fpv", Count: 3}},
	}
	sim := sim.NewSimulator("test-cluster", cfg, nil, nil, 1)
	server := NewServer(sim)

	// Create a request to toggle chaos
	req := httptest.NewRequest(http.MethodPost, "/toggle-chaos", nil)
	w := httptest.NewRecorder()

	// Call the handler
	server.handleToggleChaos(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.StatusCode)
	}

	// Verify chaos mode is toggled
	if !sim.Chaos() {
		t.Errorf("Expected chaos mode to be enabled, but it was not")
	}

	// Call the handler again to toggle chaos off
	w = httptest.NewRecorder()
	server.handleToggleChaos(w, req)

	// Verify chaos mode is toggled off
	if sim.Chaos() {
		t.Errorf("Expected chaos mode to be disabled, but it was enabled")
	}
}

func TestHandleLaunchDrones(t *testing.T) {
	// Setup simulator and server
	cfg := &config.SimulationConfig{
		Zones:  []config.Region{{Name: "region-1", CenterLat: 48.2, CenterLon: 16.4, RadiusKM: 50}},
		Fleets: []config.Fleet{{Name: "fleet-1", Model: "small-fpv", Count: 3}},
	}
	sim := sim.NewSimulator("test-cluster", cfg, nil, nil, 1)
	server := NewServer(sim)

	// Create a request to launch drones
	req := httptest.NewRequest(http.MethodPost, "/launch-drones?model=medium-uav&count=5", nil)
	w := httptest.NewRecorder()

	// Call the handler
	server.handleLaunch(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status NoContent, got %v", resp.StatusCode)
	}

	// Verify drones are launched
	fleetFound := false
	for _, fleet := range sim.Health() {
		if fleet.Name == "medium-uav" {
			fleetFound = true
			if fleet.Total != 5 {
				t.Errorf("Expected 5 drones, got %d", fleet.Total)
			}
		}
	}
	if !fleetFound {
		t.Errorf("Expected fleet 'medium-uav' to be found, but it was not")
	}
}

func TestHandleHealth(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:  []config.Region{{Name: "region-1", CenterLat: 48.2, CenterLon: 16.4, RadiusKM: 50}},
		Fleets: []config.Fleet{{Name: "fleet-1", Model: "small-fpv", Count: 1}},
	}
	simulator := sim.NewSimulator("cluster", cfg, nil, nil, 1)
	server := NewServer(simulator)

	req := httptest.NewRequest(http.MethodGet, "/fleet-health", nil)
	w := httptest.NewRecorder()
	server.handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status OK, got %v", resp.StatusCode)
	}
	var data []sim.FleetHealth
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(data) != 1 || data[0].Total != 1 {
		t.Errorf("unexpected health data: %+v", data)
	}
}

func TestHandleTelemetry(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:  []config.Region{{Name: "r1", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets: []config.Fleet{{Name: "f1", Model: "small-fpv", Count: 1}},
	}

	simulator := sim.NewSimulator("cluster", cfg, nil, nil, 1)
	server := NewServer(simulator)

	req := httptest.NewRequest(http.MethodGet, "/telemetry", nil)
	w := httptest.NewRecorder()
	server.handleTelemetry(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status OK, got %v", resp.StatusCode)
	}
	var rows []telemetry.TelemetryRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 telemetry row, got %d", len(rows))
	}
}

func TestHandleMapData(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:  []config.Region{{Name: "r1", CenterLat: 0, CenterLon: 0, RadiusKM: 1}},
		Fleets: []config.Fleet{{Name: "f1", Model: "small-fpv", Count: 1}},
	}

	simulator := sim.NewSimulator("cluster", cfg, nil, nil, 1)
	server := NewServer(simulator)

	req := httptest.NewRequest(http.MethodGet, "/map-data", nil)
	w := httptest.NewRecorder()
	server.handleMapData(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status OK, got %v", resp.StatusCode)
	}
	var data sim.MapData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(data.Drones) != 1 {
		t.Errorf("expected 1 drone, got %d", len(data.Drones))
	}
	if len(data.Enemies) == 0 {
		t.Errorf("expected enemies to be included")
	}
}
