package main

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
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
	fmt.Println(styleDim.Render("  1. Edit .env and fill in your API keys"))
	fmt.Println(styleDim.Render("  2. Run: mediamate stack up"))
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
	fmt.Printf("  %-20s %-12s %-12s %s\n",
		styleDim.Render("SERVICE"),
		styleDim.Render("STATE"),
		styleDim.Render("HEALTH"),
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

		fmt.Printf("  %-20s %-12s %-12s %s\n",
			c.Service,
			stateStyle.Render(c.State),
			healthStr,
			c.Status,
		)

		serviceNames = append(serviceNames, c.Service)
	}

	return serviceNames
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

		indicator := styleSuccess.Render("OK")
		if !r.Healthy {
			indicator = styleError.Render("FAIL")
		}

		fmt.Printf("  %-20s %-6s  %-6s  %s\n",
			r.Name,
			indicator,
			statusStr,
			latencyStr,
		)
	}
}
