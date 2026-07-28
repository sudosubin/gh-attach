package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	ghjq "github.com/cli/go-gh/v2/pkg/jq"
	ghtemplate "github.com/cli/go-gh/v2/pkg/template"
	ghterm "github.com/cli/go-gh/v2/pkg/term"
	"github.com/sudosubin/gh-attach/internal/github/attachments"
)

var availableAssetFields = []string{
	"id",
	"name",
	"size",
	"contentType",
	"href",
	"originalName",
}

type options struct {
	Markdown    bool
	JSONFlagSet bool
	JSONFields  []string
	JQ          string
	Template    string
}

func writeAssets(w io.Writer, assets []attachments.Asset, opts options) error {
	if !opts.JSONFlagSet {
		for _, asset := range assets {
			output := asset.Href
			if opts.Markdown {
				output = markdownForAsset(asset)
			}
			if _, err := fmt.Fprintln(w, output); err != nil {
				return err
			}
		}
		return nil
	}

	payloads := make([]map[string]any, 0, len(assets))
	for _, asset := range assets {
		payload, err := selectFields(map[string]any{
			"id":           asset.ID,
			"name":         asset.Name,
			"size":         asset.Size,
			"contentType":  asset.ContentType,
			"href":         asset.Href,
			"originalName": asset.OriginalName,
		}, opts.JSONFields)
		if err != nil {
			return err
		}
		payloads = append(payloads, payload)
	}

	body, err := marshalOutputJSON(payloads)
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

var markdownLabelEscaper = strings.NewReplacer(
	`\`, `\\`,
	`[`, `\[`,
	`]`, `\]`,
)

func markdownForAsset(asset attachments.Asset) string {
	name := markdownLabelEscaper.Replace(asset.Name)
	switch asset.ContentType {
	case "image/gif", "image/jpeg", "image/jpg", "image/png", "image/svg+xml", "image/webp":
		return fmt.Sprintf("![%s](%s)", name, asset.Href)
	case "video/mp4", "video/quicktime":
		return asset.Href
	default:
		return fmt.Sprintf("[%s](%s)", name, asset.Href)
	}
}

func availableJSONFields() []string {
	return slices.Clone(availableAssetFields)
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
			return nil, fmt.Errorf("unknown JSON field: %q\nAvailable fields:\n  %s", field, strings.Join(availableAssetFields, "\n  "))
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
