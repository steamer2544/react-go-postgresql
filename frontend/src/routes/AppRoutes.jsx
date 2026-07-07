import { Routes, Route } from 'react-router-dom';
import HomePage from '@/pages/HomePage';
import RequireAuth from '@/components/common/RequireAuth';
import RequireRole from '@/components/common/RequireRole';
import LoginPage from '@/features/auth/pages/LoginPage';
import ProfilePage from '@/features/auth/pages/ProfilePage';

function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/profile"
        element={
          <RequireAuth>
            <ProfilePage />
          </RequireAuth>
        }
      />
      <Route
        path="/admin"
        element={
          <RequireAuth>
            <RequireRole allowed={['admin']}>
              <div>Admin Panel</div>
            </RequireRole>
          </RequireAuth>
        }
      />
      <Route path="/403" element={<div>403 Forbidden</div>} />
    </Routes>
  );
}

export default AppRoutes;
