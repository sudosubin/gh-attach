package app

import (
	"errors"
	"net/http"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

func canAttemptTokenUpload(token string) bool {
	// Unrecognized formats may be legacy PATs; let the server decide.
	return token != "" && !strings.HasPrefix(token, "ghs_") && !strings.HasPrefix(token, "ghr_")
}

func isAuthenticationError(err error) bool {
	httpErr, ok := errors.AsType[*api.HTTPError](err)
	if !ok {
		return false
	}
	switch httpErr.StatusCode {
	case http.StatusUnauthorized, http.StatusNotFound:
		return true
	case http.StatusForbidden:
		message := strings.ToLower(httpErr.Message)
		return httpErr.Headers.Get("Retry-After") == "" && httpErr.Headers.Get("X-RateLimit-Remaining") != "0" &&
			!strings.Contains(message, "rate limit") && !strings.Contains(message, "abuse")
	default:
		return false
	}
}
