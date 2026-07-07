// Test cases: AC17 (successful login stores token + redirects away from /login)
// and AC18 (401 shows a fixed, safe message compared by error.code, not raw text).
//
// Assumed contract (documented — not yet in the codebase):
//   - "@/constants/apiEndpoints" exports AUTH_LOGIN = "/auth/login"
//   - LoginPage has accessible fields: label "Email" (input), label "Password"
//     (input), and a submit button named "Log in" (or "Sign in")
//   - on success: AuthContext token is set (persisted under localStorage key
//     "accessToken") and the app navigates away from "/login"
//   - on a 401 UNAUTHORIZED error: an element with data-testid="login-error"
//     shows the fixed text "Invalid email or password. Please try again."
//     regardless of the backend's actual error.message (comparison by code only)
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';

import { server } from '@/test/mswServer';
import { AUTH_LOGIN } from '@/constants/apiEndpoints';
import { AuthProvider } from '@/contexts/AuthContext';
import LoginPage from '@/features/auth/pages/LoginPage';

const ACCESS_TOKEN_KEY = 'accessToken';

function renderLoginPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <MemoryRouter initialEntries={['/login']}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/" element={<div>Home Page</div>} />
          </Routes>
        </MemoryRouter>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

async function submitLoginForm(email, password) {
  const user = userEvent.setup();
  await user.type(screen.getByLabelText(/email/i), email);
  await user.type(screen.getByLabelText(/password/i), password);
  await user.click(screen.getByRole('button', { name: /log ?in|sign ?in/i }));
}

it('TC-17 happy: successful login stores the access token and navigates away from /login', async () => {
  // Arrange
  server.use(
    http.post(`${import.meta.env.VITE_API_URL}${AUTH_LOGIN}`, () =>
      HttpResponse.json({ data: { access_token: 'abc.def.ghi' }, message: 'logged in' }),
    ),
  );
  renderLoginPage();

  // Act
  await submitLoginForm('user@example.com', 'Sup3rSecret!');

  // Assert
  await screen.findByText('Home Page');
  expect(window.localStorage.getItem(ACCESS_TOKEN_KEY)).toBe('abc.def.ghi');
});

it('TC-18 error: 401 shows a fixed generic message and never the raw backend wording', async () => {
  // Arrange: backend message deliberately unusual — UI must not depend on its wording.
  server.use(
    http.post(`${import.meta.env.VITE_API_URL}${AUTH_LOGIN}`, () =>
      HttpResponse.json(
        { error: { code: 'UNAUTHORIZED', message: 'internal-detail-that-must-not-leak-to-user' } },
        { status: 401 },
      ),
    ),
  );
  renderLoginPage();

  // Act
  await submitLoginForm('user@example.com', 'wrong-password');

  // Assert
  const errorEl = await screen.findByTestId('login-error');
  expect(errorEl).toHaveTextContent('Invalid email or password. Please try again.');
  expect(errorEl.textContent).not.toContain('internal-detail-that-must-not-leak-to-user');
  expect(document.body.textContent).not.toContain('[object Object]');
  expect(window.localStorage.getItem(ACCESS_TOKEN_KEY)).toBeNull();
});
