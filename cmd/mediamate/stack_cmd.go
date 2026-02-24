package main

import (
	"fmt"
	"log/slog"

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
	fmt.Println(styleDim.Render("  2. Run: docker compose up -d"))
	fmt.Println(styleDim.Render("  3. Run: mediamate chat"))

	return nil
}
