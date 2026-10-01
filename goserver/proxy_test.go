package main

import (
	"net/url"
	"strings"
	"testing"
)

func base(t *testing.T) *url.URL {
	t.Helper()
	u, err := url.Parse("https://cdn.example.com/live/channel/index.m3u8")
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestPassesThroughCommentsAndBlankLines(t *testing.T) {
	b := base(t)
	if got := rewriteLine("#EXTM3U", b); got != "#EXTM3U" {
		t.Errorf("got %q", got)
	}
	if got := rewriteLine("", b); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestRewritesRelativeSegmentURIToSegmentProxy(t *testing.T) {
	got := rewriteLine("segment001.ts", base(t))
	want := "/api/proxy/segment?url=" + url.QueryEscape("https://cdn.example.com/live/channel/segment001.ts")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRewritesRelativeSubManifestToManifestProxy(t *testing.T) {
	got := rewriteLine("1080p/playlist.m3u8", base(t))
	want := "/api/proxy/manifest?url=" + url.QueryEscape("https://cdn.example.com/live/channel/1080p/playlist.m3u8")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRewritesAbsolutePathURIAgainstHost(t *testing.T) {
	got := rewriteLine("/live/channel/segment002.ts", base(t))
	want := "/api/proxy/segment?url=" + url.QueryEscape("https://cdn.example.com/live/channel/segment002.ts")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRewritesExtXKeyURI(t *testing.T) {
	line := `#EXT-X-KEY:METHOD=AES-128,URI="key/enc.key",IV=0x1234`
	got := rewriteLine(line, base(t))
	wantURI := `URI="/api/proxy/segment?url=` + url.QueryEscape("https://cdn.example.com/live/channel/key/enc.key") + `"`
	if !strings.Contains(got, wantURI) {
		t.Errorf("got %q, want it to contain %q", got, wantURI)
	}
	if !strings.Contains(got, "METHOD=AES-128") {
		t.Errorf("expected rest of tag preserved, got %q", got)
	}
}

func TestRewritesExtXMediaSubtitleURIToManifestProxy(t *testing.T) {
	line := `#EXT-X-MEDIA:TYPE=SUBTITLES,GROUP-ID="subs",NAME="English",URI="subs/en/playlist.m3u8"`
	got := rewriteLine(line, base(t))
	wantURI := `URI="/api/proxy/manifest?url=` + url.QueryEscape("https://cdn.example.com/live/channel/subs/en/playlist.m3u8") + `"`
	if !strings.Contains(got, wantURI) {
		t.Errorf("got %q, want it to contain %q", got, wantURI)
	}
}

func TestAbsoluteURIInSourceIsPreservedVerbatimThenProxied(t *testing.T) {
	got := rewriteLine("https://other-cdn.example.com/seg.ts", base(t))
	want := "/api/proxy/segment?url=" + url.QueryEscape("https://other-cdn.example.com/seg.ts")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProxyURLForRoutesByExtensionIgnoringQueryString(t *testing.T) {
	if got, want := proxyURLFor("https://x/y.m3u8?token=abc"), "/api/proxy/manifest?url="+url.QueryEscape("https://x/y.m3u8?token=abc"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := proxyURLFor("https://x/y.ts?token=abc"), "/api/proxy/segment?url="+url.QueryEscape("https://x/y.ts?token=abc"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

