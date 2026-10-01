package main

// history.go — remembers the playlist URLs that have been loaded, in a small
// JSON file on disk, so the UI can offer them again next time.
//
// Stored server-side (not in the browser) so the history survives clearing
// site data, is shared by every browser/device that uses this server, and --
// in Docker -- lives on a mounted volume rather than inside the container.
//
// The file holds playlist URLs, which for Xtream-style providers embed the
// account's username/password in the query string, so it is written 0600 and
// its directory 0700.

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxHistory = 20

type historyEntry struct {
	URL      string `json:"url"`
	LastUsed int64  `json:"lastUsed"` // epoch milliseconds
}

type historyStore struct {
	mu      sync.Mutex
	path    string
	entries []historyEntry // most recently used first
}

// dataDir is where persistent state lives: $IPTV_DATA_DIR if set (the Docker
// image sets it to /data), else the per-user config dir (on macOS,
// ~/Library/Application Support/IPTV Player), else ./data.
func dataDir() string {
	if d := os.Getenv("IPTV_DATA_DIR"); d != "" {
		return d
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "IPTV Player")
	}
	return "data"
}

// newHistoryStore loads the history file at path. A missing file is normal
// (first run); an unreadable or corrupt one is logged and treated as empty
// rather than stopping the app from starting -- the next successful load
// rewrites it.
func newHistoryStore(path string) *historyStore {
	h := &historyStore{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("history: cannot read %s: %v", path, err)
		}
		return h
	}
	if err := json.Unmarshal(data, &h.entries); err != nil {
		log.Printf("history: ignoring corrupt %s: %v", path, err)
		h.entries = nil
	}
	return h
}

// List returns a copy of the history, most recently used first.
func (h *historyStore) List() []historyEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]historyEntry{}, h.entries...)
}

// Add records url as just used: it moves to the front (no duplicates) and
// the oldest entries beyond maxHistory are dropped.
func (h *historyStore) Add(url string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	next := []historyEntry{{URL: url, LastUsed: time.Now().UnixMilli()}}
	for _, e := range h.entries {
		if e.URL != url {
			next = append(next, e)
		}
	}
	if len(next) > maxHistory {
		next = next[:maxHistory]
	}
	return h.commit(next)
}

// Remove deletes url from the history (a no-op if it isn't there).
func (h *historyStore) Remove(url string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	next := []historyEntry{}
	for _, e := range h.entries {
		if e.URL != url {
			next = append(next, e)
		}
	}
	return h.commit(next)
}

// commit writes next to disk and only then makes it the in-memory state, so
// a failed write can't leave memory and file disagreeing.
func (h *historyStore) commit(next []historyEntry) error {
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFileAtomic(h.path, data); err != nil {
		return err
	}
	h.entries = next
	return nil
}
