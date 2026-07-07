// Test cases: AC21 — signature round-trip (fetch existing + upload new).
//
// These tests verify the full client-side round-trip that the original
// ProfilePage.test.jsx scope note explicitly deferred:
//   - GET /me/signature → blob URL → display
//   - POST /me/signature → upload confirmation
import { expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';

import { server } from '@/test/mswServer';
import { ME, ME_PROFILE, ME_SIGNATURE } from '@/constants/apiEndpoints';
import { AuthProvider } from '@/contexts/AuthContext';
import ProfilePage from '@/features/auth/pages/ProfilePage';

/* ------------------------------------------------------------------ */
/* Helpers                                                             */
/* ------------------------------------------------------------------ */

function renderProfilePage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <MemoryRouter initialEntries={['/profile']}>
          <ProfilePage />
        </MemoryRouter>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

/* ------------------------------------------------------------------ */
/* Test 1: existing signature is fetched and displayed via GET /me/signature */
/* ------------------------------------------------------------------ */

it('existing signature is fetched and displayed via GET /me/signature on page load', async () => {
  // Arrange — mock GET /me with a signature_image_path
  server.use(
    http.get(`${import.meta.env.VITE_API_URL}${ME}`, () =>
      HttpResponse.json({
        data: {
          id: 1,
          email: 'u@example.com',
          role: 'creator',
          full_name: 'Somchai',
          position: 'Staff',
          signature_image_path: '/uploads/signatures/user_1.png',
        },
        message: 'ok',
      }),
    ),
    // Mock GET /me/signature — return raw PNG bytes
    http.get(`${import.meta.env.VITE_API_URL}${ME_SIGNATURE}`, () =>
      HttpResponse.arrayBuffer(new Uint8Array([0x89, 0x50, 0x4e, 0x47]).buffer, {
        headers: { 'Content-Type': 'image/png' },
      }),
    ),
  );

  renderProfilePage();

  // The page should show the existing signature image with src starting with 'blob:'
  const img = await screen.findByTestId('current-signature');
  expect(img.getAttribute('src')).toEqual(expect.stringContaining('blob:'));
});

/* ------------------------------------------------------------------ */
/* Test 2: selecting a new signature file and clicking Save uploads it */
/* ------------------------------------------------------------------ */

it('selecting a new signature file and clicking Save uploads it via POST /me/signature', async () => {
  // Arrange
  let profile = {
    id: 1,
    email: 'u@example.com',
    role: 'creator',
    full_name: 'Somchai',
    position: 'Staff',
    signature_image_path: null,
  };
  let uploadCalled = false;

  server.use(
    http.get(`${import.meta.env.VITE_API_URL}${ME}`, () =>
      HttpResponse.json({ data: profile, message: 'ok' }),
    ),
    http.put(`${import.meta.env.VITE_API_URL}${ME_PROFILE}`, async ({ request }) => {
      const body = await request.json();
      profile = { ...profile, full_name: body.full_name, position: body.position };
      return HttpResponse.json({ data: null, message: 'updated' });
    }),
    http.post(`${import.meta.env.VITE_API_URL}${ME_SIGNATURE}`, async () => {
      uploadCalled = true;
      return HttpResponse.json({
        data: { path: '/uploads/signatures/user_1.png' },
        message: 'uploaded',
      });
    }),
  );

  renderProfilePage();
  await screen.findByDisplayValue('Somchai');

  // Act — select a file and click Save
  const file = new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], 'sig.png', {
    type: 'image/png',
  });
  const user = userEvent.setup();
  await user.upload(screen.getByLabelText(/signature/i), file);
  await user.click(screen.getByRole('button', { name: /save/i }));

  // Assert — wait for upload to be confirmed called
  await waitFor(() => {
    expect(uploadCalled).toBe(true);
  });
});
