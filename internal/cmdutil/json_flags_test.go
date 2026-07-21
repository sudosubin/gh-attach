package cmdutil

import (
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddJSONFlags_Valid(t *testing.T) {
	t.Parallel()

	var parsed JSONFlags
	cmd := &cobra.Command{RunE: func(*cobra.Command, []string) error { return nil }}
	AddJSONFlags(cmd, &parsed, []string{"id", "href", "name"})
	cmd.SetArgs([]string{"--json", "id,href", "--jq", ".href"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !parsed.Enabled {
		t.Fatalf("expected json enabled")
	}
	if len(parsed.Fields) != 2 || parsed.Fields[0] != "id" || parsed.Fields[1] != "href" {
		t.Fatalf("unexpected fields: %#v", parsed.Fields)
	}
	if parsed.Filter != ".href" {
		t.Fatalf("unexpected jq: %q", parsed.Filter)
	}
}

func TestAddJSONFlags_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing json arg", args: []string{"--json"}, want: "specify one or more comma-separated fields for `--json`"},
		{name: "jq without json", args: []string{"--jq", ".href"}, want: "cannot use `--jq` without specifying `--json`"},
		{name: "unknown field", args: []string{"--json", "nope"}, want: "unknown JSON field: \"nope\""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var parsed JSONFlags
			cmd := &cobra.Command{RunE: func(*cobra.Command, []string) error { return nil }}
			AddJSONFlags(cmd, &parsed, []string{"id", "href", "name"})
			cmd.SetArgs(tt.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err := cmd.ExecuteC()
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.want)
			}
		})
	}
}
