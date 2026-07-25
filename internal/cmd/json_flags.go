package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type jsonFlagError struct {
	error
}

type jsonFlags struct {
	Enabled  bool
	Fields   []string
	Filter   string
	Template string
}

func addJSONFlags(cmd *cobra.Command, target *jsonFlags, fields []string) {
	f := cmd.Flags()
	f.StringSlice("json", nil, "Output JSON with the specified fields")
	f.StringP("jq", "q", "", "Filter JSON output using a jq expression")
	f.StringP("template", "t", "", "Format JSON output using a Go template; see \"gh help formatting\"")

	oldPreRun := cmd.PreRunE
	cmd.PreRunE = func(c *cobra.Command, args []string) error {
		if oldPreRun != nil {
			if err := oldPreRun(c, args); err != nil {
				return err
			}
		}

		export, err := checkJSONFlags(c)
		if err != nil {
			return err
		}
		if export == nil {
			if target != nil {
				*target = jsonFlags{}
			}
			return nil
		}

		if len(fields) > 0 {
			allowed := make(map[string]struct{}, len(fields))
			for _, name := range fields {
				allowed[name] = struct{}{}
			}
			for _, name := range export.Fields {
				if _, ok := allowed[name]; !ok {
					sorted := append([]string(nil), fields...)
					sort.Strings(sorted)
					return jsonFlagError{fmt.Errorf("unknown JSON field: %q\nAvailable fields:\n  %s", name, strings.Join(sorted, "\n  "))}
				}
			}
		}

		if target != nil {
			*target = *export
		}
		return nil
	}

	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if c == cmd && err.Error() == "flag needs an argument: --json" {
			sorted := append([]string(nil), fields...)
			sort.Strings(sorted)
			return jsonFlagError{fmt.Errorf("specify one or more comma-separated fields for `--json`:\n  %s", strings.Join(sorted, "\n  "))}
		}
		if cmd.HasParent() {
			return cmd.Parent().FlagErrorFunc()(c, err)
		}
		return err
	})

	if len(fields) > 0 {
		if cmd.Annotations == nil {
			cmd.Annotations = map[string]string{}
		}
		cmd.Annotations["help:json-fields"] = strings.Join(fields, ",")
	}
}

func checkJSONFlags(cmd *cobra.Command) (*jsonFlags, error) {
	f := cmd.Flags()
	jsonFlag := f.Lookup("json")
	jqFlag := f.Lookup("jq")
	tplFlag := f.Lookup("template")

	if jsonFlag.Changed {
		jv, ok := jsonFlag.Value.(pflag.SliceValue)
		if !ok {
			return nil, fmt.Errorf("--json flag must be a string slice")
		}
		return &jsonFlags{
			Enabled:  true,
			Fields:   jv.GetSlice(),
			Filter:   jqFlag.Value.String(),
			Template: tplFlag.Value.String(),
		}, nil
	}
	if jqFlag.Changed {
		return nil, errors.New("cannot use `--jq` without specifying `--json`")
	}
	if tplFlag.Changed {
		return nil, errors.New("cannot use `--template` without specifying `--json`")
	}
	return nil, nil
}
