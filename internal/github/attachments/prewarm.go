package attachments

import (
	"context"
	"net"
	"time"

	"github.com/cli/go-gh/v2/pkg/auth"
)

// cloudUploadHosts: both resolved; content-type picks one, server-side.
var cloudUploadHosts = []string{
	"objects-origin.githubusercontent.com",
	"github-production-user-asset-6210df.s3.amazonaws.com",
}

const prewarmTimeout = 5 * time.Second

type hostLookupFunc func(ctx context.Context, host string) ([]string, error)

// PrewarmUploadHostDNS resolves upload-host DNS in the background, DNS-only.
func PrewarmUploadHostDNS(ctx context.Context, host string) {
	prewarmUploadHostDNS(ctx, host, net.DefaultResolver.LookupHost)
}

// lookup is a param, not a package var, so test stubs don't race goroutines.
func prewarmUploadHostDNS(ctx context.Context, host string, lookup hostLookupFunc) {
	if auth.IsEnterprise(host) {
		return // GHES upload host isn't known until the policies response
	}

	for _, h := range cloudUploadHosts {
		go func(h string) {
			ctx, cancel := context.WithTimeout(ctx, prewarmTimeout)
			defer cancel()
			_, _ = lookup(ctx, h)
		}(h)
	}
}
