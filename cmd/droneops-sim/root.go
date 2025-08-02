package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "droneops-sim",
	Short: "DroneOps simulation toolkit",
	Long:  "DroneOps-Sim provides simulation and replay utilities for drone telemetry.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(simulateCmd)
	rootCmd.AddCommand(replayCmd)
}
