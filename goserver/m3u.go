package main

// m3u.go — parses Extended M3U (M3U8) IPTV playlist text into a flat list
// of channels. Handles the real-world shape of these files: an #EXTINF
// line (duration, quoted key="value" attributes, then a trailing
// ",Display Name") immediately followed by the stream URL line, with
// occasional VLC-only directives (#EXTGRP, #EXTVLCOPT, ...) mixed in that
// should be tolerated, not treated as errors.
//
// Port of lib/m3u_parser.rb -- keep behavior identical if either changes.

import (
	"bufio"
	"regexp"
	"strings"
)

type Channel struct {
	Name  string `json:"name"`
	Logo  string `json:"logo,omitempty"`
	Group string `json:"group,omitempty"`
	TvgID string `json:"tvg_id,omitempty"`
	URL   string `json:"url"`
}

var attrRe = regexp.MustCompile(`([\w-]+)="([^"]*)"`)

func parseM3U(text string) []Channel {
	channels := []Channel{}
	var pending *Channel

	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // real playlists can have very long lines
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "#EXTINF:"):
			ch := parseExtinf(line)
			pending = &ch
		case strings.HasPrefix(line, "#EXTGRP:"):
			if pending != nil && pending.Group == "" {
				pending.Group = strings.TrimSpace(strings.TrimPrefix(line, "#EXTGRP:"))
			}
		case strings.HasPrefix(line, "#"):
			// #EXTM3U, #EXTVLCOPT, #EXT-X-*, etc -- not needed for channel listing
		default:
			// A bare, non-comment line is the stream URL for the preceding
			// #EXTINF block. A URL with no preceding #EXTINF is malformed --
			// skip it rather than guessing a name.
			if pending != nil {
				pending.URL = line
				channels = append(channels, *pending)
			}
			pending = nil
		}
	}
	return channels
}

func parseExtinf(line string) Channel {
	body := strings.TrimPrefix(line, "#EXTINF:")
	attrs := map[string]string{}
	for _, m := range attrRe.FindAllStringSubmatch(body, -1) {
		attrs[m[1]] = m[2]
	}
	// Strip matched attributes out before splitting on the name comma, so a
	// comma *inside* a quoted attribute value (rare but real) doesn't break
	// the split.
	remainder := attrRe.ReplaceAllString(body, "")
	name := ""
	if idx := strings.Index(remainder, ","); idx >= 0 {
		name = strings.TrimSpace(remainder[idx+1:])
	}
	if name == "" {
		name = strings.TrimSpace(attrs["tvg-name"])
	}
	if name == "" {
		name = "Unnamed channel"
	}

	return Channel{
		Name:  name,
		Logo:  strings.TrimSpace(attrs["tvg-logo"]),
		Group: strings.TrimSpace(attrs["group-title"]),
		TvgID: strings.TrimSpace(attrs["tvg-id"]),
	}
}
