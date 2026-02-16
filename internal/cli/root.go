package cli

import (
	"fmt"

	"github.com/robmcelhinney/spanforge/internal/app"
	"github.com/robmcelhinney/spanforge/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewRootCmd(version string) *cobra.Command {
	var showVersion bool
	var flags config.FlagValues

	cmd := &cobra.Command{
		Use:   "spanforge",
		Short: "Generate fake distributed traces",
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Fprintln(cmd.OutOrStdout(), version)
				return nil
			}
			overrides := make(map[string]bool)
			cmd.Flags().Visit(func(f *pflag.Flag) {
				overrides[f.Name] = true
			})
			cfg, err := config.FromFlagsWithOverrides(flags, overrides)
			if err != nil {
				return err
			}
			return app.Run(cfg, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&showVersion, "version", false, "Print version and exit")
	config.AddFlags(cmd.Flags(), &flags)

	return cmd
}
