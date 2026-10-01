import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ChannelList } from '../components/ChannelList.jsx';

const channels = [
  { name: 'BBC News', group: 'News', url: 'http://x/bbc.m3u8', logo: null },
  { name: 'CNN', group: 'News', url: 'http://x/cnn.m3u8', logo: null },
  { name: 'ESPN', group: 'Sports', url: 'http://x/espn.m3u8', logo: null },
  { name: 'No Group Channel', group: null, url: 'http://x/nogroup.m3u8', logo: null },
];

describe('ChannelList', () => {
  it('renders every channel by default', () => {
    render(<ChannelList channels={channels} selected={null} onSelect={() => {}} />);
    expect(screen.getByText('BBC News')).toBeInTheDocument();
    expect(screen.getByText('CNN')).toBeInTheDocument();
    expect(screen.getByText('ESPN')).toBeInTheDocument();
    expect(screen.getByText('4 channels')).toBeInTheDocument();
  });

  it('filters by search query', () => {
    render(<ChannelList channels={channels} selected={null} onSelect={() => {}} />);
    fireEvent.change(screen.getByPlaceholderText('Search channels…'), { target: { value: 'cnn' } });
    expect(screen.getByText('CNN')).toBeInTheDocument();
    expect(screen.queryByText('BBC News')).not.toBeInTheDocument();
  });

  it('filters by group, grouping ungrouped channels under "Other"', () => {
    render(<ChannelList channels={channels} selected={null} onSelect={() => {}} />);
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'Sports' } });
    expect(screen.getByText('ESPN')).toBeInTheDocument();
    expect(screen.queryByText('CNN')).not.toBeInTheDocument();

    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'Other' } });
    expect(screen.getByText('No Group Channel')).toBeInTheDocument();
  });

  it('calls onSelect with the clicked channel', () => {
    const onSelect = vi.fn();
    render(<ChannelList channels={channels} selected={null} onSelect={onSelect} />);
    fireEvent.click(screen.getByText('ESPN'));
    expect(onSelect).toHaveBeenCalledWith(channels[2]);
  });

  it('shows an empty state when nothing matches', () => {
    render(<ChannelList channels={channels} selected={null} onSelect={() => {}} />);
    fireEvent.change(screen.getByPlaceholderText('Search channels…'), { target: { value: 'zzz-nomatch' } });
    expect(screen.getByText('No channels match.')).toBeInTheDocument();
  });

  it('only renders the rows near the visible viewport, not the whole list, for a huge playlist', () => {
    const huge = Array.from({ length: 5000 }, (_, i) => ({
      name: `Channel ${i}`,
      group: 'All Channels',
      url: `http://x/ch${i}.m3u8`,
      logo: null,
    }));
    const { container } = render(<ChannelList channels={huge} selected={null} onSelect={() => {}} />);
    expect(screen.getByText('5000 channels')).toBeInTheDocument();
    const renderedRows = container.querySelectorAll('.channel-row').length;
    expect(renderedRows).toBeGreaterThan(0);
    expect(renderedRows).toBeLessThan(100); // virtualized: nowhere near all 5000 in the DOM
  });

  describe('favorites', () => {
    it('marks favorite channels with a filled star and others with an empty one', () => {
      render(<ChannelList channels={channels} favorites={[channels[1]]} selected={null} onSelect={() => {}} />);
      expect(screen.getByRole('button', { name: 'Remove from favorites: CNN' })).toHaveAttribute('aria-pressed', 'true');
      expect(screen.getByRole('button', { name: 'Add to favorites: BBC News' })).toHaveAttribute('aria-pressed', 'false');
    });

    it('clicking a star toggles that channel without selecting/playing it', () => {
      const onSelect = vi.fn();
      const onToggleFavorite = vi.fn();
      render(<ChannelList channels={channels} favorites={[]} selected={null} onSelect={onSelect}
        onToggleFavorite={onToggleFavorite} />);
      fireEvent.click(screen.getByRole('button', { name: 'Add to favorites: ESPN' }));
      expect(onToggleFavorite).toHaveBeenCalledWith(channels[2]);
      expect(onSelect).not.toHaveBeenCalled();
    });

    it('the Favorites filter lists only the favorites, with a count in the dropdown', () => {
      render(<ChannelList channels={channels} favorites={[channels[0], channels[2]]} selected={null}
        onSelect={() => {}} />);
      expect(screen.getByRole('option', { name: '★ Favorites (2)' })).toBeInTheDocument();
      fireEvent.change(screen.getByRole('combobox'), { target: { value: '__favorites__' } });
      expect(screen.getByText('BBC News')).toBeInTheDocument();
      expect(screen.getByText('ESPN')).toBeInTheDocument();
      expect(screen.queryByText('CNN')).not.toBeInTheDocument();
      expect(screen.getByText('2 channels')).toBeInTheDocument();
    });

    it('search applies inside the favorites view', () => {
      render(<ChannelList channels={channels} favorites={[channels[0], channels[2]]} selected={null}
        onSelect={() => {}} />);
      fireEvent.change(screen.getByRole('combobox'), { target: { value: '__favorites__' } });
      fireEvent.change(screen.getByPlaceholderText('Search channels…'), { target: { value: 'espn' } });
      expect(screen.getByText('ESPN')).toBeInTheDocument();
      expect(screen.queryByText('BBC News')).not.toBeInTheDocument();
    });

    it('shows favorites that are not in the loaded playlist', () => {
      const saved = { name: 'From Another Playlist', group: 'X', url: 'http://elsewhere/1.m3u8' };
      render(<ChannelList channels={channels} favorites={[saved]} selected={null} onSelect={() => {}} />);
      fireEvent.change(screen.getByRole('combobox'), { target: { value: '__favorites__' } });
      expect(screen.getByText('From Another Playlist')).toBeInTheDocument();
    });

    it('shows a hint when there are no favorites yet', () => {
      render(<ChannelList channels={channels} favorites={[]} selected={null} onSelect={() => {}} />);
      fireEvent.change(screen.getByRole('combobox'), { target: { value: '__favorites__' } });
      expect(screen.getByText(/no favorites yet/i)).toBeInTheDocument();
    });

    it('starts on the favorites when no playlist is loaded', () => {
      const saved = { name: 'Saved One', url: 'http://elsewhere/1.m3u8' };
      render(<ChannelList channels={[]} favorites={[saved]} selected={null} onSelect={() => {}} />);
      expect(screen.getByText('Saved One')).toBeInTheDocument();
    });

    it('highlights the playing channel even when it is a different object than the list entry', () => {
      const { container } = render(<ChannelList channels={channels} selected={{ ...channels[1] }} onSelect={() => {}} />);
      const selectedRows = container.querySelectorAll('.channel-row.selected');
      expect(selectedRows).toHaveLength(1);
      expect(selectedRows[0]).toHaveTextContent('CNN');
    });
  });
});

