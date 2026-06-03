package browserprovider

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

func TestSplitContainer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		raw      string
		wantID   string
		wantName string
	}{
		{"", "", ""},
		{"2", "2", ""},
		{"2|sudosubin@example.com", "2", "sudosubin@example.com"},
		{"2|name|with|bars", "2", "name|with|bars"},
		{"|orphan-name", "", "orphan-name"},
	}

	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			id, name := splitContainer(tc.raw)
			if id != tc.wantID || name != tc.wantName {
				t.Fatalf("splitContainer(%q) = (%q, %q), want (%q, %q)", tc.raw, id, name, tc.wantID, tc.wantName)
			}
		})
	}
}

func TestContainerMatches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		sel   cookies.ProfileSelector
		id    string
		cname string
		want  bool
	}{
		{"match by name", cookies.ProfileSelector{Container: "sudosubin@example.com"}, "2", "sudosubin@example.com", true},
		{"name match is case-insensitive", cookies.ProfileSelector{Container: "SUDOSUBIN@example.com"}, "2", "sudosubin@example.com", true},
		{"name mismatch", cookies.ProfileSelector{Container: "Work"}, "2", "sudosubin@example.com", false},
		{"name selector rejects unnamed container", cookies.ProfileSelector{Container: "2"}, "2", "", false},
		{"match by id", cookies.ProfileSelector{Container: "2", MatchByID: true}, "2", "sudosubin@example.com", true},
		{"id match without name", cookies.ProfileSelector{Container: "2", MatchByID: true}, "2", "", true},
		{"id mismatch", cookies.ProfileSelector{Container: "9", MatchByID: true}, "2", "sudosubin@example.com", false},
		{"id selector rejects empty raw", cookies.ProfileSelector{Container: "2", MatchByID: true}, "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := containerMatches(tc.sel, tc.id, tc.cname)
			if got != tc.want {
				t.Fatalf("containerMatches(%+v, %q, %q) = %v, want %v", tc.sel, tc.id, tc.cname, got, tc.want)
			}
		})
	}
}

func TestCookieGroupKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		sel     cookies.ProfileSelector
		profile string
		id      string
		cname   string
		wantOK  bool
		wantKey kookyGroupKey
	}{
		{
			"no filter accepts",
			cookies.ProfileSelector{}, "default", "2", "Work", true,
			kookyGroupKey{Profile: "default", ContainerID: "2", ContainerName: "Work"},
		},
		{
			"profile match",
			cookies.ProfileSelector{Profile: "default"}, "default", "", "", true,
			kookyGroupKey{Profile: "default"},
		},
		{
			"profile mismatch",
			cookies.ProfileSelector{Profile: "default"}, "other", "", "", false,
			kookyGroupKey{},
		},
		{
			"container name match",
			cookies.ProfileSelector{Container: "Work"}, "default", "2", "Work", true,
			kookyGroupKey{Profile: "default", ContainerID: "2", ContainerName: "Work"},
		},
		{
			"container name mismatch",
			cookies.ProfileSelector{Container: "Work"}, "default", "2", "Home", false,
			kookyGroupKey{},
		},
		{
			"container id match",
			cookies.ProfileSelector{Container: "2", MatchByID: true}, "default", "2", "Work", true,
			kookyGroupKey{Profile: "default", ContainerID: "2", ContainerName: "Work"},
		},
		{
			"profile + container both required",
			cookies.ProfileSelector{Profile: "default", Container: "Work"}, "default", "2", "Home", false,
			kookyGroupKey{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, ok := cookieGroupKey(tc.sel, tc.profile, tc.id, tc.cname)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && key != tc.wantKey {
				t.Fatalf("key = %+v, want %+v", key, tc.wantKey)
			}
		})
	}
}

func TestFinalizeKookyGroups_SplitsByContainerAndSortsDeterministically(t *testing.T) {
	t.Parallel()

	groups := map[kookyGroupKey][]*http.Cookie{
		{Profile: "default", ContainerID: "2", ContainerName: "sudosubin@example.com"}: {
			{Name: "dotcom_user", Value: "octocat"},
			{Name: "user_session", Value: "octocat-session"},
		},
		{Profile: "default", ContainerID: "1", ContainerName: "sudosubin@gmail.com"}: {
			{Name: "dotcom_user", Value: "sudosubin"},
			{Name: "user_session", Value: "sudo-session"},
		},
		{Profile: "default"}: {
			{Name: "_octo", Value: "anon"},
		},
	}

	sets := finalizeKookyGroups(groups)

	wantProfiles := []string{"default", "default:sudosubin@gmail.com", "default:sudosubin@example.com"}
	gotProfiles := make([]string, 0, len(sets))
	for _, s := range sets {
		gotProfiles = append(gotProfiles, s.Profile)
	}
	if !slices.Equal(gotProfiles, wantProfiles) {
		t.Fatalf("set profiles = %v, want %v", gotProfiles, wantProfiles)
	}

	// no cross-container cookie leakage
	for _, s := range sets {
		seen := map[string]string{}
		for _, c := range s.Cookies {
			if prev, dup := seen[c.Name]; dup {
				t.Fatalf("profile=%q duplicate cookie %q values %q vs %q", s.Profile, c.Name, prev, c.Value)
			}
			seen[c.Name] = c.Value
		}
	}
}

func TestCompareContainerID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		a, b string
		want int
	}{
		{"1", "2", -1},
		{"2", "10", -1}, // numeric, not lex
		{"10", "2", 1},  // numeric, not lex
		{"2", "2", 0},
		{"abc", "abd", -1}, // both non-numeric → lex
		{"2", "abc", strings.Compare("2", "abc")}, // mixed → lex fallback
		{"", "", 0},
	}

	for _, tc := range cases {
		got := compareContainerID(tc.a, tc.b)
		// Normalize sign to -1/0/1 for stable comparison.
		sign := func(n int) int {
			switch {
			case n < 0:
				return -1
			case n > 0:
				return 1
			default:
				return 0
			}
		}
		if sign(got) != sign(tc.want) {
			t.Fatalf("compareContainerID(%q, %q) = %d, want sign %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestFinalizeKookyGroups_SortsContainerIDNumerically(t *testing.T) {
	t.Parallel()

	// "10" and "2": numeric sort puts 2 before 10 (lex would reverse).
	groups := map[kookyGroupKey][]*http.Cookie{
		{Profile: "default", ContainerID: "10", ContainerName: "ten"}: {{Name: "k", Value: "v"}},
		{Profile: "default", ContainerID: "2", ContainerName: "two"}:  {{Name: "k", Value: "v"}},
	}

	sets := finalizeKookyGroups(groups)

	want := []string{"default:two", "default:ten"}
	got := []string{sets[0].Profile, sets[1].Profile}
	if !slices.Equal(got, want) {
		t.Fatalf("profiles = %v, want %v (numeric ContainerID sort)", got, want)
	}
}

func TestFinalizeKookyGroups_EmptyReturnsEmpty(t *testing.T) {
	t.Parallel()
	sets := finalizeKookyGroups(map[kookyGroupKey][]*http.Cookie{})
	if len(sets) != 0 {
		t.Fatalf("expected empty, got %v", sets)
	}
}
