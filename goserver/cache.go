package main

// cache.go — keeps the parsed channel list of each playlist on disk, so
// starting the app (or picking a playlist from the history) shows the
// channels immediately instead of re-downloading and re-parsing a playlist
// that can be many megabytes. The list is only refreshed when the user asks
// (the Reload button -> ?refresh=1).
//
// One file per playlist URL, named by the URL's hash, under <data dir>/cache.
// Files for playlists that have fallen out of the history are pruned so the
// cache can't grow without bound.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type cachedPlaylist struct {
	Channels  []Channel `json:"channels"`
	FetchedAt int64     `json:"fetchedAt"` // epoch milliseconds
}

type playlistCache struct {
	dir string
}

func newPlaylistCache(dir string) *playlistCache {
	return &playlistCache{dir: dir}
}

func cacheFileName(url string) string {
	sum := sha256.Sum256([]byte(url))
	return hex.EncodeToString(sum[:]) + ".json"
}

func (c *playlistCache) path(url string) string {
	return filepath.Join(c.dir, cacheFileName(url))
}

// Get returns the cached playlist for url, if there is a usable one. A
// missing, unreadable or corrupt file is just a cache miss.
func (c *playlistCache) Get(url string) (*cachedPlaylist, bool) {
	data, err := os.ReadFile(c.path(url))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("cache: cannot read %s: %v", c.path(url), err)
		}
		return nil, false
	}
	var p cachedPlaylist
	if err := json.Unmarshal(data, &p); err != nil || len(p.Channels) == 0 {
		return nil, false
	}
	return &p, true
}

// Put stores channels as the cached playlist for url and returns the
// timestamp it was stored with.
func (c *playlistCache) Put(url string, channels []Channel) (int64, error) {
	p := cachedPlaylist{Channels: channels, FetchedAt: time.Now().UnixMilli()}
	data, err := json.Marshal(p)
	if err != nil {
		return 0, err
	}
	if err := writeFileAtomic(c.path(url), data); err != nil {
		return 0, err
	}
	return p.FetchedAt, nil
}

func (c *playlistCache) Delete(url string) {
	if err := os.Remove(c.path(url)); err != nil && !os.IsNotExist(err) {
		log.Printf("cache: cannot delete %s: %v", c.path(url), err)
	}
}

// Retain deletes every cached playlist whose URL is not in keep.
func (c *playlistCache) Retain(keep []string) {
	want := map[string]bool{}
	for _, u := range keep {
		want[cacheFileName(u)] = true
	}
	files, err := os.ReadDir(c.dir)
	if err != nil {
		return // no cache dir yet -- nothing to prune
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") || want[f.Name()] {
			continue
		}
		os.Remove(filepath.Join(c.dir, f.Name()))
	}
}
