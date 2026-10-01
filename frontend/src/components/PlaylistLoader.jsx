import { useState } from 'react';

export function PlaylistLoader({
  initialUrl, loading, error, onLoad, onReload, fetchedAt, history = [], onRemoveHistory,
}) {
  const [url, setUrl] = useState(initialUrl || '');

  function submit(e) {
    e.preventDefault();
    const trimmed = url.trim();
    if (!trimmed) return;
    onLoad(trimmed);
  }

  function loadFromHistory(entryUrl) {
    setUrl(entryUrl);
    onLoad(entryUrl);
  }

  return (
    <form className="playlist-loader" onSubmit={submit}>
      <input
        type="url"
        inputMode="url"
        placeholder="Paste your IPTV playlist URL (.m3u / .m3u8)…"
        value={url}
        onChange={e => setUrl(e.target.value)}
        required
      />
      <button type="submit" className="primary" disabled={loading}>
        {loading ? 'Loading…' : 'Load playlist'}
      </button>
      {onReload && (
        <button
          type="button"
          className="secondary"
          disabled={loading}
          title="Re-download the current playlist from your provider"
          onClick={onReload}
        >
          ↻ Reload
        </button>
      )}
      {onReload && fetchedAt && (
        <div className="playlist-updated">
          Channels last downloaded {new Date(fetchedAt).toLocaleString()}
        </div>
      )}
      {error && <div className="error-banner">{error}</div>}
      {history.length > 0 && (
        <div className="playlist-history">
          <div className="playlist-history-title">Recent playlists</div>
          <ul>
            {history.map(entry => (
              <li key={entry.url} className={entry.url === url.trim() ? 'active' : ''}>
                <button
                  type="button"
                  className="history-load"
                  title={entry.url}
                  disabled={loading}
                  onClick={() => loadFromHistory(entry.url)}
                >
                  {entry.url}
                </button>
                <button
                  type="button"
                  className="history-remove"
                  aria-label={`Remove ${entry.url} from history`}
                  onClick={() => onRemoveHistory?.(entry.url)}
                >
                  ×
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </form>
  );
}
