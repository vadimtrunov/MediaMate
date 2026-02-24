package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/vadimtrunov/MediaMate/internal/stack"
)

// newStackCmd returns the "stack" parent command with its sub-commands.
func newStackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Manage the MediaMate Docker Compose stack",
		Long: "The stack commands help you set up and manage the Docker Compose stack\n" +
			"that runs MediaMate alongside Radarr, Sonarr, Jellyfin, and more.",
	}

	cmd.AddCommand(newStackInitCmd())
	cmd.AddCommand(newStackUpCmd())
	cmd.AddCommand(newStackDownCmd())
	cmd.AddCommand(newStackStatusCmd())
	cmd.AddCommand(newStackSetupCmd())

	return cmd
}

// newStackInitCmd returns the "stack init" sub-command.
func newStackInitCmd() *cobra.Command {
	var (
		outputDir      string
		overwrite      bool
		nonInteractive bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new MediaMate stack",
		Long: "Run an interactive wizard to select components, configure paths,\n" +
			"and generate docker-compose.yml, .env, and mediamate.yaml files.",
		RunE: func(_ *cobra.Command, _ []string) error {
			if nonInteractive {
				return runStackInitNonInteractive(outputDir, overwrite)
			}
			return runStackInitInteractive(outputDir, overwrite)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "output directory for generated files")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing files")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "use defaults without interactive wizard")

	return cmd
}

// runStackInitInteractive launches the Bubble Tea wizard and generates files.
func runStackInitInteractive(outputDir string, overwrite bool) error {
	p := tea.NewProgram(stack.NewWizardModel(), tea.WithAltScreen())

	result, err := p.Run()
	if err != nil {
		return fmt.Errorf("run wizard: %w", err)
	}

	model, ok := result.(stack.WizardModel)
	if !ok {
		return fmt.Errorf("unexpected model type from wizard")
	}

	if model.Aborted() {
		fmt.Println(styleDim.Render("Setup canceled."))
		return nil
	}

	if !model.Done() {
		return nil
	}

	cfg := model.Config()
	if outputDir != "." {
		cfg.OutputDir = outputDir
	}

	return generateStackFiles(&cfg, overwrite)
}

// runStackInitNonInteractive generates files with default configuration.
func runStackInitNonInteractive(outputDir string, overwrite bool) error {
	cfg := stack.DefaultConfig()
	if outputDir != "." {
		cfg.OutputDir = outputDir
	}

	return generateStackFiles(&cfg, overwrite)
}

// generateStackFiles runs the generator and prints results.
func generateStackFiles(cfg *stack.Config, overwrite bool) error {
	logger := slog.Default()
	gen := stack.NewGenerator(logger)

	result, err := gen.Generate(cfg, overwrite)
	if err != nil {
		return err
	}

	fmt.Println(styleSuccess.Render("Stack files generated successfully!"))
	fmt.Println()
	fmt.Printf("  %s %s\n", styleInfo.Render("Docker Compose:"), result.ComposePath)
	fmt.Printf("  %s %s\n", styleInfo.Render("Environment:   "), result.EnvPath)
	fmt.Printf("  %s %s\n", styleInfo.Render("Config:        "), result.ConfigPath)
	fmt.Println()
	fmt.Println(styleDim.Render("Next steps:"))
	fmt.Println(styleDim.Render("  1. Run: mediamate stack up"))
	fmt.Println(styleDim.Render("  2. Run: mediamate stack setup  (auto-configure services)"))
	fmt.Println(styleDim.Render("  3. Run: mediamate chat"))

	return nil
}

// newStackUpCmd returns the "stack up" sub-command.
func newStackUpCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start the MediaMate stack",
		Long:  "Start all services defined in the Docker Compose file in detached mode.",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			logger := slog.Default()
			compose := stack.NewCompose(file, logger)

			fmt.Println(styleInfo.Render("Starting stack..."))
			if err := compose.Up(ctx); err != nil {
				return err
			}
			fmt.Println(styleSuccess.Render("Stack started successfully!"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "docker-compose.yml", "path to docker-compose.yml")

	return cmd
}

// newStackDownCmd returns the "stack down" sub-command.
func newStackDownCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Stop the MediaMate stack",
		Long:  "Stop and remove all containers defined in the Docker Compose file.",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			logger := slog.Default()
			compose := stack.NewCompose(file, logger)

			fmt.Println(styleInfo.Render("Stopping stack..."))
			if err := compose.Down(ctx); err != nil {
				return err
			}
			fmt.Println(styleSuccess.Render("Stack stopped successfully!"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "docker-compose.yml", "path to docker-compose.yml")

	return cmd
}

// newStackStatusCmd returns the "stack status" sub-command.
func newStackStatusCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show stack service status",
		Long:  "Show the status of all containers in the stack and probe their HTTP endpoints.",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			logger := slog.Default()
			compose := stack.NewCompose(file, logger)

			containers, err := compose.PS(ctx)
			if err != nil {
				return err
			}

			if len(containers) == 0 {
				fmt.Println(styleDim.Render("No containers found. Is the stack running?"))
				return nil
			}

			serviceNames := printContainerStatus(containers)
			printHealthProbes(ctx, serviceNames, logger)

			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "docker-compose.yml", "path to docker-compose.yml")

	return cmd
}

