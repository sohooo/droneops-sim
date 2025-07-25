package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"droneops-sim/internal/config"
	"droneops-sim/internal/sim"
)

func main() {
	// CLI flags
	printOnly := flag.Bool("print-only", false, "Print telemetry to STDOUT instead of writing to DB")
	configPath := flag.String("config", "config/fleet.yaml", "Path to fleet configuration YAML")
	cueSchemaPath := flag.String("schema", "schemas/fleet.cue", "Path to CUE schema file")
	flag.Parse()

	// Load fleet configuration
	cfg, err := config.Load(*configPath, *cueSchemaPath)
	if err != nil {
		log.Fatalf("Config load failed: %v", err)
	}

	// Determine writer (GreptimeDB or STDOUT)
	var writer sim.TelemetryWriter
	if *printOnly || os.Getenv("GREPTIMEDB_ENDPOINT") == "" {
		log.Println("[Main] Print-only mode: telemetry will be printed to STDOUT")
		writer = &sim.StdoutWriter{}
	} else {
		endpoint := os.Getenv("GREPTIMEDB_ENDPOINT")
		table := os.Getenv("GREPTIMEDB_TABLE")
		writer, err = sim.NewGreptimeDBWriter(endpoint, "public", table)
		if err != nil {
			log.Fatalf("Failed to init GreptimeDB writer: %v", err)
		}
	}

	// Cluster identity (defaults to mission-01)
	clusterID := os.Getenv("CLUSTER_ID")
	if clusterID == "" {
		clusterID = "mission-01"
	}

	// Simulator setup
	simulator := sim.NewSimulator(clusterID, cfg, writer, 1*time.Second)

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
