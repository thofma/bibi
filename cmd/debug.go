package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/internal/httpclient"
)

func addDebugFlag(command *cobra.Command) {
	command.PersistentFlags().Bool("debug", false, "Write diagnostic details to stderr")
}

func withDebug(run func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(command *cobra.Command, args []string) error {
		enabled, _ := command.Flags().GetBool("debug")
		writer := command.ErrOrStderr()
		if !enabled {
			writer = nil
		}
		restore := diagnostics.SetOutput(writer)
		defer restore()
		previousContext := command.Context()
		command.SetContext(httpclient.WithObserver(httpclient.WithSession(previousContext), func(event httpclient.Event) {
			if !enabled {
				_, _ = fmt.Fprintln(command.ErrOrStderr(), event.String())
			}
		}))
		defer command.SetContext(previousContext)

		diagnostics.Printf("command=%s args=%q", command.CommandPath(), args)
		start := time.Now()
		err := run(command, args)
		if err != nil {
			diagnostics.Printf("command failed after %s: %v", time.Since(start).Round(time.Millisecond), err)
		} else {
			diagnostics.Printf("command completed in %s", time.Since(start).Round(time.Millisecond))
		}
		return err
	}
}
