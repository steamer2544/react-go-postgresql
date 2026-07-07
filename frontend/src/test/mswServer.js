// Shared MSW (Mock Service Worker) server instance for frontend tests.
// Each test file registers its own request handlers via `server.use(...)`
// inside the test body (kept independent — testing.md rule: tests must not
// share state / depend on each other). No default handlers are registered
// here so an unmocked request fails loudly (`onUnhandledRequest: "error"`).
import { setupServer } from 'msw/node';

export const server = setupServer();
