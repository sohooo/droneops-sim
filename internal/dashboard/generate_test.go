package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMissingEnv(t *testing.T) {
	os.Unsetenv("GREPTIMEDB_DATASOURCE_UID")
	os.Unsetenv("POSTGRES_DATASOURCE_UID")
	if err := Render(t.TempDir()); err == nil {
		t.Fatalf("expected error for missing env vars")
	}
}

func TestRenderSuccess(t *testing.T) {
	os.Setenv("GREPTIMEDB_DATASOURCE_UID", "uid1")
	os.Setenv("POSTGRES_DATASOURCE_UID", "uid2")
	defer os.Unsetenv("GREPTIMEDB_DATASOURCE_UID")
	defer os.Unsetenv("POSTGRES_DATASOURCE_UID")

	dir := t.TempDir()
	if err := Render(dir); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "grafana-dashboard.json"))
	if err != nil {
		t.Fatalf("read dashboard: %v", err)
	}
	if !strings.Contains(string(b), "uid1") {
		t.Fatalf("greptime uid not rendered")
	}

	b, err = os.ReadFile(filepath.Join(dir, "grafana_dashboard_sling.json"))
	if err != nil {
		t.Fatalf("read sling dashboard: %v", err)
	}
	if !strings.Contains(string(b), "uid2") {
		t.Fatalf("postgres uid not rendered")
	}
}
