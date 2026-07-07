// Test cases: AC20 — role-based route guard.
//
// Assumed contract (documented — not yet in the codebase):
//   - `RequireRole` is a children-based guard: `<RequireRole allowed={["admin"]}>{children}</RequireRole>`
//   - reads the current user's role from `AuthContext` (`user.role`)
//   - role not in `allowed` -> renders `<Navigate to="/403" replace />`
//   - role in `allowed` -> renders children
import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';

import { AuthContext } from '@/contexts/AuthContext';
import RequireRole from '@/components/common/RequireRole';

function renderAdminRoute(role) {
  return render(
    <AuthContext.Provider
      value={{
        token: 'abc.def.ghi',
        user: { role },
        isAuthenticated: true,
        login: vi.fn(),
        logout: vi.fn(),
      }}
    >
      <MemoryRouter initialEntries={['/admin']}>
        <Routes>
          <Route path="/403" element={<div>403 Forbidden</div>} />
          <Route
            path="/admin"
            element={
              <RequireRole allowed={['admin']}>
                <div>Admin Panel</div>
              </RequireRole>
            }
          />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

it('TC-20 error: role not in the allowed list -> redirected to /403, protected content never renders', () => {
  // Arrange + Act
  renderAdminRoute('creator');

  // Assert
  expect(screen.getByText('403 Forbidden')).toBeInTheDocument();
  expect(screen.queryByText('Admin Panel')).not.toBeInTheDocument();
});

it('TC-20 happy: role in the allowed list -> protected content renders', () => {
  // Arrange + Act
  renderAdminRoute('admin');

  // Assert
  expect(screen.getByText('Admin Panel')).toBeInTheDocument();
});
