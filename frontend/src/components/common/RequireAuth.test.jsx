// Test cases: AC20 — unauthenticated access to a protected route redirects to /login.
//
// Assumed contract (documented — not yet in the codebase):
//   - `AuthContext` (the raw React context object, not just a default-exported
//     Provider) is exported from "@/contexts/AuthContext" so tests can inject
//     an arbitrary auth state without going through a real login network call.
//   - `RequireAuth` is a children-based guard: `<RequireAuth>{children}</RequireAuth>`
//     renders children when `isAuthenticated` is true, otherwise renders
//     `<Navigate to="/login" replace />`.
import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';

import { AuthContext } from '@/contexts/AuthContext';
import RequireAuth from '@/components/common/RequireAuth';

function renderProtectedRoute(authValue) {
  return render(
    <AuthContext.Provider value={authValue}>
      <MemoryRouter initialEntries={['/protected']}>
        <Routes>
          <Route path="/login" element={<div>Login Page</div>} />
          <Route
            path="/protected"
            element={
              <RequireAuth>
                <div>Secret Content</div>
              </RequireAuth>
            }
          />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

it('TC-20 edge: no token -> redirected to /login and protected content never renders', () => {
  // Arrange + Act
  renderProtectedRoute({
    token: null,
    user: null,
    isAuthenticated: false,
    login: vi.fn(),
    logout: vi.fn(),
  });

  // Assert
  expect(screen.getByText('Login Page')).toBeInTheDocument();
  expect(screen.queryByText('Secret Content')).not.toBeInTheDocument();
});

it('TC-20 happy: has a token -> protected content renders', () => {
  // Arrange + Act
  renderProtectedRoute({
    token: 'abc.def.ghi',
    user: { role: 'creator' },
    isAuthenticated: true,
    login: vi.fn(),
    logout: vi.fn(),
  });

  // Assert
  expect(screen.getByText('Secret Content')).toBeInTheDocument();
});
