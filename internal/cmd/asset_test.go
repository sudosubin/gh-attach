package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/sudosubin/gh-attach/internal/github/attachments"
)

func TestWriteAsset_DefaultOutputIsHref(t *testing.T) {
	asset := attachments.Asset{Href: "https://github.com/user-attachments/assets/abc"}

	var out bytes.Buffer
	if err := writeAssets(&out, []attachments.Asset{asset}, options{}); err != nil {
		t.Fatalf("writeAssets() error = %v", err)
	}

	want := asset.Href + "\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestWriteAsset_JSONOutput(t *testing.T) {
	asset := attachments.Asset{
		ID:          1,
		Name:        "image.png",
		ContentType: "image/png",
		Href:        "https://github.com/user-attachments/assets/abc",
	}

	var out bytes.Buffer
	if err := writeAssets(&out, []attachments.Asset{asset}, options{JSONFlagSet: true, JSONFields: []string{"href", "name"}}); err != nil {
		t.Fatalf("writeAssets() error = %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("unmarshal output: %v, output=%q", err, out.String())
	}
	if len(got) != 1 {
		t.Fatalf("len(json output) = %d, want 1", len(got))
	}
	if got[0]["href"] != asset.Href {
		t.Fatalf("href = %v, want %v", got[0]["href"], asset.Href)
	}
	if got[0]["name"] != asset.Name {
		t.Fatalf("name = %v, want %v", got[0]["name"], asset.Name)
	}
	if len(got[0]) != 2 {
		t.Fatalf("len(json fields) = %d, want 2", len(got[0]))
	}
}

func TestWriteAssets_TemplateUsesArray(t *testing.T) {
	asset := attachments.Asset{
		Name: "image.png",
		Href: "https://example.com/image.png",
	}

	var out bytes.Buffer
	err := writeAssets(&out, []attachments.Asset{asset}, options{
		JSONFlagSet: true,
		JSONFields:  []string{"href", "name"},
		Template:    `{{range .}}{{.name}} -> {{.href}}{{"\n"}}{{end}}`,
	})
	if err != nil {
		t.Fatalf("writeAssets() error = %v", err)
	}
	if want := asset.Name + " -> " + asset.Href + "\n"; out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestWriteAssets_Multiple(t *testing.T) {
	assets := []attachments.Asset{
		{Name: "image.png", Href: "https://example.com/image.png"},
		{Name: "report.pdf", Href: "https://example.com/report.pdf"},
	}

	t.Run("default output", func(t *testing.T) {
		var out bytes.Buffer
		if err := writeAssets(&out, assets, options{}); err != nil {
			t.Fatalf("writeAssets() error = %v", err)
		}
		if want := assets[0].Href + "\n" + assets[1].Href + "\n"; out.String() != want {
			t.Fatalf("output = %q, want %q", out.String(), want)
		}
	})

	t.Run("json output", func(t *testing.T) {
		var out bytes.Buffer
		if err := writeAssets(&out, assets, options{JSONFlagSet: true, JSONFields: []string{"href"}}); err != nil {
			t.Fatalf("writeAssets() error = %v", err)
		}

		var got []map[string]any
		if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
			t.Fatalf("unmarshal output: %v, output=%q", err, out.String())
		}
		if len(got) != 2 || got[0]["href"] != assets[0].Href || got[1]["href"] != assets[1].Href {
			t.Fatalf("json output = %#v", got)
		}
	})
}
