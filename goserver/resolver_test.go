package main

import "testing"

func TestHlsByExtensionRecognizesM3u8(t *testing.T) {
	cases := map[string]bool{
		"http://x/stream.m3u8":            true,
		"http://x/stream.m3u8?token=abc":  true,
		"http://x/stream.mp4":             false,
		"http://x/user/pass/12345":        false,
	}
	for url, want := range cases {
		if got := hlsByExtension(url); got != want {
			t.Errorf("hlsByExtension(%q) = %v, want %v", url, got, want)
		}
	}
}

func TestXtreamHLSCandidateForBareThreeSegmentPath(t *testing.T) {
	got := xtreamHLSCandidate("http://host:8080/myuser/mypass/50637")
	want := "http://host:8080/live/myuser/mypass/50637.m3u8"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestXtreamHLSCandidateStripsExistingExtensionBeforeAppendingM3u8(t *testing.T) {
	got := xtreamHLSCandidate("http://host:8080/myuser/mypass/50637.ts")
	want := "http://host:8080/live/myuser/mypass/50637.m3u8"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestXtreamHLSCandidateReturnsEmptyForNonThreeSegmentPaths(t *testing.T) {
	for _, u := range []string{"http://host/a/b/c/d", "http://host/onlyone", "http://host/"} {
		if got := xtreamHLSCandidate(u); got != "" {
			t.Errorf("xtreamHLSCandidate(%q) = %q, want empty", u, got)
		}
	}
}

func TestXtreamHLSCandidateReturnsEmptyIfAlreadyUnderLive(t *testing.T) {
	if got := xtreamHLSCandidate("http://host/live/user/pass"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
