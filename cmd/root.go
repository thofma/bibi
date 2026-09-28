package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bibi",
	Short: "Retrieve BibTeX for mathematical literature",
	Long: `bibi is a command line tool to retrieve bibliographic information
for literature in mathematics using MR Lookup, zbMATH Open, and the
Mathematics Genealogy Project.`,
	SilenceUsage: true,
}

func Execute() {
	if exitCode := commandExitCode(rootCmd); exitCode != 0 {
		os.Exit(exitCode)
	}
}

func commandExitCode(command *cobra.Command) int {
	if err := command.Execute(); err != nil {
		return 1
	}
	return 0
}
