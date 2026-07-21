package formatting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	ghjq "github.com/cli/go-gh/v2/pkg/jq"
	ghtemplate "github.com/cli/go-gh/v2/pkg/template"
	ghterm "github.com/cli/go-gh/v2/pkg/term"
	"github.com/sudosubin/gh-attach/internal/upload"
)

var availableAssetFields = []string{
	"id",
	"name",
	"size",
	"contentType",
	"href",
	"originalName",
}

type Options struct {
	JSONFlagSet bool
	JSONFields  []string
	JQ          string
	Template    string
}

func WriteAsset(w io.Writer, asset upload.Asset, opts Options) error {
	if !opts.JSONFlagSet {
		_, err := fmt.Fprintln(w, asset.Href)
		return err
	}

	payload := map[string]any{
		"id":           asset.ID,
		"name":         asset.Name,
		"size":         asset.Size,
		"contentType":  asset.ContentType,
		"href":         asset.Href,
		"originalName": asset.OriginalName,
	}

	selected, err := selectFields(payload, opts.JSONFields)
	if err != nil {
		return err
	}
	payload = selected

	body, err := marshalOutputJSON(payload)
	if err != nil {
		return err
	}

	if opts.JQ != "" {
		term := ghterm.FromEnv()
		indent := ""
		colorize := false
		if term.IsTerminalOutput() {
			indent = "  "
			colorize = term.IsColorEnabled()
		}
		return ghjq.EvaluateFormatted(bytes.NewReader(body), w, opts.JQ, indent, colorize)
	}

	if opts.Template != "" {
		term := ghterm.FromEnv()
		width, _, _ := term.Size()
		if width <= 0 {
			width = 80
		}
		t := ghtemplate.New(w, width, term.IsColorEnabled())
		if err := t.Parse(opts.Template); err != nil {
			return err
		}
		if err := t.Execute(bytes.NewReader(body)); err != nil {
			return err
		}
		return t.Flush()
	}

	_, err = fmt.Fprintln(w, string(body))
	return err
}

func AvailableJSONFields() []string {
	out := make([]string, len(availableAssetFields))
	copy(out, availableAssetFields)
	return out
}

func selectFields(payload map[string]any, fields []string) (map[string]any, error) {
	if len(fields) == 0 {
		return payload, nil
	}

	selected := make(map[string]any, len(fields))
	for _, part := range fields {
		field := strings.TrimSpace(part)
		if field == "" {
			continue
		}
		v, ok := payload[field]
		if !ok {
			return nil, fmt.Errorf("Unknown JSON field: %q\nAvailable fields:\n  %s", field, strings.Join(availableAssetFields, "\n  "))
		}
		selected[field] = v
	}
	return selected, nil
}

func marshalOutputJSON(v any) ([]byte, error) {
	term := ghterm.FromEnv()
	if term.IsTerminalOutput() {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}
