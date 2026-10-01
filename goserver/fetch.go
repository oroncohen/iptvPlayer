package main

// fetch.go — shared HTTP GET-with-manual-redirect-following helper used by
// playlist fetching, HLS proxying, and stream resolution. Redirects are
// followed manually (not via Go's built-in client redirect handling) so
// callers know exactly which URL content actually came from -- relative
// URIs inside e.g. an HLS manifest are relative to THAT, not the original
// URL if it redirected (link shorteners, region redirects, signed URL
// issuers are all common in real IPTV providers).
//
// doGetFollowingRedirects never raises/errors on a non-2xx final response
// -- it just returns whatever response it got, same as
// lib/stream_resolver.rb's get_following_redirects/head_following_redirects.
// Callers that need "non-2xx is an error" (the old stream_proxy.rb
// behavior) check resp.StatusCode themselves.

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const userAgent = "Mozilla/5.0 (compatible; IPTVPlayer/1.0)"
const maxRedirects = 5

func isHTTPURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}

// doGetFollowingRedirects performs a GET, following up to maxRedirects 3xx
// responses. Returns the final response (caller must close resp.Body) and
// the URL that response actually came from.
func doGetFollowingRedirects(rawURL string, timeout time.Duration) (resp *http.Response, finalURL string, err error) {
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	current := rawURL
	for i := 0; i <= maxRedirects; i++ {
		req, err := http.NewRequest(http.MethodGet, current, nil)
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			return nil, "", err
		}

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := resp.Header.Get("Location")
			resp.Body.Close()
			if loc == "" {
				return nil, "", fmt.Errorf("redirect with no Location header")
			}
			next, err := resolveURLString(current, loc)
			if err != nil {
				return nil, "", err
			}
			current = next
			continue
		}

		return resp, current, nil
	}
	return nil, "", fmt.Errorf("too many redirects")
}

func resolveURLString(base, ref string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	r, err := b.Parse(ref)
	if err != nil {
		return "", err
	}
	return r.String(), nil
}
