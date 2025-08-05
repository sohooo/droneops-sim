package sim

import (
	"context"
	"testing"
	"time"

	gpb "github.com/GreptimeTeam/greptime-proto/go/greptime/v1"
	"github.com/GreptimeTeam/greptimedb-ingester-go/table"

	"droneops-sim/internal/telemetry"
)

type mockGreptimeClient struct {
	table *table.Table
}

func (m *mockGreptimeClient) Write(ctx context.Context, tables ...*table.Table) (*gpb.GreptimeResponse, error) {
	if len(tables) > 0 {
		m.table = tables[0]
	}
	return &gpb.GreptimeResponse{}, nil
}

func TestGreptimeWriterSwarmEventsJSON(t *testing.T) {
	ts := time.Unix(0, 0).UTC()
	rows := []telemetry.SwarmEventRow{
		{
			ClusterID: "c1",
			EventType: telemetry.SwarmEventAssignment,
			DroneIDs:  []string{"d1", "d2"},
			EnemyID:   "e1",
			Timestamp: ts,
		},
	}

	m := &mockGreptimeClient{}
	w := &GreptimeDBWriter{client: m, swarmTable: "swarm_events"}

	if err := w.WriteSwarmEvents(rows); err != nil {
		t.Fatalf("WriteSwarmEvents: %v", err)
	}
	if m.table == nil {
		t.Fatalf("expected table to be captured")
	}

	schema := m.table.GetRows().Schema
	if len(schema) < 3 {
		t.Fatalf("unexpected schema length: %d", len(schema))
	}
	if schema[2].Datatype != gpb.ColumnDataType_JSON {
		t.Fatalf("drone_ids column type = %v, want %v", schema[2].Datatype, gpb.ColumnDataType_JSON)
	}

	got := m.table.GetRows().Rows[0].Values[2].GetStringValue()
	want := "[\"d1\",\"d2\"]"
	if got != want {
		t.Fatalf("drone_ids = %s, want %s", got, want)
	}
}

func TestGreptimeWriterMissions(t *testing.T) {
	rows := []telemetry.MissionRow{{
		ID:          "m1",
		Name:        "Mission",
		Objective:   "Obj",
		Description: "Desc",
		Region:      telemetry.Region{Name: "R", CenterLat: 1, CenterLon: 2, RadiusKM: 3},
	}}

	m := &mockGreptimeClient{}
	w := &GreptimeDBWriter{client: m, missionTable: "missions"}

	if err := w.WriteMissions(rows); err != nil {
		t.Fatalf("WriteMissions: %v", err)
	}
	if m.table == nil {
		t.Fatalf("expected table to be captured")
	}
	if got := m.table.GetRows().Rows[0].Values[0].GetStringValue(); got != "m1" {
		t.Fatalf("id = %s, want m1", got)
	}
	if got := m.table.GetRows().Rows[0].Values[4].GetStringValue(); got != "R" {
		t.Fatalf("region_name = %s, want R", got)
	}
}
