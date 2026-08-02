package attachments

import (
	"context"
	"net"
	"time"

	"github.com/cli/go-gh/v2/pkg/auth"
)

// cloudUploadHosts both get resolved: which one a file lands on is content-type
// dependent and only decided server-side by the policies response.
var cloudUploadHosts = []string{
	"objects-origin.githubusercontent.com",
	"github-production-user-asset-6210df.s3.amazonaws.com",
}

const prewarmTimeout = 5 * time.Second

type hostLookupFunc func(ctx context.Context, host string) ([]string, error)

// PrewarmUploadHostDNS resolves the upload hosts' DNS in the background so the
// lookup overlaps referer-page fetch rather than sitting on the critical path.
// DNS-only (no HTTP, no held connection); skipped on GHES, whose upload host
// isn't known until the policies response.
func PrewarmUploadHostDNS(ctx context.Context, host string) {
	prewarmUploadHostDNS(ctx, host, net.DefaultResolver.LookupHost)
}

// lookup is a parameter, not a package var, so a test's stub can't race the
// fire-and-forget goroutines that may still read it after the test returns.
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
