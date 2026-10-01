package main

import "testing"

func TestParsesBasicPlaylist(t *testing.T) {
	text := "#EXTM3U\n" +
		"#EXTINF:-1 tvg-id=\"ch1.id\" tvg-logo=\"http://x/logo1.png\" group-title=\"News\",Channel One\n" +
		"http://server/stream1.m3u8\n" +
		"#EXTINF:-1 tvg-id=\"ch2.id\" group-title=\"Sports\",Channel Two\n" +
		"http://server/stream2/index.m3u8\n"

	channels := parseM3U(text)
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}

	c0 := channels[0]
	if c0.Name != "Channel One" || c0.Logo != "http://x/logo1.png" || c0.Group != "News" ||
		c0.TvgID != "ch1.id" || c0.URL != "http://server/stream1.m3u8" {
		t.Fatalf("unexpected channel 0: %+v", c0)
	}

	c1 := channels[1]
	if c1.Name != "Channel Two" || c1.Logo != "" || c1.Group != "Sports" {
		t.Fatalf("unexpected channel 1: %+v", c1)
	}
}

func TestNameWithCommaInsideQuotedAttributeDoesNotBreakSplit(t *testing.T) {
	text := "#EXTINF:-1 tvg-logo=\"http://x/logo.png?a=1,2\" group-title=\"News, Live\",Channel, With Comma\n" +
		"http://server/stream.m3u8\n"

	channels := parseM3U(text)
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
	if channels[0].Name != "Channel, With Comma" {
		t.Errorf("expected name 'Channel, With Comma', got %q", channels[0].Name)
	}
	if channels[0].Group != "News, Live" {
		t.Errorf("expected group 'News, Live', got %q", channels[0].Group)
	}
}

func TestFallsBackToTvgNameWhenDisplayNameMissing(t *testing.T) {
	text := "#EXTINF:-1 tvg-name=\"Fallback Name\" group-title=\"Misc\",\n" +
		"http://server/stream.m3u8\n"

	channels := parseM3U(text)
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
	if channels[0].Name != "Fallback Name" {
		t.Errorf("expected 'Fallback Name', got %q", channels[0].Name)
	}
}

func TestIgnoresVlcAndExtgrpDirectivesWithoutBreakingParsing(t *testing.T) {
	text := "#EXTM3U\n" +
		"#EXTINF:-1 group-title=\"News\",Channel One\n" +
		"#EXTGRP:News\n" +
		"#EXTVLCOPT:http-user-agent=SomeAgent\n" +
		"http://server/stream1.m3u8\n"

	channels := parseM3U(text)
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
	if channels[0].URL != "http://server/stream1.m3u8" {
		t.Errorf("unexpected url: %q", channels[0].URL)
	}
}

func TestStrayURLWithNoPrecedingExtinfIsSkipped(t *testing.T) {
	text := "http://server/orphan.m3u8\n" +
		"#EXTINF:-1,Channel One\n" +
		"http://server/stream1.m3u8\n"

	channels := parseM3U(text)
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
	if channels[0].Name != "Channel One" {
		t.Errorf("unexpected name: %q", channels[0].Name)
	}
}

func TestEmptyPlaylistReturnsEmptySlice(t *testing.T) {
	channels := parseM3U("#EXTM3U\n")
	if len(channels) != 0 {
		t.Fatalf("expected 0 channels, got %d", len(channels))
	}
}
