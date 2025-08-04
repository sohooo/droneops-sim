package sim

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"strings"
	"testing"
	"time"

	"droneops-sim/internal/config"
	"droneops-sim/internal/enemy"
	"droneops-sim/internal/logging"
	"droneops-sim/internal/telemetry"
)

type MockStateWriter struct {
	Rows []telemetry.SimulationStateRow
}

func (w *MockStateWriter) WriteState(r telemetry.SimulationStateRow) error {
	w.Rows = append(w.Rows, r)
	return nil
}

func (w *MockStateWriter) Write(telemetry.TelemetryRow) error { return nil }

func TestSimulationStateEmission(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:             []config.Region{{Name: "r1", CenterLat: 1, CenterLon: 2, RadiusKM: 1}},
		Missions:          []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "r1", CenterLat: 1, CenterLon: 2, RadiusKM: 1}}},
		Fleets:            []config.Fleet{{Name: "f1", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "r1", MissionID: "m1"}},
		CommunicationLoss: 0.25,
		SensorNoise:       0.1,
		WeatherImpact:     0.2,
	}
	writer := &MockStateWriter{}
	sim := NewSimulator("c1", cfg, writer, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	sim.chaosMode = true
	sim.enemyFollowerTargets["e1"] = 1
	sim.enemyFollowers["e1"] = []string{}
	sim.enemyObjects["e1"] = &enemy.Enemy{ID: "e1"}

	sim.tick(context.Background())

	if len(writer.Rows) != 1 {
		t.Fatalf("expected 1 state row, got %d", len(writer.Rows))
	}
	r1 := writer.Rows[0]
	if !r1.ChaosMode || r1.MessagesSent == 0 {
		t.Fatalf("unexpected state row: %+v", r1)
	}
	if r1.CommunicationLoss != 0.25 || r1.SensorNoise != 0.1 || r1.WeatherImpact != 0.2 {
		t.Fatalf("unexpected metrics: %+v", r1)
	}

	// second tick with chaos mode off and no messages
	writer.Rows = nil
	sim.chaosMode = false
	delete(sim.enemyFollowerTargets, "e1")
	delete(sim.enemyFollowers, "e1")
	sim.tick(context.Background())
	if len(writer.Rows) != 1 {
		t.Fatalf("expected 1 state row, got %d", len(writer.Rows))
	}
	r2 := writer.Rows[0]
	if r2.ChaosMode {
		t.Fatalf("unexpected second state row: %+v", r2)
	}
}

type errStateWriter struct{}

func (w *errStateWriter) Write(telemetry.TelemetryRow) error { return nil }
func (w *errStateWriter) WriteState(telemetry.SimulationStateRow) error {
	return errors.New("write failure")
}

type errBatchStateWriter struct{}

func (w *errBatchStateWriter) Write(telemetry.TelemetryRow) error            { return nil }
func (w *errBatchStateWriter) WriteState(telemetry.SimulationStateRow) error { return nil }
func (w *errBatchStateWriter) WriteStates([]telemetry.SimulationStateRow) error {
	return errors.New("batch failure")
}

func TestSimulationStateWriteErrorsLogged(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "r1", CenterLat: 1, CenterLon: 2, RadiusKM: 1}},
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "r1", CenterLat: 1, CenterLon: 2, RadiusKM: 1}}},
		Fleets:   []config.Fleet{{Name: "f1", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "r1", MissionID: "m1"}},
	}
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	ctx := logging.NewContext(context.Background(), logger)

	writer := &errStateWriter{}
	sim := NewSimulator("c1", cfg, writer, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	sim.enableMovement = false

	sim.tick(ctx)

	if !strings.Contains(buf.String(), "state write failed") {
		t.Fatalf("expected state write failure log, got %s", buf.String())
	}
}

func TestSimulationStateBatchWriteErrorsLogged(t *testing.T) {
	cfg := &config.SimulationConfig{
		Zones:    []config.Region{{Name: "r1", CenterLat: 1, CenterLon: 2, RadiusKM: 1}},
		Missions: []config.Mission{{ID: "m1", Name: "m1", Objective: "", Description: "", Region: config.Region{Name: "r1", CenterLat: 1, CenterLon: 2, RadiusKM: 1}}},
		Fleets:   []config.Fleet{{Name: "f1", Model: "small-fpv", Count: 1, MovementPattern: "patrol", HomeRegion: "r1", MissionID: "m1"}},
	}
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	ctx := logging.NewContext(context.Background(), logger)

	writer := &errBatchStateWriter{}
	sim := NewSimulator("c1", cfg, writer, nil, time.Second, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0).UTC() })
	sim.enableMovement = false

	sim.tick(ctx)

	if !strings.Contains(buf.String(), "state batch write failed") {
		t.Fatalf("expected state batch write failure log, got %s", buf.String())
	}
}
