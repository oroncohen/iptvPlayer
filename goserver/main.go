package main

// main.go — IPTV Player: local web app, paste an IPTV M3U playlist URL,
// browse its channels, click one to play it.
//
// Run:  ./iptvplayer [port]   (default port 4568)
// Then open http://localhost:4568 in your browser.
//
// Why there's a backend at all instead of a static page: real IPTV
// providers essentially never send CORS headers, so a browser can't
// fetch() the playlist file directly, and hls.js can't fetch() the HLS
// manifest/segments directly either. Both go through this server instead,
// which has no such restriction.
//
// This is a from-scratch Go port of the original Ruby/WEBrick backend
// (server.rb + lib/*.rb), written specifically so the whole app can ship
// as a single static binary with zero runtime dependencies -- no Ruby (or
// any other interpreter) needs to be present on the machine that runs it,
// which matters for packaging this as a standalone installer for other
// Macs. Keep this in sync with the Ruby version if that's still used
// anywhere, or retire the Ruby version once this is confirmed to fully
// replace it.

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func jsonResponse(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func proxyError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusBadGateway)
	fmt.Fprintf(w, "proxy error: %s", err.Error())
}

func findPublicDir() string {
	// The compiled binary and its public/ folder are always siblings in the
	// shipped layout (plain folder or inside a .app's Resources) -- resolve
	// relative to the binary's own real path so this works no matter where
	// it's installed. Falls back to a plain "public" (relative to the
	// current directory) for local dev via `go run .`, where os.Executable()
	// points at a temp build directory instead.
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "public")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "public"
}

func main() {
	port := 4568
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil {
			port = p
		}
	}

	publicDir := findPublicDir()

	history := newHistoryStore(filepath.Join(dataDir(), "history.json"))
	cache := newPlaylistCache(filepath.Join(dataDir(), "cache"))
	favorites := newFavoritesStore(filepath.Join(dataDir(), "favorites.json"))

	mux := http.NewServeMux()

	// GET /api/playlist?url=...           -> the saved copy if there is one,
	//                                        else downloads + saves it
	// GET /api/playlist?url=...&refresh=1 -> always re-downloads (Reload)
	mux.HandleFunc("/api/playlist", func(w http.ResponseWriter, r *http.Request) {
		u := strings.TrimSpace(r.URL.Query().Get("url"))
		if u == "" {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "missing url param"})
			return
		}
		refresh := r.URL.Query().Get("refresh") == "1"

		// Recording the use (history order + cache pruning) must never fail
		// the load itself, so problems are only logged.
		recordUse := func() {
			if err := history.Add(u); err != nil {
				log.Printf("history: could not save %s: %v", history.path, err)
				return
			}
			kept := []string{}
			for _, e := range history.List() {
				kept = append(kept, e.URL)
			}
			cache.Retain(kept)
		}

		if !refresh {
			if saved, ok := cache.Get(u); ok {
				recordUse()
				jsonResponse(w, http.StatusOK, map[string]interface{}{
					"channels": saved.Channels, "count": len(saved.Channels),
					"fetchedAt": saved.FetchedAt, "cached": true,
				})
				return
			}
		}

		body, err := fetchPlaylist(u)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		channels := parseM3U(body)
		if len(channels) == 0 {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "no channels found -- is this a valid M3U playlist URL?"})
			return
		}
		// Only playlists that actually parsed are saved, so a failed reload
		// leaves the previous good copy in place.
		fetchedAt, err := cache.Put(u, channels)
		if err != nil {
			log.Printf("cache: could not save playlist: %v", err)
			fetchedAt = time.Now().UnixMilli()
		}
		recordUse()
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"channels": channels, "count": len(channels),
			"fetchedAt": fetchedAt, "cached": false,
		})
	})

	mux.HandleFunc("GET /api/history", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"history": history.List()})
	})

	mux.HandleFunc("DELETE /api/history", func(w http.ResponseWriter, r *http.Request) {
		u := strings.TrimSpace(r.URL.Query().Get("url"))
		if u == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "missing url param"})
			return
		}
		if err := history.Remove(u); err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		cache.Delete(u)
		jsonResponse(w, http.StatusOK, map[string]interface{}{"history": history.List()})
	})

	// Favorite channels. POST takes the channel as JSON ({name, logo, group,
	// tvg_id, url}); DELETE takes the channel's stream ?url=. Every response
	// is the full updated list, so the UI can just adopt it.
	mux.HandleFunc("GET /api/favorites", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"favorites": favorites.List()})
	})

	mux.HandleFunc("POST /api/favorites", func(w http.ResponseWriter, r *http.Request) {
		var c Channel
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&c); err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid channel JSON"})
			return
		}
		c.URL = strings.TrimSpace(c.URL)
		c.Name = strings.TrimSpace(c.Name)
		if c.URL == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "channel needs a url"})
			return
		}
		if c.Name == "" {
			c.Name = c.URL
		}
		if err := favorites.Add(c); err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{"favorites": favorites.List()})
	})

	mux.HandleFunc("DELETE /api/favorites", func(w http.ResponseWriter, r *http.Request) {
		u := strings.TrimSpace(r.URL.Query().Get("url"))
		if u == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "missing url param"})
			return
		}
		if err := favorites.Remove(u); err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{"favorites": favorites.List()})
	})

	mux.HandleFunc("/api/resolve", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("url")
		if u == "" {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "missing url param"})
			return
		}
		jsonResponse(w, http.StatusOK, resolveStream(u))
	})

	mux.HandleFunc("/api/proxy/manifest", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("url")
		if u == "" {
			proxyError(w, fmt.Errorf("missing url param"))
			return
		}
		body, err := rewriteManifest(u)
		if err != nil {
			proxyError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	mux.HandleFunc("/api/proxy/segment", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("url")
		if u == "" {
			proxyError(w, fmt.Errorf("missing url param"))
			return
		}
		status, contentType, body, err := fetchSegment(u)
		if err != nil {
			proxyError(w, err)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		w.Write(body)
	})

	mux.Handle("/", http.FileServer(http.Dir(publicDir)))

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("IPTV Player running at http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(addr, mux))
}
