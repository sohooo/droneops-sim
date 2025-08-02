package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"droneops-sim/internal/admin"
	"droneops-sim/internal/config"
	"droneops-sim/internal/sim"
)

func main() {
	// CLI flags
	printOnly := flag.Bool("print-only", false, "Print telemetry to STDOUT instead of writing to DB")
	configPath := flag.String("config", "config/simulation.yaml", "Path to simulation configuration YAML")
	cueSchemaPath := flag.String("schema", "schemas/simulation.cue", "Path to CUE schema file")
	tickFlag := flag.Duration("tick", time.Second, "Telemetry tick interval (e.g. 500ms, 2s)")
	logFile := flag.String("log-file", "", "Path to export telemetry/detection logs (JSONL)")
	flag.Parse()

	// Load simulation configuration
	cfg, err := config.Load(*configPath, *cueSchemaPath)
	if err != nil {
		log.Fatalf("Config load failed: %v", err)
	}

	// Determine writer (GreptimeDB or STDOUT)
	var writer sim.TelemetryWriter
	var detectWriter sim.DetectionWriter
	if *printOnly || os.Getenv("GREPTIMEDB_ENDPOINT") == "" {
		log.Println("[Main] Print-only mode: telemetry will be printed to STDOUT")
		sw := &sim.StdoutWriter{}
		writer = sw
		detectWriter = sw
	} else {
		endpoint := os.Getenv("GREPTIMEDB_ENDPOINT")
		table := os.Getenv("GREPTIMEDB_TABLE")
		detTable := os.Getenv("ENEMY_DETECTION_TABLE")
		writer, err = sim.NewGreptimeDBWriter(endpoint, "public", table, detTable)
		if err != nil {
			log.Fatalf("Failed to init GreptimeDB writer: %v", err)
		}
		detectWriter = writer.(sim.DetectionWriter)
	}

	// Optional log export
	if *logFile != "" {
		fw, err := sim.NewFileWriter(*logFile, *logFile+".detections")
		if err != nil {
			log.Fatalf("Failed to create log file: %v", err)
		}
		defer fw.Close()
		mw := sim.NewMultiWriter([]sim.TelemetryWriter{writer, fw}, []sim.DetectionWriter{detectWriter, fw})
		writer = mw
		detectWriter = mw
	}

	// Cluster identity (defaults to mission-01)
	clusterID := os.Getenv("CLUSTER_ID")
	if clusterID == "" {
		clusterID = "mission-01"
	}

	// Simulator setup
	tickInterval := *tickFlag
	if envTick := os.Getenv("TICK_INTERVAL"); envTick != "" {
		d, err := time.ParseDuration(envTick)
		if err != nil {
			log.Fatalf("Invalid TICK_INTERVAL: %v", err)
		}
		tickInterval = d
	}

	simulator := sim.NewSimulator(clusterID, cfg, writer, detectWriter, tickInterval)

	// Start admin UI
	go func() {
		srv := admin.NewServer(simulator)
		log.Println("[Main] Admin UI listening on :8080")
		if err := srv.Start(":8080"); err != nil {
			log.Fatalf("Admin server failed: %v", err)
		}
	}()

	// Graceful shutdown handling
	stop := make(chan struct{})
	go func() {
		simulator.Run(stop)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	close(stop)
	log.Println("[Main] Drone simulation stopped.")
}
