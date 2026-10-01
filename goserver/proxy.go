package main

// proxy.go — lets the browser play HLS (.m3u8) channels without hitting
// CORS: the browser only ever talks to *us*, and we do the real fetching
// to the IPTV provider server-side.
//
// Two pieces:
//   rewriteManifest(url)  fetch a .m3u8, rewrite every URI it references
//                          (segments, sub-playlists, encryption keys) into
//                          an absolute URL routed back through our own
//                          /api/proxy/* endpoints, return the rewritten text
//   fetchSegment(url)     fetch one proxied resource (a .ts/.m4s segment, a
//                          key file, ...) fully buffered, for the caller to
//                          write into the HTTP response verbatim
//
// Segments/manifests in real HLS are always small (a few seconds of video
// each), so buffering each one fully in memory per request is fine -- no
// need for a streaming/chunked implementation here, and it avoids ever
// starting to write a response before knowing whether the upstream fetch
// actually succeeded.
//
// Port of lib/stream_proxy.rb -- keep behavior identical if either changes.

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var extXURIRe = regexp.MustCompile(`URI="([^"]+)"`)

func rewriteManifest(rawURL string) (string, error) {
	resp, finalURL, err := doGetFollowingRedirects(rawURL, 20*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Rewrite against the URL the manifest actually came from (post-redirect)
	// -- relative segment URIs inside it are relative to that, not the
	// original shortener/redirect URL.
	base, err := url.Parse(finalURL)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(body), "\n")
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = rewriteLine(strings.TrimRight(line, "\r"), base)
	}
	return strings.Join(out, "\n"), nil
}

func rewriteLine(line string, base *url.URL) string {
	switch {
	case strings.HasPrefix(line, "#EXT-X-KEY") || strings.HasPrefix(line, "#EXT-X-MAP") || strings.HasPrefix(line, "#EXT-X-MEDIA"):
		return extXURIRe.ReplaceAllStringFunc(line, func(m string) string {
			sub := extXURIRe.FindStringSubmatch(m)
			abs := resolveAgainst(base, sub[1])
			return `URI="` + proxyURLFor(abs) + `"`
		})
	case strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "":
		return line
	default:
		abs := resolveAgainst(base, line)
		return proxyURLFor(abs)
	}
}

func resolveAgainst(base *url.URL, ref string) string {
	u, err := base.Parse(ref)
	if err != nil {
		return ref
	}
	return u.String()
}

func proxyURLFor(absURL string) string {
	path := strings.ToLower(strings.SplitN(absURL, "?", 2)[0])
	endpoint := "segment"
	if strings.HasSuffix(path, ".m3u8") {
		endpoint = "manifest"
	}
	return "/api/proxy/" + endpoint + "?url=" + url.QueryEscape(absURL)
}

// fetchSegment fetches a proxied resource fully into memory (segments/keys
// are always small -- see file header) and returns its status code,
// content type, and body for the caller to write out.
func fetchSegment(rawURL string) (status int, contentType string, body []byte, err error) {
	resp, _, err := doGetFollowingRedirects(rawURL, 20*time.Second)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, "", nil, fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", nil, err
	}
	return resp.StatusCode, ct, b, nil
}
