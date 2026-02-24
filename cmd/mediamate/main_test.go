package main

import (
	"testing"
)

func TestRootCommand_HasSubcommands(t *testing.T) {
	root := newRootCmd()

	want := map[string]bool{
		"version": false,
		"chat":    false,
		"query":   false,
		"status":  false,
		"config":  false,
	}

	for _, cmd := range root.Commands() {
		if _, ok := want[cmd.Name()]; ok {
			want[cmd.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered", name)
		}
	}
}

func TestRootCommand_ConfigFlag(t *testing.T) {
	root := newRootCmd()
	flag := root.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("--config flag not registered")
	}
	if flag.DefValue != "configs/mediamate.yaml" {
		t.Errorf("--config default = %q, want %q", flag.DefValue, "configs/mediamate.yaml")
	}
	if flag.Shorthand != "c" {
		t.Errorf("--config shorthand = %q, want %q", flag.Shorthand, "c")
	}
}

func TestVersionCommand(t *testing.T) {
	cmd := newVersionCmd()
	if cmd.Use != "version" {
		t.Errorf("Use = %q, want %q", cmd.Use, "version")
	}
}

func TestQueryCommand_RequiresArgs(t *testing.T) {
	cmd := newQueryCmd()
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("query command should require at least 1 argument")
	}
	err = cmd.Args(cmd, []string{"hello"})
	if err != nil {
		t.Errorf("query command should accept args: %v", err)
	}
}

func TestConfigCommand_HasValidateSubcommand(t *testing.T) {
	cmd := newConfigCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Name() == "validate" {
			found = true
			break
		}
	}
	if !found {
		t.Error("config command missing 'validate' subcommand")
	}
}
