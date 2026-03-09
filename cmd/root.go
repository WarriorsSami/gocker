package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gocker",
	Short: "A minimal Docker-like runtime built for learning",
	Long:  "Gocker is a small container runtime in Go for learning how Docker and Linux containers work under the hood.",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
