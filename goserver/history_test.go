package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func urls(entries []historyEntry) []string {
	out := []string{}
	for _, e := range entries {
		out = append(out, e.URL)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestHistoryEmptyWhenNoFile(t *testing.T) {
	h := newHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	if got := h.List(); len(got) != 0 {
		t.Fatalf("expected empty history, got %v", got)
	}
}

func TestHistoryAddOrdersMostRecentFirstAndDedupes(t *testing.T) {
	h := newHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	for _, u := range []string{"http://a", "http://b", "http://a"} {
		if err := h.Add(u); err != nil {
			t.Fatal(err)
		}
	}
	if got := urls(h.List()); !equal(got, []string{"http://a", "http://b"}) {
		t.Fatalf("got %v", got)
	}
}

func TestHistoryCapsAtMax(t *testing.T) {
	h := newHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	for i := 0; i < maxHistory+5; i++ {
		if err := h.Add(fmt.Sprintf("http://host/%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	got := urls(h.List())
	if len(got) != maxHistory {
		t.Fatalf("expected %d entries, got %d", maxHistory, len(got))
	}
	if got[0] != fmt.Sprintf("http://host/%d", maxHistory+4) || got[maxHistory-1] != "http://host/5" {
		t.Fatalf("wrong entries kept: first=%s last=%s", got[0], got[maxHistory-1])
	}
}

func TestHistoryRemove(t *testing.T) {
	h := newHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	h.Add("http://a")
	h.Add("http://b")
	if err := h.Remove("http://a"); err != nil {
		t.Fatal(err)
	}
	if got := urls(h.List()); !equal(got, []string{"http://b"}) {
		t.Fatalf("got %v", got)
	}
	if err := h.Remove("http://not-there"); err != nil {
		t.Fatalf("removing a missing URL should be a no-op, got %v", err)
	}
}

func TestHistoryPersistsAcrossReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "history.json") // dir doesn't exist yet
	h := newHistoryStore(path)
	h.Add("http://a")
	h.Add("http://b")

	reloaded := newHistoryStore(path)
	if got := urls(reloaded.List()); !equal(got, []string{"http://b", "http://a"}) {
		t.Fatalf("got %v", got)
	}
}

func TestHistoryFileIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	h := newHistoryStore(path)
	h.Add("http://user:pass@a")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("history file mode = %o, want 600", perm)
	}
}

func TestHistoryCorruptFileIsTreatedAsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	os.WriteFile(path, []byte("{not json"), 0o600)
	h := newHistoryStore(path)
	if got := h.List(); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
	if err := h.Add("http://a"); err != nil {
		t.Fatal(err)
	}
	if got := urls(newHistoryStore(path).List()); !equal(got, []string{"http://a"}) {
		t.Fatalf("history not recovered after corrupt file: %v", got)
	}
}
