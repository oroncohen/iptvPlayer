package main

import (
	"os"
	"path/filepath"
	"testing"
)

func favURLs(items []Channel) []string {
	out := []string{}
	for _, c := range items {
		out = append(out, c.URL)
	}
	return out
}

func TestFavoritesEmptyWhenNoFile(t *testing.T) {
	f := newFavoritesStore(filepath.Join(t.TempDir(), "favorites.json"))
	if got := f.List(); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestFavoritesAddKeepsInsertionOrder(t *testing.T) {
	f := newFavoritesStore(filepath.Join(t.TempDir(), "favorites.json"))
	for _, u := range []string{"http://s/a", "http://s/b", "http://s/c"} {
		if err := f.Add(Channel{Name: u, URL: u}); err != nil {
			t.Fatal(err)
		}
	}
	if got := favURLs(f.List()); !equal(got, []string{"http://s/a", "http://s/b", "http://s/c"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFavoritesAddingAgainRefreshesDetailsInPlace(t *testing.T) {
	f := newFavoritesStore(filepath.Join(t.TempDir(), "favorites.json"))
	f.Add(Channel{Name: "A", URL: "http://s/a"})
	f.Add(Channel{Name: "B", URL: "http://s/b"})
	f.Add(Channel{Name: "A renamed", Group: "News", URL: "http://s/a"})
	got := f.List()
	if len(got) != 2 || got[0].Name != "A renamed" || got[0].Group != "News" || got[1].URL != "http://s/b" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestFavoritesRemove(t *testing.T) {
	f := newFavoritesStore(filepath.Join(t.TempDir(), "favorites.json"))
	f.Add(Channel{Name: "A", URL: "http://s/a"})
	f.Add(Channel{Name: "B", URL: "http://s/b"})
	if err := f.Remove("http://s/a"); err != nil {
		t.Fatal(err)
	}
	if got := favURLs(f.List()); !equal(got, []string{"http://s/b"}) {
		t.Fatalf("got %v", got)
	}
	if err := f.Remove("http://s/not-there"); err != nil {
		t.Fatalf("removing a missing favorite should be a no-op, got %v", err)
	}
}

func TestFavoritesPersistAcrossReloadWithAllFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "favorites.json")
	f := newFavoritesStore(path)
	f.Add(Channel{Name: "A", Logo: "http://l/a.png", Group: "News", TvgID: "a.id", URL: "http://s/a"})
	got := newFavoritesStore(path).List()
	if len(got) != 1 || got[0] != (Channel{Name: "A", Logo: "http://l/a.png", Group: "News", TvgID: "a.id", URL: "http://s/a"}) {
		t.Fatalf("got %+v", got)
	}
}

func TestFavoritesFileIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	f := newFavoritesStore(path)
	f.Add(Channel{Name: "A", URL: "http://user:pass@s/a"})
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("mode = %o, want 600", perm)
	}
}

func TestFavoritesCorruptFileIsTreatedAsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	os.WriteFile(path, []byte("{not json"), 0o600)
	f := newFavoritesStore(path)
	if got := f.List(); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
	if err := f.Add(Channel{Name: "A", URL: "http://s/a"}); err != nil {
		t.Fatal(err)
	}
	if got := favURLs(newFavoritesStore(path).List()); !equal(got, []string{"http://s/a"}) {
		t.Fatalf("not recovered: %v", got)
	}
}
