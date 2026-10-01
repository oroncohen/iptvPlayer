import { useCallback, useEffect, useRef, useState } from 'react';
import {
  loadPlaylist,
  saveLastPlaylistUrl,
  getPlaylistHistory,
  removePlaylistHistory,
} from '../api.js';

export function useChannels() {
  const [channels, setChannels] = useState(null);
  const [fetchedAt, setFetchedAt] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [history, setHistory] = useState([]);
  // The URL whose channels are currently shown -- what Reload refreshes.
  const currentUrl = useRef(null);

  const refreshHistory = useCallback(() => {
    return getPlaylistHistory().then(setHistory);
  }, []);

  useEffect(() => {
    refreshHistory();
  }, [refreshHistory]);

  const load = useCallback((url, { refresh = false } = {}) => {
    setLoading(true);
    setError(null);
    return loadPlaylist(url, { refresh })
      .then(result => {
        currentUrl.current = url;
        setChannels(result.channels);
        setFetchedAt(result.fetchedAt);
        saveLastPlaylistUrl(url);
        refreshHistory(); // the server records the URL itself on success
        return result.channels;
      })
      .catch(e => {
        setError(e.message);
        // A failed *reload* leaves the channels already on screen (and the
        // server's saved copy) untouched; a failed first load has nothing to show.
        if (!refresh) setChannels(null);
        throw e;
      })
      .finally(() => setLoading(false));
  }, [refreshHistory]);

  // Re-downloads the currently shown playlist from the provider.
  const reload = useCallback(() => {
    if (!currentUrl.current) return Promise.resolve();
    return load(currentUrl.current, { refresh: true }).catch(() => {});
  }, [load]);

  const removeFromHistory = useCallback((url) => {
    return removePlaylistHistory(url).then(setHistory).catch(e => setError(e.message));
  }, []);

  return { channels, fetchedAt, loading, error, load, reload, history, removeFromHistory };
}
