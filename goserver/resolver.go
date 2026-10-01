package main

// resolver.go — figures out *how* a channel's raw playlist URL should
// actually be played in a browser.
//
// Real IPTV playlists mix three shapes of stream URL:
//  1. Already HLS (.m3u8)                    -> proxy it through proxy.go
//  2. A directly-playable file (mp4/webm/...) -> <video src> it, no proxy needed
//  3. A bare Xtream-Codes stream URL
//     (http://host:port/user/pass/streamId, no extension) that serves raw,
//     continuous MPEG-TS (`video/mp2t`) -- which browsers cannot decode at
//     all, HLS.js or not; MSE only understands fragmented containers.
//     Xtream-Codes panels virtually always *also* expose the same channel
//     as real HLS at http://host:port/live/user/pass/streamId.m3u8 -- this
//     was verified directly against a real panel (see project session
//     notes): the bare URL redirected to a `video/mp2t` response, while the
//     `/live/.../id.m3u8` URL for the same stream ID redirected to a real
//     `#EXTM3U` manifest. So for shape 3 we try that sibling URL first.
//
// peekContentType only reads response headers (never the body), which
// matters because a live TS stream's body never ends.
//
// Port of lib/stream_resolver.rb -- keep behavior identical if either changes.

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

var directContentTypes = map[string]bool{
	"video/mp4": true, "video/webm": true, "video/quicktime": true,
	"audio/mpeg": true, "audio/mp4": true, "audio/aac": true,
	"audio/ogg": true, "audio/wav": true,
}

type ResolveResult struct {
	Type   string `json:"type"`
	URL    string `json:"url,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func resolveStream(rawURL string) ResolveResult {
	if hlsByExtension(rawURL) {
		return ResolveResult{Type: "hls", URL: rawURL}
	}

	contentType := peekContentType(rawURL)
	if strings.Contains(contentType, "mpegurl") {
		return ResolveResult{Type: "hls", URL: rawURL}
	}
	if contentType != "" && directContentTypes[contentType] {
		return ResolveResult{Type: "direct", URL: rawURL}
	}

	if candidate := xtreamHLSCandidate(rawURL); candidate != "" && isHLSManifest(candidate) {
		return ResolveResult{Type: "hls", URL: candidate}
	}

	if contentType == "video/mp2t" {
		return ResolveResult{Type: "unsupported", Reason: "This is a raw MPEG-TS stream and no HLS version of it could be found. " +
			"Browsers can't play raw MPEG-TS directly (try VLC or a dedicated IPTV app instead)."}
	}
	reason := "Unrecognized stream format."
	if contentType != "" {
		reason = fmt.Sprintf("Unrecognized stream format (%s).", contentType)
	}
	return ResolveResult{Type: "unsupported", Reason: reason}
}

func hlsByExtension(rawURL string) bool {
	path := strings.ToLower(strings.SplitN(rawURL, "?", 2)[0])
	return strings.HasSuffix(path, ".m3u8")
}

// http://host:port/user/pass/streamId -> http://host:port/live/user/pass/streamId.m3u8
// Only fires for that exact 3-segment bare shape, so it doesn't misfire on
// arbitrary direct-file URLs with different path shapes. Returns "" if the
// URL doesn't match that shape or can't be parsed.
func xtreamHLSCandidate(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := []string{}
	for _, p := range strings.Split(u.Path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) != 3 || parts[0] == "live" {
		return ""
	}

	user, pass, id := parts[0], parts[1], parts[2]
	if idx := strings.LastIndex(id, "."); idx > 0 {
		id = id[:idx]
	}
	u.Path = "/live/" + user + "/" + pass + "/" + id + ".m3u8"
	return u.String()
}

func isHLSManifest(rawURL string) bool {
	resp, _, err := doGetFollowingRedirects(rawURL, 8*time.Second)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB cap, plenty for a manifest
	if err != nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(string(body)), "#EXTM3U")
}

func peekContentType(rawURL string) string {
	resp, _, err := doGetFollowingRedirects(rawURL, 8*time.Second)
	if err != nil {
		return ""
	}
	defer resp.Body.Close() // never read the body -- see file header
	ct := resp.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = ct[:idx]
	}
	return strings.ToLower(strings.TrimSpace(ct))
}
