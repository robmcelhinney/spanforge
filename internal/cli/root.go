package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/robmcelhinney/spanforge/internal/app"
	"github.com/robmcelhinney/spanforge/internal/config"
	"github.com/robmcelhinney/spanforge/internal/generator"
	"github.com/robmcelhinney/spanforge/internal/validate"
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
	cmd.AddCommand(newValidateCmd())

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

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate generated traces in a backend",
	}
	cmd.AddCommand(newValidateBackendCmd("tempo"))
	cmd.AddCommand(newValidateBackendCmd("jaeger"))
	return cmd
}

func newValidateBackendCmd(backend string) *cobra.Command {
	var endpoint string
	var reportFile string
	var wait time.Duration
	var pollInterval time.Duration
	var output string

	cmd := &cobra.Command{
		Use:   backend,
		Short: "Validate sampled traces in " + backend,
		RunE: func(cmd *cobra.Command, args []string) error {
			output = strings.ToLower(strings.TrimSpace(output))
			if output != "text" && output != "json" {
				return fmt.Errorf("output must be text or json")
			}
			result, err := validate.Run(context.Background(), validate.Options{
				Backend:      backend,
				Endpoint:     endpoint,
				ReportFile:   reportFile,
				Wait:         wait,
				PollInterval: pollInterval,
			})
			if err != nil {
				return err
			}
			if output == "json" {
				if err := validate.WriteJSON(cmd.OutOrStdout(), result); err != nil {
					return err
				}
			} else {
				if err := validate.WriteText(cmd.OutOrStdout(), result); err != nil {
					return err
				}
			}
			if result.Status == validate.StatusFail {
				return fmt.Errorf("%s validation failed", backend)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Backend query API endpoint")
	cmd.Flags().StringVar(&reportFile, "report-file", "", "spanforge JSON report file")
	cmd.Flags().DurationVar(&wait, "wait", 30*time.Second, "Maximum time to wait for sampled traces")
	cmd.Flags().DurationVar(&pollInterval, "poll-interval", 2*time.Second, "Polling interval while waiting")
	cmd.Flags().StringVar(&output, "output", "text", "Validation output format: text or json")
	_ = cmd.MarkFlagRequired("report-file")
	return cmd
}
