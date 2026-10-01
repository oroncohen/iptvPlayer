import { useCallback, useEffect, useMemo, useState } from 'react';
import { getFavorites, addFavorite, removeFavorite } from '../api.js';

// The user's starred channels, kept on the server. Toggling updates the UI
// immediately and then adopts the server's list; if the server call fails
// the change is rolled back by re-reading the list.
export function useFavorites() {
  const [favorites, setFavorites] = useState([]);

  useEffect(() => {
    getFavorites().then(setFavorites);
  }, []);

  const favoriteUrls = useMemo(() => new Set(favorites.map(c => c.url)), [favorites]);

  const toggleFavorite = useCallback((channel) => {
    const isFavorite = favoriteUrls.has(channel.url);
    setFavorites(list => isFavorite
      ? list.filter(c => c.url !== channel.url)
      : [...list, channel]);
    const request = isFavorite ? removeFavorite(channel.url) : addFavorite(channel);
    return request
      .then(setFavorites)
      .catch(() => getFavorites().then(setFavorites));
  }, [favoriteUrls]);

  return { favorites, favoriteUrls, toggleFavorite };
}
