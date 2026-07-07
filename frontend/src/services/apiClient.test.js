// Test cases: AC19 (Authorization header attached automatically) and the 401
// handling behaviour described in the plan for services/apiClient.js.
//
// Assumed contract (documented because it is not yet in the codebase):
//   - the access token is persisted under localStorage key "accessToken"
//   - apiClient's request interceptor reads it and sets header
//     `Authorization: Bearer <token>` when present
//   - apiClient's response interceptor clears that key on a 401 response
import { afterEach, describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';

import { server } from '@/test/mswServer';
import apiClient from '@/services/apiClient';

const ACCESS_TOKEN_KEY = 'accessToken';

afterEach(() => {
  window.localStorage.removeItem(ACCESS_TOKEN_KEY);
});

it('TC-19 happy: attaches Authorization: Bearer <token> when a token is stored', async () => {
  // Arrange
  window.localStorage.setItem(ACCESS_TOKEN_KEY, 'abc.def.ghi');
  let capturedAuthHeader = null;
  server.use(
    http.get(`${import.meta.env.VITE_API_URL}/me`, ({ request }) => {
      capturedAuthHeader = request.headers.get('authorization');
      return HttpResponse.json({ data: { id: 1 }, message: 'ok' });
    }),
  );

  // Act
  await apiClient.get('/me');

  // Assert
  expect(capturedAuthHeader).toBe('Bearer abc.def.ghi');
});

it('TC-19 edge: sends no Authorization header when no token is stored', async () => {
  // Arrange
  let capturedAuthHeader = 'not-checked-yet';
  server.use(
    http.get(`${import.meta.env.VITE_API_URL}/me`, ({ request }) => {
      capturedAuthHeader = request.headers.get('authorization');
      return HttpResponse.json({ data: { id: 1 }, message: 'ok' });
    }),
  );

  // Act
  await apiClient.get('/me');

  // Assert
  expect(capturedAuthHeader).toBeNull();
});

it('TC-19 error: clears the stored token when a request comes back 401', async () => {
  // Arrange
  window.localStorage.setItem(ACCESS_TOKEN_KEY, 'expired.token.value');
  server.use(
    http.get(`${import.meta.env.VITE_API_URL}/me`, () =>
      HttpResponse.json(
        { error: { code: 'UNAUTHORIZED', message: 'token expired' } },
        { status: 401 },
      ),
    ),
  );

  // Act
  await expect(apiClient.get('/me')).rejects.toMatchObject({ code: 'UNAUTHORIZED' });

  // Assert
  expect(window.localStorage.getItem(ACCESS_TOKEN_KEY)).toBeNull();
});
