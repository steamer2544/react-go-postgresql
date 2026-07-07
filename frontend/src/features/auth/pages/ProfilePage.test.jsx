// Test cases: AC21 — Profile page preview + edit flow.
//
// Scope note (explicit, per "AC ไหนกำกวมจนเขียน assert ไม่ได้ ให้ระบุไว้แทนการเดา"):
// the "signature is retrievable via GET /me/signature and displayed" half of
// AC21 is already covered end-to-end at the backend level (AC10/AC11 in
// docs/tests/user-auth-testcases.md). Mocking the authenticated binary blob
// fetch + object-URL round trip in jsdom adds a lot of incidental complexity
// for little extra confidence, so this file covers only: (a) the pre-submit
// preview requirement and (b) the profile-field edit + refetch requirement.
// If deeper FE coverage of the blob round-trip is wanted later, qa-tester/dev
// should add it explicitly rather than have it guessed here.
//
// Assumed contract (documented — not yet in the codebase):
//   - "@/constants/apiEndpoints" exports ME = "/me" and ME_PROFILE = "/me/profile"
//   - ProfilePage renders form fields labeled "Full name" and "Position"
//     (react-hook-form, pre-filled from GET /me), a file input labeled
//     "Signature", a preview image with data-testid="signature-preview" shown
//     only after a file is chosen (before submit), and a "Save" button that
//     triggers PUT /me/profile then invalidates the "me" query.
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';

import { server } from '@/test/mswServer';
import { ME, ME_PROFILE } from '@/constants/apiEndpoints';
import { AuthProvider } from '@/contexts/AuthContext';
import ProfilePage from '@/features/auth/pages/ProfilePage';

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

it('TC-21 happy: choosing a signature file shows a preview before submitting', async () => {
  // Arrange
  server.use(
    http.get(`${import.meta.env.VITE_API_URL}${ME}`, () =>
      HttpResponse.json({
        data: {
          id: 1,
          email: 'u@example.com',
          role: 'creator',
          full_name: 'Somchai',
          position: 'Staff',
          signature_image_path: null,
        },
        message: 'ok',
      }),
    ),
  );
  renderProfilePage();
  await screen.findByDisplayValue('Somchai');
  const file = new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], 'sig.png', {
    type: 'image/png',
  });

  // Act
  await userEvent.upload(screen.getByLabelText(/signature/i), file);

  // Assert
  const preview = await screen.findByTestId('signature-preview');
  expect(preview.getAttribute('src')).toEqual(expect.stringContaining('blob:'));
});

it('TC-21 happy: editing full name and position then saving shows the updated values after refetch', async () => {
  // Arrange
  let profile = {
    id: 1,
    email: 'u@example.com',
    role: 'creator',
    full_name: 'Somchai',
    position: 'Staff',
    signature_image_path: null,
  };
  server.use(
    http.get(`${import.meta.env.VITE_API_URL}${ME}`, () =>
      HttpResponse.json({ data: profile, message: 'ok' }),
    ),
    http.put(`${import.meta.env.VITE_API_URL}${ME_PROFILE}`, async ({ request }) => {
      const body = await request.json();
      profile = { ...profile, full_name: body.full_name, position: body.position };
      return HttpResponse.json({ data: null, message: 'updated' });
    }),
  );
  renderProfilePage();
  await screen.findByDisplayValue('Somchai');
  const user = userEvent.setup();

  // Act
  const fullNameInput = screen.getByLabelText(/full name/i);
  await user.clear(fullNameInput);
  await user.type(fullNameInput, 'Somchai Updated');
  const positionInput = screen.getByLabelText(/position/i);
  await user.clear(positionInput);
  await user.type(positionInput, 'Senior Staff');
  await user.click(screen.getByRole('button', { name: /save/i }));

  // Assert
  await screen.findByDisplayValue('Somchai Updated');
  expect(screen.getByDisplayValue('Senior Staff')).toBeInTheDocument();
});
