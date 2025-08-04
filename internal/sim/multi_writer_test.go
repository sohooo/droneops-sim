package sim

import (
	"testing"

	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

type stubSpawnWriter struct{ fn func(enemy.Enemy) }

func (s *stubSpawnWriter) Write(telemetry.TelemetryRow) error { return nil }
func (s *stubSpawnWriter) SetSpawner(f func(enemy.Enemy))     { s.fn = f }

func TestMultiWriterSetSpawner(t *testing.T) {
	s := &stubSpawnWriter{}
	mw := NewMultiWriter([]TelemetryWriter{s}, nil, nil)
	mw.SetSpawner(func(enemy.Enemy) {})
	if s.fn == nil {
		t.Fatalf("spawner not forwarded")
	}
}
