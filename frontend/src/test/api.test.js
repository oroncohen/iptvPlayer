import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import {
  resolveStream,
  manifestProxyUrl,
  getLastPlaylistUrl,
  saveLastPlaylistUrl,
  loadPlaylist,
  getPlaylistHistory,
  removePlaylistHistory,
  getFavorites,
  addFavorite,
  removeFavorite,
} from '../api.js';

describe('manifestProxyUrl', () => {
  it('builds a proxy URL for the given HLS url', () => {
    expect(manifestProxyUrl('http://server/stream/index.m3u8'))
      .toBe('/api/proxy/manifest?url=' + encodeURIComponent('http://server/stream/index.m3u8'));
  });
});

describe('resolveStream', () => {
  afterEach(() => vi.unstubAllGlobals());

  function stubFetch(jsonBody) {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ json: () => Promise.resolve(jsonBody) })));
  }

  it('calls /api/resolve with the encoded stream URL', async () => {
    stubFetch({ type: 'direct', url: 'http://server/stream.mp4' });
    await resolveStream('http://server/stream.mp4');
    expect(fetch).toHaveBeenCalledWith('/api/resolve?url=' + encodeURIComponent('http://server/stream.mp4'));
  });

  it('returns the backend\'s classification as-is', async () => {
    stubFetch({ type: 'hls', url: 'http://server/live/user/pass/123.m3u8' });
    const result = await resolveStream('http://server/user/pass/123');
    expect(result).toEqual({ type: 'hls', url: 'http://server/live/user/pass/123.m3u8' });
  });

  it('throws when the backend reports an error', async () => {
    stubFetch({ error: 'missing url param' });
    await expect(resolveStream('')).rejects.toThrow('missing url param');
  });
});

describe('last playlist URL persistence', () => {
  beforeEach(() => localStorage.clear());

  it('returns empty string when nothing saved yet', () => {
    expect(getLastPlaylistUrl()).toBe('');
  });

  it('round-trips a saved URL', () => {
    saveLastPlaylistUrl('http://example.com/playlist.m3u8');
    expect(getLastPlaylistUrl()).toBe('http://example.com/playlist.m3u8');
  });
});

function stubFetch(jsonBody) {
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ json: () => Promise.resolve(jsonBody) })));
}

describe('loadPlaylist', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('requests the saved copy by default and returns channels + fetchedAt', async () => {
    stubFetch({ channels: [{ name: 'A', url: 'http://s/a' }], count: 1, fetchedAt: 1234, cached: true });
    const result = await loadPlaylist('http://p/list.m3u');
    expect(fetch).toHaveBeenCalledWith('/api/playlist?url=' + encodeURIComponent('http://p/list.m3u'));
    expect(result).toEqual({ channels: [{ name: 'A', url: 'http://s/a' }], fetchedAt: 1234, cached: true });
  });

  it('asks the backend to re-download when refresh is set', async () => {
    stubFetch({ channels: [], fetchedAt: 1, cached: false });
    await loadPlaylist('http://p/list.m3u', { refresh: true });
    expect(fetch).toHaveBeenCalledWith('/api/playlist?url=' + encodeURIComponent('http://p/list.m3u') + '&refresh=1');
  });

  it('throws when the backend reports an error', async () => {
    stubFetch({ error: 'playlist server returned HTTP 404' });
    await expect(loadPlaylist('http://p/list.m3u')).rejects.toThrow('playlist server returned HTTP 404');
  });
});

describe('playlist history', () => {
  beforeEach(() => localStorage.clear());
  afterEach(() => vi.unstubAllGlobals());

  it('reads the history from the server', async () => {
    const history = [{ url: 'http://a', lastUsed: 2 }, { url: 'http://b', lastUsed: 1 }];
    stubFetch({ history });
    expect(await getPlaylistHistory()).toEqual(history);
    expect(fetch).toHaveBeenCalledWith('/api/history');
  });

  it('falls back to an empty history if the server call fails', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new Error('network down'))));
    expect(await getPlaylistHistory()).toEqual([]);
  });

  it('deletes an entry on the server and returns the updated history', async () => {
    stubFetch({ history: [{ url: 'http://b', lastUsed: 1 }] });
    const result = await removePlaylistHistory('http://a');
    expect(fetch).toHaveBeenCalledWith('/api/history?url=' + encodeURIComponent('http://a'), { method: 'DELETE' });
    expect(result).toEqual([{ url: 'http://b', lastUsed: 1 }]);
  });

  it('forgets the last-playlist URL when that entry is removed, so it is not auto-loaded again', async () => {
    saveLastPlaylistUrl('http://a');
    stubFetch({ history: [] });
    await removePlaylistHistory('http://a');
    expect(getLastPlaylistUrl()).toBe('');
  });

  it('keeps the last-playlist URL when a different entry is removed', async () => {
    saveLastPlaylistUrl('http://a');
    stubFetch({ history: [] });
    await removePlaylistHistory('http://b');
    expect(getLastPlaylistUrl()).toBe('http://a');
  });
});

describe('favorites api', () => {
  afterEach(() => vi.unstubAllGlobals());
  const ch = { name: 'CNN', group: 'News', url: 'http://s/cnn' };

  it('reads favorites from the server', async () => {
    stubFetch({ favorites: [ch] });
    expect(await getFavorites()).toEqual([ch]);
    expect(fetch).toHaveBeenCalledWith('/api/favorites');
  });

  it('falls back to no favorites if the server call fails', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new Error('down'))));
    expect(await getFavorites()).toEqual([]);
  });

  it('adds a favorite by POSTing the channel as JSON and returns the updated list', async () => {
    stubFetch({ favorites: [ch] });
    expect(await addFavorite(ch)).toEqual([ch]);
    expect(fetch).toHaveBeenCalledWith('/api/favorites', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(ch),
    });
  });

  it('removes a favorite by stream URL and returns the updated list', async () => {
    stubFetch({ favorites: [] });
    expect(await removeFavorite('http://s/cnn')).toEqual([]);
    expect(fetch).toHaveBeenCalledWith('/api/favorites?url=' + encodeURIComponent('http://s/cnn'), { method: 'DELETE' });
  });

  it('throws when the server reports an error', async () => {
    stubFetch({ error: 'channel needs a url' });
    await expect(addFavorite({ name: 'x' })).rejects.toThrow('channel needs a url');
  });
});