// printContainerStatus prints a table of container states and returns
// the list of service names for subsequent health probing.
func printContainerStatus(containers []stack.ContainerStatus) []string {
	fmt.Println(styleHeader.Render("Container Status"))
	fmt.Printf("  %s %s %s %s\n",
		styleDim.Render(fmt.Sprintf("%-20s", "SERVICE")),
		styleDim.Render(fmt.Sprintf("%-12s", "STATE")),
		styleDim.Render(fmt.Sprintf("%-12s", "HEALTH")),
		styleDim.Render("STATUS"),
	)

	serviceNames := make([]string, 0, len(containers))
	for _, c := range containers {
		stateStyle := styleSuccess
		if c.State != "running" {
			stateStyle = styleError
		}

		healthStr := c.Health
		if healthStr == "" {
			healthStr = "-"
		}

		fmt.Printf("  %-20s %s %-12s %s\n",
			c.Service,
			stateStyle.Render(fmt.Sprintf("%-12s", c.State)),
			healthStr,
			c.Status,
		)

		serviceNames = append(serviceNames, c.Service)
	}

	return serviceNames
}

// newStackSetupCmd returns the "stack setup" sub-command.
func newStackSetupCmd() *cobra.Command {
	var (
		dir       string
		configDir string
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Auto-configure running stack services",
		Long: "Read API keys from service configs and automatically set up\n" +
			"Radarr, Prowlarr, and qBittorrent connections.\n\n" +
			"Run this after 'mediamate stack up' to configure all services.",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			logger := slog.Default()

			// Load config from the generated docker-compose.yml and .env so
			// that the setup respects the user's component selections from
			// "stack init" rather than using hardcoded defaults.
			cfg, err := stack.LoadConfigFromCompose(dir)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("load config from %s: %w", dir, err)
				}
				logger.Warn("compose files not found, falling back to defaults",
					slog.String("dir", dir),
				)
				cfg = stack.DefaultConfig()
			}

			if configDir != "" {
				cfg.ConfigDir = configDir
			}

			// Build a GenerateResult pointing to files in the specified directory.
			genResult := &stack.GenerateResult{
				ComposePath: filepath.Join(dir, "docker-compose.yml"),
				EnvPath:     filepath.Join(dir, ".env"),
				ConfigPath:  filepath.Join(dir, "mediamate.yaml"),
			}

			fmt.Println(styleInfo.Render("Running auto-setup..."))
			fmt.Println()

			runner := stack.NewSetupRunner(&cfg, genResult, logger)
			results := runner.Run(ctx)

			printSetupResults(results)
			return checkSetupFailures(results)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "directory containing generated stack files")
	cmd.Flags().StringVar(&configDir, "config-dir", "", "config directory (default: /srv/mediamate/config)")

	return cmd
}

// checkSetupFailures returns an error if any setup result indicates failure.
func checkSetupFailures(results []stack.SetupResult) error {
	for _, r := range results {
		if !r.OK {
			return fmt.Errorf("one or more setup steps failed")
		}
	}
	return nil
}

// printSetupResults prints a formatted table of setup outcomes.
func printSetupResults(results []stack.SetupResult) {
	fmt.Println(styleHeader.Render("Setup Results"))
	for _, r := range results {
		indicator := styleSuccess.Render("OK")
		detail := ""
		if !r.OK {
			indicator = styleError.Render("FAIL")
			detail = styleDim.Render("  " + r.Error)
		}

		fmt.Printf("  %-20s %-30s %s%s\n",
			r.Service,
			r.Action,
			indicator,
			detail,
		)
	}
	fmt.Println()
}

// printHealthProbes runs HTTP health probes against the given services and
// prints the results.
func printHealthProbes(ctx context.Context, serviceNames []string, logger *slog.Logger) {
	fmt.Println()
	fmt.Println(styleHeader.Render("HTTP Health Probes"))

	hc := stack.NewHealthChecker("", logger)
	results := hc.CheckAll(ctx, serviceNames)

	for _, r := range results {
		if r.Error == "unknown service" {
			continue
		}

		statusStr := fmt.Sprintf("%d", r.Status)
		if r.Status == 0 {
			statusStr = "-"
		}

		latencyStr := r.Latency.Round(time.Millisecond).String()
		if r.Status == 0 {
			latencyStr = "-"
		}

		indicator := styleSuccess.Render(fmt.Sprintf("%-6s", "OK"))
		if !r.Healthy {
			indicator = styleError.Render(fmt.Sprintf("%-6s", "FAIL"))
		}

		fmt.Printf("  %-20s %s  %-6s  %s\n",
			r.Name,
			indicator,
			statusStr,
			latencyStr,
		)
	}
}
