package main

import (
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/stack"
)

// ---------------------------------------------------------------------------
// 1. TestStackCommand_HasSubcommands
// ---------------------------------------------------------------------------

func TestStackCommand_HasSubcommands(t *testing.T) {
	cmd := newStackCmd()

	want := map[string]bool{
		"init":   false,
		"up":     false,
		"down":   false,
		"status": false,
		"setup":  false,
	}

	for _, sub := range cmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("stack subcommand %q not registered", name)
		}
	}
}

// ---------------------------------------------------------------------------
// 2. TestStackSetupCommand_Flags
// ---------------------------------------------------------------------------

func TestStackSetupCommand_Flags(t *testing.T) {
	cmd := newStackSetupCmd()

	dirFlag := cmd.Flags().Lookup("dir")
	if dirFlag == nil {
		t.Fatal("--dir flag not registered")
	}
	if dirFlag.DefValue != "." {
		t.Errorf("--dir default = %q, want %q", dirFlag.DefValue, ".")
	}

	configDirFlag := cmd.Flags().Lookup("config-dir")
	if configDirFlag == nil {
		t.Fatal("--config-dir flag not registered")
	}
	if configDirFlag.DefValue != "" {
		t.Errorf("--config-dir default = %q, want empty", configDirFlag.DefValue)
	}
}

// ---------------------------------------------------------------------------
// 3. TestCheckSetupFailures
// ---------------------------------------------------------------------------

func TestCheckSetupFailures(t *testing.T) {
	t.Run("all OK returns nil", func(t *testing.T) {
		results := []stack.SetupResult{
			{Service: "radarr", Action: "create root folder", OK: true},
			{Service: "prowlarr", Action: "add application", OK: true},
			{Service: "qbittorrent", Action: "set preferences", OK: true},
		}
		if err := checkSetupFailures(results); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("one failure returns error", func(t *testing.T) {
		results := []stack.SetupResult{
			{Service: "radarr", Action: "create root folder", OK: true},
			{Service: "prowlarr", Action: "add application", OK: false, Error: "connection refused"},
			{Service: "qbittorrent", Action: "set preferences", OK: true},
		}
		if err := checkSetupFailures(results); err == nil {
			t.Error("expected error when one step failed")
		}
	})

	t.Run("all failures returns error", func(t *testing.T) {
		results := []stack.SetupResult{
			{Service: "radarr", Action: "health check", OK: false, Error: "timeout"},
			{Service: "prowlarr", Action: "health check", OK: false, Error: "timeout"},
		}
		if err := checkSetupFailures(results); err == nil {
			t.Error("expected error when all steps failed")
		}
	})

	t.Run("empty results returns nil", func(t *testing.T) {
		if err := checkSetupFailures(nil); err != nil {
			t.Errorf("expected nil for empty results, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// 4. TestStackUpCommand_FileFlag
// ---------------------------------------------------------------------------

func TestStackUpCommand_FileFlag(t *testing.T) {
	cmd := newStackUpCmd()

	flag := cmd.Flags().Lookup("file")
	if flag == nil {
		t.Fatal("--file flag not registered")
	}
	if flag.DefValue != "docker-compose.yml" {
		t.Errorf("--file default = %q, want docker-compose.yml", flag.DefValue)
	}
	if flag.Shorthand != "f" {
		t.Errorf("--file shorthand = %q, want %q", flag.Shorthand, "f")
	}
}

// ---------------------------------------------------------------------------
// 5. TestStackDownCommand_FileFlag
// ---------------------------------------------------------------------------

func TestStackDownCommand_FileFlag(t *testing.T) {
	cmd := newStackDownCmd()

	flag := cmd.Flags().Lookup("file")
	if flag == nil {
		t.Fatal("--file flag not registered")
	}
	if flag.DefValue != "docker-compose.yml" {
		t.Errorf("--file default = %q, want docker-compose.yml", flag.DefValue)
	}
}

// ---------------------------------------------------------------------------
// 6. TestStackStatusCommand_FileFlag
// ---------------------------------------------------------------------------

func TestStackStatusCommand_FileFlag(t *testing.T) {
	cmd := newStackStatusCmd()

	flag := cmd.Flags().Lookup("file")
	if flag == nil {
		t.Fatal("--file flag not registered")
	}
	if flag.DefValue != "docker-compose.yml" {
		t.Errorf("--file default = %q, want docker-compose.yml", flag.DefValue)
	}
}
