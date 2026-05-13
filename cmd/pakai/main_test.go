package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	cmd := newVersionCmd()
	if cmd.Use != "version" {
		t.Errorf("version command Use = %q, want %q", cmd.Use, "version")
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("version command returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("version command produced no output")
	}
}

func TestVersionCommandRegistered(t *testing.T) {
	// Simulate what main() does: add version to root, then verify
	r := *rootCmd
	r.AddCommand(newVersionCmd())

	found := false
	for _, sub := range r.Commands() {
		if sub.Name() == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("root command should have a 'version' subcommand when added")
	}
}

func TestRootCommandStderr(t *testing.T) {
	if rootCmd.ErrOrStderr() != os.Stderr {
		t.Error("root command should write errors to stderr")
	}
}

func TestSetupTmuxCommandPrintsStatusSnippet(t *testing.T) {
	cmd := newSetupTmuxCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("setup tmux command returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "status-right") {
		t.Fatalf("expected tmux setup output to contain status-right, got %q", out)
	}
	if !strings.Contains(out, "status-interval 30") {
		t.Fatalf("expected tmux setup output to contain status-interval, got %q", out)
	}
}
