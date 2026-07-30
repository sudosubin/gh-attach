package app

import (
	"strings"
	"testing"
)

func TestNewTokenSession(t *testing.T) {
	const token = "0123456789abcdef"

	session, err := newTokenSession("github.example.com:443", "  "+token+"  ")
	if err != nil {
		t.Fatalf("newTokenSession() error = %v", err)
	}
	if session.UserAgent == "" {
		t.Fatal("UserAgent is empty")
	}
	if len(session.Cookies) != 2 {
		t.Fatalf("len(Cookies) = %d, want 2", len(session.Cookies))
	}
	for i, name := range []string{"user_session", "__Host-user_session_same_site"} {
		cookie := session.Cookies[i]
		if cookie.Name != name || cookie.Value != token {
			t.Fatalf("cookie[%d] = %s=%q, want %s=%q", i, cookie.Name, cookie.Value, name, token)
		}
		if cookie.Domain != "github.example.com" || cookie.Path != "/" || !cookie.Secure || !cookie.HttpOnly {
			t.Fatalf("cookie[%d] scope = %#v", i, cookie)
		}
	}
}

func TestNewTokenSessionRejectsInvalidValuesWithoutLeakingThem(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty"},
		{name: "cookie header", value: "Cookie: user_session=secret"},
		{name: "user session cookie", value: "user_session=secret"},
		{name: "same site cookie", value: "__Host-user_session_same_site=secret"},
		{name: "cookie pair", value: "secret; other=value"},
		{name: "invalid cookie byte", value: "invalid\nsecret"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newTokenSession("github.com", test.value)
			if err == nil {
				t.Fatal("newTokenSession() error = nil, want an error")
			}
			if test.value != "" && strings.Contains(err.Error(), test.value) {
				t.Fatalf("error leaked the rejected value: %v", err)
			}
		})
	}
}
