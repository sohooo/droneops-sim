package main

import (
	"flag"
	"log"
	"os"

	"droneops-sim/internal/sim"
)

func main() {
	input := flag.String("input", "", "Path to telemetry log file")
	speed := flag.Float64("speed", 1.0, "Playback speed multiplier")
	printOnly := flag.Bool("print-only", false, "Print telemetry to STDOUT instead of writing to DB")
	flag.Parse()

	if *input == "" {
		log.Fatal("input file required")
	}

	var writer sim.TelemetryWriter
	if *printOnly || os.Getenv("GREPTIMEDB_ENDPOINT") == "" {
		writer = &sim.StdoutWriter{}
	} else {
		endpoint := os.Getenv("GREPTIMEDB_ENDPOINT")
		table := os.Getenv("GREPTIMEDB_TABLE")
		detTable := os.Getenv("ENEMY_DETECTION_TABLE")
		w, err := sim.NewGreptimeDBWriter(endpoint, "public", table, detTable)
		if err != nil {
			log.Fatalf("Failed to init GreptimeDB writer: %v", err)
		}
		writer = w
	}

	if err := sim.ReplayLogFile(*input, writer, *speed); err != nil {
		log.Fatalf("Replay failed: %v", err)
	}
}
