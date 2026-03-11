package cmd

import (
	"fmt"
	"gocker/internal/runtime"
	"os"

	"github.com/spf13/cobra"
)

// childCmd represents the child command
var childCmd = &cobra.Command{
	Use:   "child",
	Short: "A command that is re-execed by the parent process to run the target command",
	Long: `The child command is an internal command that is not meant to be called directly by users. It is used by the parent process to re-exec itself with a special argument to indicate that it should run the target command instead of the normal CLI logic.

This allows us to have a clean separation between the parent process, which is responsible for setup and coordination, and the child process, which is responsible for executing the target command within the configured environment.
Example:
  gocker child -- echo hello
  (this is not meant to be called directly, it's used internally by the parent process during re-exec)`,
	Hidden: true,
	Args: func(cmd *cobra.Command, args []string) error {
		if !isReExec() {
			return fmt.Errorf("the child command should only be called by the parent process during re-exec")
		}
		if len(args) == 0 {
			return fmt.Errorf("a command to run is required (this should be passed by the parent process during re-exec)")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		req := runtime.RunCmdRequest{
			Command: args[0],
			Args:    args[1:],
		}
		return runtime.RunChild(cmd.Context(), req)
	},
}

// isReExec returns true only when the process was launched by RunParent,
// which injects GOCKER_REEXEC_TOKEN into the child's environment.
// The token itself is verified by RunChild via the pipe.
func isReExec() bool {
	return os.Getenv(runtime.GockerReExecEnv) != ""
}

func init() {
	rootCmd.AddCommand(childCmd)
}
