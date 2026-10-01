import { useEffect, useState } from 'react';
import { getLastPlaylistUrl } from './api.js';
import { useChannels } from './hooks/useChannels.js';
import { useFavorites } from './hooks/useFavorites.js';
import { PlaylistLoader } from './components/PlaylistLoader.jsx';
import { ChannelList } from './components/ChannelList.jsx';
import { Player } from './components/Player.jsx';

export function Root() {
  const { channels, fetchedAt, loading, error, load, reload, history, removeFromHistory } = useChannels();
  const { favorites, favoriteUrls, toggleFavorite } = useFavorites();
  const [selected, setSelected] = useState(null);
  const [initialUrl] = useState(getLastPlaylistUrl);

  // Auto-load whatever playlist URL worked last time, so returning here
  // doesn't require re-pasting the URL every time.
  useEffect(() => {
    if (initialUrl) load(initialUrl).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="wrap">
      <header>
        <h1>IPTV Player</h1>
        <div className="sub">Paste a playlist URL, pick a channel, watch.</div>
      </header>

      <PlaylistLoader
        initialUrl={initialUrl}
        loading={loading}
        error={error}
        onLoad={load}
        onReload={channels ? reload : undefined}
        fetchedAt={fetchedAt}
        history={history}
        onRemoveHistory={removeFromHistory}
      />

      {(channels || favorites.length > 0) && (
        <div className="app-main">
          <ChannelList
            channels={channels || []}
            selected={selected}
            onSelect={setSelected}
            favorites={favorites}
            favoriteUrls={favoriteUrls}
            onToggleFavorite={toggleFavorite}
          />
          <Player channel={selected} />
        </div>
      )}
    </div>
  );
}
