package sim

import (
	"testing"

	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

type stubSpawnWriter struct {
	fn func(enemy.Enemy)
	rm func(string)
	up func(string, enemy.EnemyStatus)
}

func (s *stubSpawnWriter) Write(telemetry.TelemetryRow) error { return nil }
func (s *stubSpawnWriter) SetSpawner(f func(enemy.Enemy))     { s.fn = f }
func (s *stubSpawnWriter) SetRemover(f func(string))          { s.rm = f }
func (s *stubSpawnWriter) SetStatusUpdater(f func(string, enemy.EnemyStatus)) {
	s.up = f
}

func TestMultiWriterSetSpawner(t *testing.T) {
	s := &stubSpawnWriter{}
	mw := NewMultiWriter([]TelemetryWriter{s}, nil, nil)
	mw.SetSpawner(func(enemy.Enemy) {})
	if s.fn == nil {
		t.Fatalf("spawner not forwarded")
	}
}

func TestMultiWriterSetRemover(t *testing.T) {
	s := &stubSpawnWriter{}
	mw := NewMultiWriter([]TelemetryWriter{s}, nil, nil)
	mw.SetRemover(func(string) {})
	if s.rm == nil {
		t.Fatalf("remover not forwarded")
	}
}

func TestMultiWriterSetStatusUpdater(t *testing.T) {
	s := &stubSpawnWriter{}
	mw := NewMultiWriter([]TelemetryWriter{s}, nil, nil)
	mw.SetStatusUpdater(func(string, enemy.EnemyStatus) {})
	if s.up == nil {
		t.Fatalf("status updater not forwarded")
	}
}
