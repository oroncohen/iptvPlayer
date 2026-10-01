import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useChannels } from '../hooks/useChannels.js';

// Minimal fake of the backend: /api/history + /api/playlist.
function stubBackend({ playlists, history = [] }) {
  const calls = [];
  vi.stubGlobal('fetch', vi.fn((url) => {
    calls.push(url);
    const respond = body => Promise.resolve({ json: () => Promise.resolve(body) });
    if (url.startsWith('/api/history')) return respond({ history });
    const handler = playlists.shift();
    return respond(handler);
  }));
  return calls;
}

describe('useChannels', () => {
  beforeEach(() => localStorage.clear());
  afterEach(() => vi.unstubAllGlobals());

  it('loads the history from the server on mount', async () => {
    stubBackend({ playlists: [], history: [{ url: 'http://a', lastUsed: 1 }] });
    const { result } = renderHook(() => useChannels());
    await waitFor(() => expect(result.current.history).toEqual([{ url: 'http://a', lastUsed: 1 }]));
  });

  it('load() shows the channels and exposes when they were downloaded', async () => {
    stubBackend({ playlists: [{ channels: [{ name: 'A', url: 'http://s/a' }], fetchedAt: 42, cached: true }] });
    const { result } = renderHook(() => useChannels());
    await act(() => result.current.load('http://p/list.m3u'));
    expect(result.current.channels).toEqual([{ name: 'A', url: 'http://s/a' }]);
    expect(result.current.fetchedAt).toBe(42);
  });

  it('reload() re-requests the current playlist with refresh=1 and swaps in the new channels', async () => {
    const calls = stubBackend({
      playlists: [
        { channels: [{ name: 'Old', url: 'http://s/old' }], fetchedAt: 1, cached: true },
        { channels: [{ name: 'New', url: 'http://s/new' }], fetchedAt: 2, cached: false },
      ],
    });
    const { result } = renderHook(() => useChannels());
    await act(() => result.current.load('http://p/list.m3u'));
    await act(() => result.current.reload());
    expect(calls.filter(c => c.startsWith('/api/playlist')).at(-1)).toContain('&refresh=1');
    expect(result.current.channels).toEqual([{ name: 'New', url: 'http://s/new' }]);
    expect(result.current.fetchedAt).toBe(2);
  });

  it('a failed reload keeps the channels already on screen and shows the error', async () => {
    stubBackend({
      playlists: [
        { channels: [{ name: 'A', url: 'http://s/a' }], fetchedAt: 1, cached: true },
        { error: 'playlist server returned HTTP 503' },
      ],
    });
    const { result } = renderHook(() => useChannels());
    await act(() => result.current.load('http://p/list.m3u'));
    await act(() => result.current.reload());
    expect(result.current.channels).toEqual([{ name: 'A', url: 'http://s/a' }]);
    expect(result.current.error).toBe('playlist server returned HTTP 503');
  });

  it('a failed first load clears the channels', async () => {
    stubBackend({ playlists: [{ error: 'boom' }] });
    const { result } = renderHook(() => useChannels());
    await act(() => result.current.load('http://p/list.m3u').catch(() => {}));
    expect(result.current.channels).toBeNull();
    expect(result.current.error).toBe('boom');
  });

  it('reload() does nothing before any playlist has been loaded', async () => {
    const calls = stubBackend({ playlists: [] });
    const { result } = renderHook(() => useChannels());
    await act(() => result.current.reload());
    expect(calls.filter(c => c.startsWith('/api/playlist'))).toHaveLength(0);
  });
});
