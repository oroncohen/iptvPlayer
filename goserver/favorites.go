package main

// favorites.go — the channels the user has starred, kept in a small JSON
// file in the data dir (next to history.json), so they survive restarts and
// container rebuilds and are shared by every browser using this server.
//
// A favorite is stored as a full copy of the channel (name, logo, group,
// stream URL), not just a reference into one playlist: that way favorites
// keep working after a playlist is reloaded, changed or removed from the
// history. A channel's stream URL is its identity.
//
// Stream URLs usually embed the provider account's credentials, so the file
// is written 0600 like the rest of the data dir.

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type favoritesStore struct {
	mu    sync.Mutex
	path  string
	items []Channel // in the order they were added
}

// newFavoritesStore loads the favorites file at path. As with the history, a
// missing file is normal and a corrupt one is logged and treated as empty.
func newFavoritesStore(path string) *favoritesStore {
	f := &favoritesStore{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("favorites: cannot read %s: %v", path, err)
		}
		return f
	}
	if err := json.Unmarshal(data, &f.items); err != nil {
		log.Printf("favorites: ignoring corrupt %s: %v", path, err)
		f.items = nil
	}
	return f
}

// List returns a copy of the favorites, in the order they were added.
func (f *favoritesStore) List() []Channel {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Channel{}, f.items...)
}

// Add stars c. Starring a channel that is already a favorite refreshes its
// stored details but keeps its position.
func (f *favoritesStore) Add(c Channel) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	next := append([]Channel{}, f.items...)
	replaced := false
	for i, e := range next {
		if e.URL == c.URL {
			next[i] = c
			replaced = true
			break
		}
	}
	if !replaced {
		next = append(next, c)
	}
	return f.commit(next)
}

// Remove un-stars the channel with this stream URL (a no-op if it isn't one).
func (f *favoritesStore) Remove(url string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	next := []Channel{}
	for _, e := range f.items {
		if e.URL != url {
			next = append(next, e)
		}
	}
	return f.commit(next)
}

// commit writes next to disk and only then makes it the in-memory state.
func (f *favoritesStore) commit(next []Channel) error {
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFileAtomic(f.path, data); err != nil {
		return err
	}
	f.items = next
	return nil
}
