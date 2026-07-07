// Vitest global setup: jest-dom matchers + MSW lifecycle + jsdom polyfills that
// are missing by default (URL.createObjectURL/revokeObjectURL used for the
// signature preview feature). This file is test infrastructure only, referenced
// from vite.config.js `test.setupFiles` — it contains no feature/production code.
import '@testing-library/jest-dom/vitest';
import { afterAll, afterEach, beforeAll, vi } from 'vitest';
import { cleanup } from '@testing-library/react';

import { server } from './mswServer';

// jsdom does not implement createObjectURL/revokeObjectURL.
if (!window.URL.createObjectURL) {
  window.URL.createObjectURL = vi.fn(() => 'blob:mock-object-url');
}
if (!window.URL.revokeObjectURL) {
  window.URL.revokeObjectURL = vi.fn();
}

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => {
  // React Testing Library's automatic cleanup relies on a global `afterEach`
  // being present (Jest-style globals). This project imports `afterEach`
  // explicitly from vitest instead of enabling `test.globals` in
  // vite.config.js, so RTL never detects a global afterEach and its
  // auto-cleanup registration never fires — every test in a file kept
  // rendering into the same document. Call `cleanup()` explicitly so each
  // test starts from an empty DOM (this is test-infra wiring, not
  // feature/test-assertion code).
  cleanup();
  server.resetHandlers();
  window.localStorage.clear();
});
afterAll(() => server.close());
