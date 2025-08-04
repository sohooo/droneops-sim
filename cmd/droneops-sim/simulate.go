package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"droneops-sim/internal/admin"
	"droneops-sim/internal/config"
	"droneops-sim/internal/logging"
	"droneops-sim/internal/sim"
)

var (
	simPrintOnly         bool
	simConfigPath        string
	simSchemaPath        string
	simTick              time.Duration
	simLogFile           string
	simEnableDetections  bool = true
	simEnableSwarmEvents bool = true
	simEnableMovement    bool = true
	simEnableState       bool = true
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

		if v := os.Getenv("ENABLE_DETECTIONS"); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				simEnableDetections = b
			}
		}
		if v := os.Getenv("ENABLE_SWARM_EVENTS"); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				simEnableSwarmEvents = b
			}
		}
		if v := os.Getenv("ENABLE_MOVEMENT_METRICS"); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				simEnableMovement = b
			}
		}
		if v := os.Getenv("ENABLE_SIMULATION_STATE"); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				simEnableState = b
			}
		}

		cfg.Telemetry.Detections = &simEnableDetections
		cfg.Telemetry.SwarmEvents = &simEnableSwarmEvents
		cfg.Telemetry.MovementMetrics = &simEnableMovement
		cfg.Telemetry.SimulationState = &simEnableState

		writer, detectWriter, cleanup, err := newWriters(cfg, simPrintOnly, simLogFile, simEnableDetections, simEnableSwarmEvents, simEnableState)
		if err != nil {
			return err
		}
		defer cleanup()
		if c, ok := writer.(io.Closer); ok {
			defer c.Close()
		}

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

		ctx, cancel := context.WithCancel(context.Background())
		log := logging.New()
		ctx = logging.NewContext(ctx, log)
		defer cancel()

		simulator := sim.NewSimulator(clusterID, cfg, writer, detectWriter, tickInterval, nil, nil)

		srv := admin.NewServer(simulator)
		if aw, ok := writer.(sim.AdminStatusWriter); ok {
			aw.SetAdminStatus(true)
		}
		go func() {
			log := logging.FromContext(ctx)
			log.Info("Admin UI listening", "addr", ":8080")
			if err := srv.Start(ctx, ":8080"); err != nil && err != http.ErrServerClosed {
				log.Error("Admin server failed", "err", err)
				os.Exit(1)
			}
		}()

		go simulator.Run(ctx)

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs

		cancel()
		log.Info("Drone simulation stopped")
		return nil
	},
}

func init() {
	simulateCmd.Flags().BoolVar(&simPrintOnly, "print-only", false, "Print telemetry to STDOUT instead of writing to DB")
	simulateCmd.Flags().StringVar(&simConfigPath, "config", "config/simulation.yaml", "Path to simulation configuration YAML")
	simulateCmd.Flags().StringVar(&simSchemaPath, "schema", "schemas/simulation.cue", "Path to CUE schema file")
	simulateCmd.Flags().DurationVar(&simTick, "tick", time.Second, "Telemetry tick interval (e.g. 500ms, 2s)")
	simulateCmd.Flags().StringVar(&simLogFile, "log-file", "", "Path to export telemetry/detection logs (JSONL)")
	simulateCmd.Flags().BoolVar(&simEnableDetections, "detections", true, "Enable enemy detection stream")
	simulateCmd.Flags().BoolVar(&simEnableSwarmEvents, "swarm-events", true, "Enable swarm event stream")
	simulateCmd.Flags().BoolVar(&simEnableMovement, "movement-metrics", true, "Enable drone movement telemetry stream")
	simulateCmd.Flags().BoolVar(&simEnableState, "simulation-state", true, "Enable simulation state stream")
}
