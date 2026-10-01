package main

// playlist.go — fetches a remote M3U playlist over HTTP(S) from the server
// side (never the browser), which is what actually lets this app work with
// real IPTV providers: almost none of them send CORS headers, so a direct
// browser fetch of the playlist URL would just fail silently.
//
// Port of lib/playlist_fetcher.rb -- keep behavior identical if either changes.

import (
	"fmt"
	"io"
	"time"
)

func fetchPlaylist(rawURL string) (string, error) {
	if !isHTTPURL(rawURL) {
		return "", fmt.Errorf("unsupported URL (must be http:// or https://)")
	}

	resp, _, err := doGetFollowingRedirects(rawURL, 30*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("playlist server returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
