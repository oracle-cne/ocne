package loadimage

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCommandSelectionContract(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		message string
	}{
		{"missing-source", nil, "accepts 1 arg"},
		{"extra-source", []string{"first.tar", "second.tar"}, "accepts 1 arg"},
		{"conflict", []string{"image.tar", "--node", "worker", "--selector", "pool=development"}, "none of the others can be"},
		{"invalid-selector", []string{"absent.tar", "-s", "pool in ("}, "invalid node selector"},
		{"selector-value", []string{"image.tar", "--selector"}, "flag needs an argument"},
		{"implementation-entrypoint", []string{"absent.tar", "-s", "pool=development"}, "no such file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("expected %q, got %v", tc.message, err)
			}
		})
	}
}

func TestCommandHelpContainsNoForcedLineBreaks(t *testing.T) {
	cmd := NewCmd()
	if strings.Contains(cmd.Long, "\n") {
		t.Fatal("help imposes line widths")
	}
	if strings.Contains(cmd.Long, "loaders start") {
		t.Fatal("help contains implementation narration")
	}
}

func TestCommandInheritsRootLogFlag(t *testing.T) {
	root := &cobra.Command{Use: "ocne"}
	root.PersistentFlags().StringP("log-level", "l", "info", "Log level")
	root.AddCommand(NewCmd())
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"load-image", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}
