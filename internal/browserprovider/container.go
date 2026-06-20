package browserprovider

import (
	"cmp"
	"maps"
	"net/http"
	"slices"
	"strconv"

	libsweetcookie "github.com/steipete/sweetcookie"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type containerGroupKey struct {
	Profile       string
	ContainerID   int // 0 = default container (no isolation)
	ContainerName string
}

// cookieGroupKey buckets a cookie by profile and container. Profile selection is
// handled upstream by sweetcookie; only the container selector is applied here.
func cookieGroupKey(sel cookies.ProfileSelector, profile string, ct libsweetcookie.Container) (containerGroupKey, bool) {
	if sel.Container != "" && !containerMatches(sel, ct) {
		return containerGroupKey{}, false
	}
	return containerGroupKey{Profile: profile, ContainerID: ct.ID, ContainerName: ct.Name}, true
}

func containerMatches(sel cookies.ProfileSelector, ct libsweetcookie.Container) bool {
	if ct.ID == 0 {
		// The default container is never an explicit selection target.
		return false
	}
	if sel.MatchByID {
		return sel.Container == strconv.Itoa(ct.ID)
	}
	return ct.Name != "" && sel.Container == ct.Name
}

func finalizeContainerGroups(groups map[containerGroupKey][]*http.Cookie) []CookieSet {
	keys := slices.Collect(maps.Keys(groups))
	slices.SortFunc(keys, func(a, b containerGroupKey) int {
		return cmp.Or(
			cmp.Compare(a.Profile, b.Profile),
			cmp.Compare(a.ContainerID, b.ContainerID),
			cmp.Compare(a.ContainerName, b.ContainerName),
		)
	})

	sets := make([]CookieSet, 0, len(keys))
	for _, k := range keys {
		var profile string
		switch {
		case k.ContainerID == 0:
			profile = k.Profile
		case k.ContainerName != "":
			profile = cookies.FormatProfileSelector(k.Profile, k.ContainerName, false)
		default:
			profile = cookies.FormatProfileSelector(k.Profile, strconv.Itoa(k.ContainerID), true)
		}
		sets = append(sets, CookieSet{Profile: profile, Cookies: groups[k]})
	}
	return sets
}
