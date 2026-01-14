package main

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCommandAliases(t *testing.T) {
	// Initialize commands (ensure init() is called if needed, but in main test it might be tricky.
	// However, init() runs on package load. Since we are testing main package, globals are available.)

	// We need to capture stdout/stderr to verify help output or command execution
	// But simply checking the Aliases field is enough to verify configuration.

	tests := []struct {
		name     string
		cmd      string // command use string to identify
		expected []string
	}{
		{"scan", "scan", []string{"s", "sc"}},
		{"organize", "organize", []string{"o", "org"}},
		{"undo", "undo", []string{"u"}},
		{"history", "history", []string{"h", "hist"}},
		{"junk", "junk", []string{"j"}},
		{"junk scan", "scan", []string{"s", "sc"}},          // subcommand
		{"junk clean", "clean", []string{"c", "cl", "cls"}}, // subcommand
		{"version", "version", []string{"v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmdToCheck *cobra.Command

			// Locate the command
			switch tt.name {
			case "scan":
				cmdToCheck = scanCmd
			case "organize":
				cmdToCheck = organizeCmd
			case "undo":
				cmdToCheck = undoCmd
			case "history":
				cmdToCheck = historyCmd
			case "junk":
				cmdToCheck = junkCmd
			case "junk scan":
				cmdToCheck = junkScanCmd
			case "junk clean":
				cmdToCheck = junkCleanCmd
			case "version":
				cmdToCheck = versionCmd
			}

			assert.NotNil(t, cmdToCheck, "Command %s not found", tt.name)

			// Verify aliases
			assert.ElementsMatch(t, tt.expected, cmdToCheck.Aliases, "Aliases for %s do not match", tt.name)
		})
	}
}

func TestCommandExecutionWithAliases(t *testing.T) {
	// This test verifies that Cobra actually resolves the aliases.
	// We can't easily execute the full command logic because it depends on globals/context,
	// but we can check if rootCmd.Find() resolves the alias to the correct command.

	// Ensure subcommands are added
	// Note: init() adds them, but in test execution order matters.
	// If main_test runs in same process, init() runs once.

	tests := []struct {
		inputArgs   []string
		expectedCmd *cobra.Command
	}{
		{[]string{"s"}, scanCmd},
		{[]string{"sc"}, scanCmd},
		{[]string{"scan"}, scanCmd},

		{[]string{"o"}, organizeCmd},
		{[]string{"org"}, organizeCmd},

		{[]string{"u"}, undoCmd},

		{[]string{"h"}, historyCmd},
		{[]string{"hist"}, historyCmd},

		{[]string{"j"}, junkCmd},

		{[]string{"j", "s"}, junkScanCmd},
		{[]string{"junk", "sc"}, junkScanCmd},

		{[]string{"j", "c"}, junkCleanCmd},
		{[]string{"junk", "cl"}, junkCleanCmd},

		{[]string{"v"}, versionCmd},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("alias_%v", tt.inputArgs), func(t *testing.T) {
			cmd, _, err := rootCmd.Find(tt.inputArgs)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCmd, cmd, "Input %v resolved to wrong command", tt.inputArgs)
		})
	}
}
