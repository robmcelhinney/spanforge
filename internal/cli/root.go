package cli

import (
	"fmt"
	"strings"

	"github.com/robmcelhinney/spanforge/internal/app"
	"github.com/robmcelhinney/spanforge/internal/config"
	"github.com/robmcelhinney/spanforge/internal/generator"
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
	cmd.AddCommand(newProfilesCmd())

	return cmd
}

func newProfilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "List and describe generation profiles",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List generation profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, profile := range generator.Profiles() {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", profile.Name, profile.Description)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show <name>",
		Short: "Describe a generation profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, ok := generator.Profile(args[0])
			if !ok {
				return fmt.Errorf("unknown profile %q", args[0])
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", profile.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "  Description: %s\n", profile.Description)
			fmt.Fprintf(cmd.OutOrStdout(), "  Services: %s\n", strings.Join(profile.Services, ", "))
			fmt.Fprintf(cmd.OutOrStdout(), "  Routes: %s\n", strings.Join(profile.Routes, ", "))
			fmt.Fprintf(cmd.OutOrStdout(), "  Failure modes: %s\n", strings.Join(profile.FailureModes, ", "))
			return nil
		},
	})
	return cmd
}
