package main

import (
	"os"
	"path/filepath"
	"testing"
)

func sampleChannels() []Channel {
	return []Channel{{Name: "One", URL: "http://s/1"}, {Name: "Two", Group: "News", URL: "http://s/2"}}
}

func TestCacheMissThenHit(t *testing.T) {
	c := newPlaylistCache(filepath.Join(t.TempDir(), "cache"))
	if _, ok := c.Get("http://p/list.m3u"); ok {
		t.Fatal("expected a miss on an empty cache")
	}
	at, err := c.Put("http://p/list.m3u", sampleChannels())
	if err != nil {
		t.Fatal(err)
	}
	got, ok := c.Get("http://p/list.m3u")
	if !ok || len(got.Channels) != 2 || got.Channels[1].Group != "News" || got.FetchedAt != at {
		t.Fatalf("unexpected cache result: %+v ok=%v", got, ok)
	}
}

func TestCacheKeepsPlaylistsSeparate(t *testing.T) {
	c := newPlaylistCache(t.TempDir())
	c.Put("http://a", sampleChannels()[:1])
	c.Put("http://b", sampleChannels())
	a, _ := c.Get("http://a")
	b, _ := c.Get("http://b")
	if len(a.Channels) != 1 || len(b.Channels) != 2 {
		t.Fatalf("a=%d b=%d", len(a.Channels), len(b.Channels))
	}
}

func TestCachePutReplaces(t *testing.T) {
	c := newPlaylistCache(t.TempDir())
	c.Put("http://a", sampleChannels())
	c.Put("http://a", sampleChannels()[:1])
	got, _ := c.Get("http://a")
	if len(got.Channels) != 1 {
		t.Fatalf("expected replaced list, got %d channels", len(got.Channels))
	}
}

func TestCacheFileIsPrivate(t *testing.T) {
	dir := t.TempDir()
	c := newPlaylistCache(dir)
	c.Put("http://user:pass@a", sampleChannels())
	info, err := os.Stat(c.path("http://user:pass@a"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("cache file mode = %o, want 600", perm)
	}
}

func TestCacheCorruptFileIsAMiss(t *testing.T) {
	c := newPlaylistCache(t.TempDir())
	os.WriteFile(c.path("http://a"), []byte("{not json"), 0o600)
	if _, ok := c.Get("http://a"); ok {
		t.Fatal("corrupt cache file should be a miss")
	}
}

func TestCacheDeleteAndRetain(t *testing.T) {
	c := newPlaylistCache(t.TempDir())
	for _, u := range []string{"http://a", "http://b", "http://c"} {
		c.Put(u, sampleChannels())
	}
	c.Delete("http://a")
	c.Delete("http://never-cached") // must not fail
	c.Retain([]string{"http://b"})
	if _, ok := c.Get("http://a"); ok {
		t.Error("a should be deleted")
	}
	if _, ok := c.Get("http://b"); !ok {
		t.Error("b should be retained")
	}
	if _, ok := c.Get("http://c"); ok {
		t.Error("c should be pruned")
	}
}
