package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"droneops-sim/internal/admin"
	"droneops-sim/internal/config"
	"droneops-sim/internal/sim"
)

var (
	simPrintOnly  bool
	simConfigPath string
	simSchemaPath string
	simTick       time.Duration
	simLogFile    string
)

var simulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Run the real-time drone simulator",
	Long:  "simulate starts a mission simulator emitting telemetry and optional enemy detection logs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(simConfigPath, simSchemaPath)
		if err != nil {
			return err
		}

		writer, detectWriter, cleanup, err := newWriters(simPrintOnly, simLogFile)
		if err != nil {
			return err
		}
		defer cleanup()

		clusterID := os.Getenv("CLUSTER_ID")
		if clusterID == "" {
			clusterID = "mission-01"
		}

		tickInterval := simTick
		if envTick := os.Getenv("TICK_INTERVAL"); envTick != "" {
			d, err := time.ParseDuration(envTick)
			if err != nil {
				return err
			}
			tickInterval = d
		}

		simulator := sim.NewSimulator(clusterID, cfg, writer, detectWriter, tickInterval)

		go func() {
			srv := admin.NewServer(simulator)
			log.Println("[Main] Admin UI listening on :8080")
			if err := srv.Start(":8080"); err != nil {
				log.Fatalf("Admin server failed: %v", err)
			}
		}()

		stop := make(chan struct{})
		go simulator.Run(stop)

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs

		close(stop)
		log.Println("[Main] Drone simulation stopped.")
		return nil
	},
}

func init() {
	simulateCmd.Flags().BoolVar(&simPrintOnly, "print-only", false, "Print telemetry to STDOUT instead of writing to DB")
	simulateCmd.Flags().StringVar(&simConfigPath, "config", "config/simulation.yaml", "Path to simulation configuration YAML")
	simulateCmd.Flags().StringVar(&simSchemaPath, "schema", "schemas/simulation.cue", "Path to CUE schema file")
	simulateCmd.Flags().DurationVar(&simTick, "tick", time.Second, "Telemetry tick interval (e.g. 500ms, 2s)")
	simulateCmd.Flags().StringVar(&simLogFile, "log-file", "", "Path to export telemetry/detection logs (JSONL)")
}
