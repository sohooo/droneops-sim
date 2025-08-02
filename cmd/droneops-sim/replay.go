package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"droneops-sim/internal/sim"
)

var (
	replayInput     string
	replaySpeed     float64
	replayPrintOnly bool
)

var replayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Replay a telemetry log file",
	Long:  "replay feeds telemetry rows from a log file back into GreptimeDB or STDOUT.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if replayInput == "" {
			return fmt.Errorf("input file required")
		}
		writer, err := newTelemetryWriter(replayPrintOnly)
		if err != nil {
			return err
		}
		return sim.ReplayLogFile(replayInput, writer, replaySpeed)
	},
}

func init() {
	replayCmd.Flags().StringVar(&replayInput, "input", "", "Path to telemetry log file")
	replayCmd.Flags().Float64Var(&replaySpeed, "speed", 1.0, "Playback speed multiplier")
	replayCmd.Flags().BoolVar(&replayPrintOnly, "print-only", false, "Print telemetry to STDOUT instead of writing to DB")
	replayCmd.MarkFlagRequired("input")
}
