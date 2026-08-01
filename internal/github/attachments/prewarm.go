package attachments

import (
	"context"
	"net"
	"time"

	"github.com/cli/go-gh/v2/pkg/auth"
)

// cloudUploadHosts are github.com's presigned upload destinations. Which one
// a given file lands on depends on its content type (decided server-side by
// the policies response), so both are resolved without knowing which will be
// used.
var cloudUploadHosts = []string{
	"objects-origin.githubusercontent.com",
	"github-production-user-asset-6210df.s3.amazonaws.com",
}

const prewarmTimeout = 5 * time.Second

type hostLookupFunc func(ctx context.Context, host string) ([]string, error)

// PrewarmUploadHostDNS resolves the cloud upload hosts' DNS in the
// background so the lookup overlaps with the referer-page fetch instead of
// sitting on the critical path right before the upload request. It issues
// no HTTP request and holds no connection open, so it leaves nothing for a
// server to observe. Enterprise Server's upload host isn't known ahead of
// the policies response, so it's skipped there.
func PrewarmUploadHostDNS(ctx context.Context, host string) {
	prewarmUploadHostDNS(ctx, host, net.DefaultResolver.LookupHost)
}

// prewarmUploadHostDNS takes the lookup function as a parameter (rather than
// a package-level var) so tests can inject a stub without mutating shared
// state that the fire-and-forget goroutines here may still be reading after
// the test that started them returns.
func prewarmUploadHostDNS(ctx context.Context, host string, lookup hostLookupFunc) {
	if auth.IsEnterprise(host) {
		return
	}

	for _, h := range cloudUploadHosts {
		go func(h string) {
			ctx, cancel := context.WithTimeout(ctx, prewarmTimeout)
			defer cancel()
			_, _ = lookup(ctx, h)
		}(h)
	}
}
