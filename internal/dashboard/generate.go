package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

var templateFiles = []string{
	"grafana-dashboard.json.tmpl",
	"grafana_dashboard_sling.json.tmpl",
}

func rootDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

// Render parses dashboard templates and writes rendered dashboards to outDir.
func Render(outDir string) error {
	funcMap := template.FuncMap{
		"env": func(key string) (string, error) {
			v := os.Getenv(key)
			if v == "" {
				return "", fmt.Errorf("environment variable %s not set", key)
			}
			return v, nil
		},
	}

	base := rootDir()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, tplName := range templateFiles {
		path := filepath.Join(base, tplName)
		t, err := template.New(filepath.Base(tplName)).Funcs(funcMap).ParseFiles(path)
		if err != nil {
			return err
		}
		outPath := filepath.Join(outDir, strings.TrimSuffix(filepath.Base(tplName), ".tmpl"))
		f, err := os.Create(outPath)
		if err != nil {
			return err
		}
		if err := t.Execute(f, nil); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
