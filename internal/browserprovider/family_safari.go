package browserprovider

import (
	"cmp"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

type safariFamily struct{}

// version reads Safari's version from its app bundle; storePath is unused.
func (safariFamily) version(string) string {
	return safariVersion("/Applications/Safari.app/Contents/Info.plist")
}

// userAgent builds a Safari UA; Safari applies no reduction, so the full version shows.
func (safariFamily) userAgent(_, version string) string {
	return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Safari/605.1.15",
		cmp.Or(version, "26.0")) // fallback; bump to current stable periodically
}

// safariVersion reads CFBundleShortVersionString from Safari's XML Info.plist (absent off macOS); keys/values are siblings.
func safariVersion(plistPath string) string {
	f, err := os.Open(plistPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	var elem string
	var atTarget bool
	for {
		tok, err := dec.Token()
		if err != nil {
			return ""
		}
		switch t := tok.(type) {
		case xml.StartElement:
			elem = t.Name.Local
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text == "" {
				continue
			}
			switch elem {
			case "key":
				atTarget = text == "CFBundleShortVersionString"
			case "string":
				if atTarget {
					return text
				}
			}
		}
	}
}
