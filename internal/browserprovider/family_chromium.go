package browserprovider

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

type chromiumFamily struct {
	browser cookies.Browser
}

// version reads Last Version, then Preferences (Opera lacks Last Version); cookie DB may sit under Network/.
func (chromiumFamily) version(storePath string) string {
	if storePath == "" {
		return ""
	}
	profileDir := filepath.Dir(storePath)
	if filepath.Base(profileDir) == "Network" {
		profileDir = filepath.Dir(profileDir)
	}

	if b, err := os.ReadFile(filepath.Join(filepath.Dir(profileDir), "Last Version")); err == nil {
		if major := majorOf(string(b)); major != "" {
			return major
		}
	}

	b, err := os.ReadFile(filepath.Join(profileDir, "Preferences"))
	if err != nil {
		return ""
	}
	var prefs struct {
		Profile struct {
			CreatedByVersion string `json:"created_by_version"`
		} `json:"profile"`
	}
	if err := json.Unmarshal(b, &prefs); err != nil {
		return ""
	}
	return majorOf(prefs.Profile.CreatedByVersion)
}

// userAgent builds a Chromium UA; reduction freezes all but the major. https://www.chromium.org/updates/ua-reduction/
func (f chromiumFamily) userAgent(goos, major string) string {
	major = cmp.Or(major, "150") // fallback; bump to current stable periodically
	platform := "X11; Linux x86_64"
	switch goos {
	case "windows":
		platform = "Windows NT 10.0; Win64; x64"
	case "darwin":
		platform = "Macintosh; Intel Mac OS X 10_15_7"
	}
	ua := fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36", platform, major)
	if f.browser == cookies.BrowserEdge {
		ua += fmt.Sprintf(" Edg/%s.0.0.0", major)
	}
	return ua
}
