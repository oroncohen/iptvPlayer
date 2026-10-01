const LAST_PLAYLIST_URL_KEY = 'iptv_player_last_playlist_url';

// Fetches and parses the playlist at `url` via the backend (never directly
// from the browser: providers essentially never send CORS headers).
//
// The backend keeps a saved copy of every playlist it has loaded and serves
// that by default, so opening a known playlist is instant. Pass
// { refresh: true } to make it re-download from the provider instead.
//
// Returns { channels: [{name, logo, group, tvg_id, url}], fetchedAt, cached }
// where fetchedAt is when the copy was downloaded (epoch ms).
export async function loadPlaylist(url, { refresh = false } = {}) {
  const res = await fetch('/api/playlist?url=' + encodeURIComponent(url) + (refresh ? '&refresh=1' : ''));
  const data = await res.json();
  if (data.error) throw new Error(data.error);
  return { channels: data.channels, fetchedAt: data.fetchedAt, cached: !!data.cached };
}

// Asks the backend how a channel's raw stream URL should actually be
// played (see lib/stream_resolver.rb). This has to be a server round-trip,
// not a client-side .m3u8-extension guess: plenty of real IPTV URLs have no
// extension at all (bare Xtream-Codes stream URLs) and serve raw MPEG-TS,
// which browsers can't play at all -- but the same panel almost always also
// exposes a real HLS manifest for that channel at a different, discoverable
// URL, which only the backend can safely probe for (CORS, and peeking a
// live stream's headers without downloading it).
//
// Returns:
//   { type: 'hls', url }         -- proxy `url` through /api/proxy/manifest
//   { type: 'direct', url }      -- <video src> `url` as-is
//   { type: 'unsupported', reason }
export async function resolveStream(streamUrl) {
  const res = await fetch('/api/resolve?url=' + encodeURIComponent(streamUrl));
  const data = await res.json();
  if (data.error) throw new Error(data.error);
  return data;
}

export function manifestProxyUrl(hlsUrl) {
  return '/api/proxy/manifest?url=' + encodeURIComponent(hlsUrl);
}

export function getLastPlaylistUrl() {
  try {
    return localStorage.getItem(LAST_PLAYLIST_URL_KEY) || '';
  } catch {
    return '';
  }
}

export function saveLastPlaylistUrl(url) {
  try {
    localStorage.setItem(LAST_PLAYLIST_URL_KEY, url);
  } catch {
    // localStorage unavailable (private mode etc) -- not fatal, just no memory
  }
}

// Playlist history lives on the server (a small file in its data dir), not in
// the browser, so it survives clearing site data and is shared by every
// device using this server. The server records a URL itself whenever a
// playlist loads successfully -- these only read it and delete from it.
// Entries are { url, lastUsed }, most recently used first.
export async function getPlaylistHistory() {
  try {
    const res = await fetch('/api/history');
    const data = await res.json();
    return Array.isArray(data.history) ? data.history : [];
  } catch {
    return []; // history is a convenience -- never block the app on it
  }
}

// Returns the updated history. Also forgets the "last playlist" URL if it is
// the one being removed, so the app doesn't auto-load a playlist the user
// just deleted.
export async function removePlaylistHistory(url) {
  const res = await fetch('/api/history?url=' + encodeURIComponent(url), { method: 'DELETE' });
  const data = await res.json();
  if (data.error) throw new Error(data.error);
  if (getLastPlaylistUrl() === url) saveLastPlaylistUrl('');
  return data.history;
}

// Favorite channels live on the server too (see goserver/favorites.go). A
// favorite is a full copy of the channel object, so favorites still work
// when the playlist they came from isn't loaded. A channel's stream `url` is
// its identity. Each call resolves to the full updated list.
export async function getFavorites() {
  try {
    const res = await fetch('/api/favorites');
    const data = await res.json();
    return Array.isArray(data.favorites) ? data.favorites : [];
  } catch {
    return []; // favorites are a convenience -- never block the app on them
  }
}

export async function addFavorite(channel) {
  const res = await fetch('/api/favorites', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(channel),
  });
  const data = await res.json();
  if (data.error) throw new Error(data.error);
  return data.favorites;
}

export async function removeFavorite(url) {
  const res = await fetch('/api/favorites?url=' + encodeURIComponent(url), { method: 'DELETE' });
  const data = await res.json();
  if (data.error) throw new Error(data.error);
  return data.favorites;
}
