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
	if err := writeAsset(&out, asset, options{}); err != nil {
		t.Fatalf("writeAsset() error = %v", err)
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
	if err := writeAsset(&out, asset, options{JSONFlagSet: true, JSONFields: []string{"href", "name"}}); err != nil {
		t.Fatalf("writeAsset() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("unmarshal output: %v, output=%q", err, out.String())
	}
	if got["href"] != asset.Href {
		t.Fatalf("href = %v, want %v", got["href"], asset.Href)
	}
	if got["name"] != asset.Name {
		t.Fatalf("name = %v, want %v", got["name"], asset.Name)
	}
	if len(got) != 2 {
		t.Fatalf("len(json fields) = %d, want 2", len(got))
	}
}
