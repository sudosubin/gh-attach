package cookies

import (
	"net/http"
	"testing"
)

func TestValuesForHost_ExcludesOtherSubdomains(t *testing.T) {
	t.Parallel()

	in := []*http.Cookie{
		{Name: "dotcom_user", Value: "sudosubin", Domain: "gist.github.com", Path: "/"},
		{Name: "dotcom_user", Value: "other-user", Domain: "github.com", Path: "/"},
	}

	got := ValuesForHost(in, "dotcom_user", "github.com")
	if len(got) != 1 || got[0] != "other-user" {
		t.Fatalf("ValuesForHost() = %#v, want [other-user]", got)
	}
}
