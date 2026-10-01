import { describe, it, expect, afterEach, vi } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useFavorites } from '../hooks/useFavorites.js';

const a = { name: 'A', url: 'http://s/a' };
const b = { name: 'B', url: 'http://s/b' };

// Tiny fake of /api/favorites that behaves like the real server.
function stubServer(initial = [], { failWrites = false } = {}) {
  let list = [...initial];
  vi.stubGlobal('fetch', vi.fn((url, opts = {}) => {
    const respond = body => Promise.resolve({ json: () => Promise.resolve(body) });
    if (opts.method === 'POST' || opts.method === 'DELETE') {
      if (failWrites) return Promise.reject(new Error('network down'));
      if (opts.method === 'POST') {
        const c = JSON.parse(opts.body);
        list = [...list.filter(x => x.url !== c.url), c];
      } else {
        const u = decodeURIComponent(url.split('url=')[1]);
        list = list.filter(x => x.url !== u);
      }
    }
    return respond({ favorites: list });
  }));
}

describe('useFavorites', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('loads the favorites from the server', async () => {
    stubServer([a]);
    const { result } = renderHook(() => useFavorites());
    await waitFor(() => expect(result.current.favorites).toEqual([a]));
    expect(result.current.favoriteUrls.has('http://s/a')).toBe(true);
  });

  it('toggling a channel adds it, toggling again removes it', async () => {
    stubServer([]);
    const { result } = renderHook(() => useFavorites());
    await waitFor(() => expect(fetch).toHaveBeenCalled());
    await act(() => result.current.toggleFavorite(b));
    expect(result.current.favorites).toEqual([b]);
    await act(() => result.current.toggleFavorite(b));
    expect(result.current.favorites).toEqual([]);
  });

  it('rolls back to the server state if saving fails', async () => {
    stubServer([a], { failWrites: true });
    const { result } = renderHook(() => useFavorites());
    await waitFor(() => expect(result.current.favorites).toEqual([a]));
    await act(() => result.current.toggleFavorite(b));
    expect(result.current.favorites).toEqual([a]); // the optimistic add of b was undone
  });
});
