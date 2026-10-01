import { useEffect, useMemo, useRef, useState } from 'react';
import { List } from 'react-window';

const ROW_HEIGHT = 44; // px -- keep in sync with .channel-row in index.css

// Value of the group dropdown's "Favorites" entry. Not a real group name, so
// it can't collide with a playlist group called "Favorites".
const FAVORITES = '__favorites__';

function Row({ index, style, channels, selected, onSelect, favoriteUrls, onToggleFavorite }) {
  const c = channels[index];
  const isFavorite = favoriteUrls.has(c.url);
  // Compared by stream URL + name rather than object identity: a channel
  // shown from the favorites list is a different object than the same
  // channel in the loaded playlist, and both should highlight as playing.
  const isSelected = !!selected && selected.url === c.url && selected.name === c.name;
  return (
    <div
      style={style}
      className={'channel-row' + (isSelected ? ' selected' : '')}
      onClick={() => onSelect(c)}
    >
      {c.logo
        ? <img className="channel-logo" src={c.logo} alt="" onError={e => { e.target.style.visibility = 'hidden'; }} />
        : <span className="channel-logo channel-logo-placeholder" />}
      <span className="channel-name">{c.name}</span>
      <button
        type="button"
        className={'fav-btn' + (isFavorite ? ' on' : '')}
        aria-pressed={isFavorite}
        aria-label={(isFavorite ? 'Remove from favorites: ' : 'Add to favorites: ') + c.name}
        title={isFavorite ? 'Remove from favorites' : 'Add to favorites'}
        onClick={e => { e.stopPropagation(); onToggleFavorite?.(c); }}
      >
        {isFavorite ? '★' : '☆'}
      </button>
    </div>
  );
}

// Playlists can run into the tens of thousands of channels (real public IPTV
// aggregators do), so this renders only the rows actually visible in the
// scroll viewport (react-window) rather than the whole filtered list at
// once -- otherwise the initial render and every keystroke in the search
// box would mean creating/diffing thousands of DOM nodes.
export function ChannelList({
  channels, selected, onSelect, favorites = [], favoriteUrls: favoriteUrlsProp, onToggleFavorite,
}) {
  const [query, setQuery] = useState('');
  // With no playlist loaded there is nothing under "All" -- start on the
  // favorites instead, so they are usable straight after opening the app.
  const [activeGroup, setActiveGroup] = useState(
    channels.length === 0 && favorites.length > 0 ? FAVORITES : 'All'
  );
  const favoriteUrls = useMemo(
    () => favoriteUrlsProp ?? new Set(favorites.map(c => c.url)),
    [favoriteUrlsProp, favorites]
  );
  const listRef = useRef(null);

  const groups = useMemo(() => {
    const set = new Set(channels.map(c => c.group || 'Other'));
    return ['All', ...Array.from(set).sort()];
  }, [channels]);

  const showingFavorites = activeGroup === FAVORITES;

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const source = showingFavorites ? favorites : channels;
    return source.filter(c => {
      if (!showingFavorites && activeGroup !== 'All' && (c.group || 'Other') !== activeGroup) return false;
      if (q && !c.name.toLowerCase().includes(q)) return false;
      return true;
    });
  }, [channels, favorites, query, activeGroup, showingFavorites]);

  // Switching group/search can jump from a long list to a much shorter one.
  // Without this, a scroll offset left over from the longer list can outrun
  // the new (shorter) content for a frame -- react-window computes which
  // rows to render from a stale scrollTop while the browser has already
  // clamped the actual scroll position, so the wrong rows render at the
  // wrong offsets (symptom: the first row rendered clipped against/behind
  // the header above the list). Forcing back to row 0 on every filter
  // change keeps both in sync.
  useEffect(() => {
    listRef.current?.scrollToRow({ index: 0, align: 'start', behavior: 'auto' });
  }, [query, activeGroup]);

  return (
    <div className="channel-list">
      <div className="channel-list-controls">
        <select value={activeGroup} onChange={e => setActiveGroup(e.target.value)}>
          <option value="All">All</option>
          <option value={FAVORITES}>★ Favorites ({favorites.length})</option>
          {groups.filter(g => g !== 'All').map(g => <option key={g} value={g}>{g}</option>)}
        </select>
        <input
          type="text"
          placeholder="Search channels…"
          value={query}
          onChange={e => setQuery(e.target.value)}
        />
      </div>
      <div className="channel-count">{filtered.length} channel{filtered.length === 1 ? '' : 's'}</div>
      {filtered.length === 0 ? (
        <div className="empty">
          {showingFavorites && favorites.length === 0
            ? 'No favorites yet — click the ☆ next to a channel to save it here.'
            : 'No channels match.'}
        </div>
      ) : (
        <List
          listRef={listRef}
          className="channels"
          style={{ height: '100%' }}
          defaultHeight={480}
          rowCount={filtered.length}
          rowHeight={ROW_HEIGHT}
          rowComponent={Row}
          rowProps={{ channels: filtered, selected, onSelect, favoriteUrls, onToggleFavorite }}
        />
      )}
    </div>
  );
}
