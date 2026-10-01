import '@testing-library/jest-dom/vitest';

// jsdom has no layout engine and no ResizeObserver -- react-window (used by
// ChannelList for virtualized scrolling) needs one to exist, even though it
// falls back to its `defaultHeight` prop when it can't get a real
// measurement. A no-op stub is enough: we only need it to exist, not to
// ever actually fire.
if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}
