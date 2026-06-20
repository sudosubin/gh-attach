package browserprovider

import (
	"maps"
	"net/http"
	"slices"
	"testing"

	libsweetcookie "github.com/steipete/sweetcookie"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

func TestContainerMatches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		sel  cookies.ProfileSelector
		ct   libsweetcookie.Container
		want bool
	}{
		{"match by name", cookies.ProfileSelector{Container: "Work"}, libsweetcookie.Container{ID: 2, Name: "Work"}, true},
		{"name match is case-sensitive", cookies.ProfileSelector{Container: "work"}, libsweetcookie.Container{ID: 2, Name: "Work"}, false},
		{"name mismatch", cookies.ProfileSelector{Container: "Work"}, libsweetcookie.Container{ID: 2, Name: "Home"}, false},
		{"name selector rejects unnamed container", cookies.ProfileSelector{Container: "2"}, libsweetcookie.Container{ID: 2}, false},
		{"match by id", cookies.ProfileSelector{Container: "2", MatchByID: true}, libsweetcookie.Container{ID: 2, Name: "Work"}, true},
		{"id match without name", cookies.ProfileSelector{Container: "2", MatchByID: true}, libsweetcookie.Container{ID: 2}, true},
		{"id mismatch", cookies.ProfileSelector{Container: "9", MatchByID: true}, libsweetcookie.Container{ID: 2}, false},
		{"default container is never selectable", cookies.ProfileSelector{Container: "0", MatchByID: true}, libsweetcookie.Container{}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := containerMatches(tc.sel, tc.ct)
			if got != tc.want {
				t.Fatalf("containerMatches(%+v, %+v) = %v, want %v", tc.sel, tc.ct, got, tc.want)
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
		ct      libsweetcookie.Container
		wantOK  bool
		wantKey containerGroupKey
	}{
		{
			"no container filter accepts",
			cookies.ProfileSelector{}, "default", libsweetcookie.Container{ID: 2, Name: "Work"}, true,
			containerGroupKey{Profile: "default", ContainerID: 2, ContainerName: "Work"},
		},
		{
			"profile in selector does not filter (handled by sweetcookie)",
			cookies.ProfileSelector{Profile: "anything"}, "default", libsweetcookie.Container{}, true,
			containerGroupKey{Profile: "default"},
		},
		{
			"container name match",
			cookies.ProfileSelector{Container: "Work"}, "default", libsweetcookie.Container{ID: 2, Name: "Work"}, true,
			containerGroupKey{Profile: "default", ContainerID: 2, ContainerName: "Work"},
		},
		{
			"container name mismatch",
			cookies.ProfileSelector{Container: "Work"}, "default", libsweetcookie.Container{ID: 2, Name: "Home"}, false,
			containerGroupKey{},
		},
		{
			"container id match",
			cookies.ProfileSelector{Container: "2", MatchByID: true}, "default", libsweetcookie.Container{ID: 2, Name: "Work"}, true,
			containerGroupKey{Profile: "default", ContainerID: 2, ContainerName: "Work"},
		},
		{
			"container selector rejects default container",
			cookies.ProfileSelector{Container: "Work"}, "default", libsweetcookie.Container{}, false,
			containerGroupKey{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, ok := cookieGroupKey(tc.sel, tc.profile, tc.ct)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && key != tc.wantKey {
				t.Fatalf("key = %+v, want %+v", key, tc.wantKey)
			}
		})
	}
}

func TestFinalizeContainerGroups_SplitsByContainerAndSortsDeterministically(t *testing.T) {
	t.Parallel()

	groups := map[containerGroupKey][]*http.Cookie{
		{Profile: "default", ContainerID: 2, ContainerName: "sudosubin@example.com"}: {
			{Name: "dotcom_user", Value: "octocat"},
			{Name: "user_session", Value: "octocat-session"},
		},
		{Profile: "default", ContainerID: 1, ContainerName: "sudosubin@gmail.com"}: {
			{Name: "dotcom_user", Value: "sudosubin"},
			{Name: "user_session", Value: "sudo-session"},
		},
		{Profile: "default"}: {
			{Name: "_octo", Value: "anon"},
		},
	}

	sets := finalizeContainerGroups(groups)

	wantProfiles := []string{"default", "default:sudosubin@gmail.com", "default:sudosubin@example.com"}
	gotProfiles := make([]string, 0, len(sets))
	for _, s := range sets {
		gotProfiles = append(gotProfiles, s.Profile)
	}
	if !slices.Equal(gotProfiles, wantProfiles) {
		t.Fatalf("set profiles = %v, want %v", gotProfiles, wantProfiles)
	}

	wantCookies := map[string]map[string]string{
		"default":                       {"_octo": "anon"},
		"default:sudosubin@gmail.com":   {"dotcom_user": "sudosubin", "user_session": "sudo-session"},
		"default:sudosubin@example.com": {"dotcom_user": "octocat", "user_session": "octocat-session"},
	}
	for _, s := range sets {
		got := map[string]string{}
		for _, c := range s.Cookies {
			got[c.Name] = c.Value
		}
		if !maps.Equal(got, wantCookies[s.Profile]) {
			t.Fatalf("profile=%q cookies = %v, want %v", s.Profile, got, wantCookies[s.Profile])
		}
	}
}

func TestFinalizeContainerGroups_SortsContainerIDNumerically(t *testing.T) {
	t.Parallel()

	// ID 2 sorts before 10 numerically (string sort would reverse).
	groups := map[containerGroupKey][]*http.Cookie{
		{Profile: "default", ContainerID: 10, ContainerName: "ten"}: {{Name: "k", Value: "v"}},
		{Profile: "default", ContainerID: 2, ContainerName: "two"}:  {{Name: "k", Value: "v"}},
	}

	sets := finalizeContainerGroups(groups)

	want := []string{"default:two", "default:ten"}
	got := []string{sets[0].Profile, sets[1].Profile}
	if !slices.Equal(got, want) {
		t.Fatalf("profiles = %v, want %v (numeric ContainerID sort)", got, want)
	}
}

func TestFinalizeContainerGroups_EmptyReturnsEmpty(t *testing.T) {
	t.Parallel()
	sets := finalizeContainerGroups(map[containerGroupKey][]*http.Cookie{})
	if len(sets) != 0 {
		t.Fatalf("expected empty, got %v", sets)
	}
}
