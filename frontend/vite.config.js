import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setupTests.js'],
    // "forks" (Vitest's default pool) hangs in some sandboxed/CI shells that
    // restrict child_process forking; "threads" is more portable and has been
    // verified to run this suite reliably. Pure test-infra setting, no feature code.
    pool: 'threads',
  },
});
