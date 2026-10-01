import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PlaylistLoader } from '../components/PlaylistLoader.jsx';

describe('PlaylistLoader', () => {
  it('prefills the input from initialUrl', () => {
    render(<PlaylistLoader initialUrl="http://saved/playlist.m3u8" loading={false} error={null} onLoad={() => {}} />);
    expect(screen.getByPlaceholderText(/paste your iptv playlist url/i)).toHaveValue('http://saved/playlist.m3u8');
  });

  it('calls onLoad with the trimmed URL on submit', () => {
    const onLoad = vi.fn();
    render(<PlaylistLoader initialUrl="" loading={false} error={null} onLoad={onLoad} />);
    fireEvent.change(screen.getByPlaceholderText(/paste your iptv playlist url/i), {
      target: { value: '  http://provider/list.m3u8  ' },
    });
    fireEvent.click(screen.getByRole('button', { name: /load playlist/i }));
    expect(onLoad).toHaveBeenCalledWith('http://provider/list.m3u8');
  });

  it('does not call onLoad for an empty/whitespace-only URL', () => {
    const onLoad = vi.fn();
    render(<PlaylistLoader initialUrl="" loading={false} error={null} onLoad={onLoad} />);
    const input = screen.getByPlaceholderText(/paste your iptv playlist url/i);
    // required + type=url blocks native submit on empty, but guard the handler directly too
    fireEvent.change(input, { target: { value: '   ' } });
    fireEvent.submit(input.closest('form'));
    expect(onLoad).not.toHaveBeenCalled();
  });

  it('disables the button and shows loading text while loading', () => {
    render(<PlaylistLoader initialUrl="" loading={true} error={null} onLoad={() => {}} />);
    const button = screen.getByRole('button', { name: /loading/i });
    expect(button).toBeDisabled();
  });

  it('shows the error banner when error is set', () => {
    render(<PlaylistLoader initialUrl="" loading={false} error="playlist server returned HTTP 404" onLoad={() => {}} />);
    expect(screen.getByText('playlist server returned HTTP 404')).toBeInTheDocument();
  });

  describe('history', () => {
    const history = [
      { url: 'http://one.example/get.php?username=u&password=p', lastUsed: 2 },
      { url: 'http://two.example/list.m3u8', lastUsed: 1 },
    ];

    it('renders nothing when there is no history', () => {
      render(<PlaylistLoader initialUrl="" loading={false} error={null} onLoad={() => {}} history={[]} />);
      expect(screen.queryByText(/recent playlists/i)).not.toBeInTheDocument();
    });

    it('lists history entries with their full URL, query string included', () => {
      render(<PlaylistLoader initialUrl="" loading={false} error={null} onLoad={() => {}} history={history} />);
      expect(screen.getByRole('button', { name: history[0].url })).toHaveAttribute('title', history[0].url);
      expect(screen.getByRole('button', { name: history[1].url })).toBeInTheDocument();
    });

    it('clicking an entry fills the input and loads that URL', () => {
      const onLoad = vi.fn();
      render(<PlaylistLoader initialUrl="" loading={false} error={null} onLoad={onLoad} history={history} />);
      fireEvent.click(screen.getByRole('button', { name: 'http://two.example/list.m3u8' }));
      expect(onLoad).toHaveBeenCalledWith('http://two.example/list.m3u8');
      expect(screen.getByPlaceholderText(/paste your iptv playlist url/i)).toHaveValue('http://two.example/list.m3u8');
    });

    it('the × button removes that entry without loading it', () => {
      const onLoad = vi.fn();
      const onRemoveHistory = vi.fn();
      render(
        <PlaylistLoader initialUrl="" loading={false} error={null} onLoad={onLoad}
          history={history} onRemoveHistory={onRemoveHistory} />
      );
      fireEvent.click(screen.getByRole('button', { name: /remove http:\/\/two\.example\/list\.m3u8/i }));
      expect(onRemoveHistory).toHaveBeenCalledWith('http://two.example/list.m3u8');
      expect(onLoad).not.toHaveBeenCalled();
    });
  });

  describe('reload', () => {
    it('shows no Reload button until a playlist is loaded (no onReload)', () => {
      render(<PlaylistLoader initialUrl="" loading={false} error={null} onLoad={() => {}} />);
      expect(screen.queryByRole('button', { name: /reload/i })).not.toBeInTheDocument();
    });

    it('calls onReload when Reload is clicked', () => {
      const onReload = vi.fn();
      render(<PlaylistLoader initialUrl="" loading={false} error={null} onLoad={() => {}} onReload={onReload} />);
      fireEvent.click(screen.getByRole('button', { name: /reload/i }));
      expect(onReload).toHaveBeenCalledTimes(1);
    });

    it('disables Reload while loading', () => {
      render(<PlaylistLoader initialUrl="" loading={true} error={null} onLoad={() => {}} onReload={() => {}} />);
      expect(screen.getByRole('button', { name: /reload/i })).toBeDisabled();
    });

    it('shows when the channels were last downloaded', () => {
      render(<PlaylistLoader initialUrl="" loading={false} error={null} onLoad={() => {}}
        onReload={() => {}} fetchedAt={new Date(2026, 8, 19, 22, 14).getTime()} />);
      expect(screen.getByText(/channels last downloaded/i)).toBeInTheDocument();
    });
  });
});
