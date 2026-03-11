package cmd

import (
	"errors"
	"gocker/internal/runtime"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run -- <cmd> [args...]",
	Short: "Run a command inside gocker",
	Long: `Run executes an arbitrary command, forwarding stdio and propagating the exit code.

Example:
  gocker run -- echo hello
  gocker run -- /bin/sh`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a command to run is required (use: gocker run -- <cmd> [args...])")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		req := runtime.RunCmdRequest{
			Command: args[0],
			Args:    args[1:],
		}
		return runtime.RunParent(cmd.Context(), req)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
