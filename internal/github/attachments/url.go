package attachments

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ParseURL validates the shared user-attachment URL format, without choosing an authentication host.
func ParseURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid attachment URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return nil, errors.New("attachment URL must be an HTTPS URL without credentials, query, or fragment")
	}

	segments := strings.Split(strings.TrimPrefix(parsed.EscapedPath(), "/"), "/")
	validAsset := len(segments) == 3 && segments[0] == "user-attachments" && segments[1] == "assets" && segments[2] != ""
	validFile := len(segments) >= 4 && segments[0] == "user-attachments" && segments[1] == "files" &&
		segments[2] != "" && segments[3] != ""
	if (!validAsset && !validFile) || segments[len(segments)-1] == "" {
		return nil, errors.New("URL is not a GitHub user-attachment")
	}

	return parsed, nil
}
