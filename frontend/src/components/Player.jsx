import { useEffect, useRef, useState } from 'react';
import Hls from 'hls.js';
import { resolveStream, manifestProxyUrl } from '../api.js';

export function Player({ channel }) {
  const videoRef = useRef(null);
  const hlsRef = useRef(null);
  const [status, setStatus] = useState('idle'); // 'idle' | 'resolving' | 'playing' | 'error'
  const [errorMessage, setErrorMessage] = useState(null);

  useEffect(() => {
    const video = videoRef.current;
    if (hlsRef.current) {
      hlsRef.current.destroy();
      hlsRef.current = null;
    }
    if (!video || !channel) {
      setStatus('idle');
      return;
    }

    let cancelled = false;
    setStatus('resolving');
    setErrorMessage(null);

    resolveStream(channel.url)
      .then(resolved => {
        if (cancelled) return;

        if (resolved.type === 'unsupported') {
          setStatus('error');
          setErrorMessage(resolved.reason || "This stream isn't playable in a browser.");
          return;
        }

        const src = resolved.type === 'hls' ? manifestProxyUrl(resolved.url) : resolved.url;

        if (resolved.type === 'hls' && !video.canPlayType('application/vnd.apple.mpegurl') && Hls.isSupported()) {
          const hls = new Hls();
          hls.on(Hls.Events.ERROR, (_event, data) => {
            if (data.fatal && !cancelled) {
              setStatus('error');
              setErrorMessage(`Playback error: ${data.details}`);
            }
          });
          hls.loadSource(src);
          hls.attachMedia(video);
          hlsRef.current = hls;
        } else {
          video.src = src;
        }
        setStatus('playing');
        video.play().catch(() => {}); // autoplay can be blocked -- controls let the user hit play
      })
      .catch(e => {
        if (cancelled) return;
        setStatus('error');
        setErrorMessage(e.message);
      });

    return () => {
      cancelled = true;
      if (hlsRef.current) {
        hlsRef.current.destroy();
        hlsRef.current = null;
      }
    };
  }, [channel]);

  if (!channel) {
    return <div className="player player-empty">Select a channel to start watching.</div>;
  }

  return (
    <div className="player">
      <div className="player-title">{channel.name}</div>
      {status === 'resolving' && <div className="player-status">Resolving stream…</div>}
      {status === 'error' && <div className="error-banner">{errorMessage}</div>}
      {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
      <video ref={videoRef} controls autoPlay className="player-video" />
    </div>
  );
}
