import { Routes, Route } from 'react-router-dom';
import HomePage from '@/pages/HomePage';
import RequireAuth from '@/components/common/RequireAuth';
import RequireRole from '@/components/common/RequireRole';
import LoginPage from '@/features/auth/pages/LoginPage';
import ProfilePage from '@/features/auth/pages/ProfilePage';
import QuotationListPage from '@/features/quotation/pages/QuotationListPage';
import QuotationFormPage from '@/features/quotation/pages/QuotationFormPage';
import QuotationDetailPage from '@/features/quotation/pages/QuotationDetailPage';

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
      <Route
        path="/quotations"
        element={
          <RequireAuth>
            <QuotationListPage />
          </RequireAuth>
        }
      />
      <Route
        path="/quotations/new"
        element={
          <RequireAuth>
            <RequireRole allowed={['admin', 'creator']}>
              <QuotationFormPage />
            </RequireRole>
          </RequireAuth>
        }
      />
      <Route
        path="/quotations/:id/edit"
        element={
          <RequireAuth>
            <RequireRole allowed={['admin', 'creator']}>
              <QuotationFormPage />
            </RequireRole>
          </RequireAuth>
        }
      />
      <Route
        path="/quotations/:id"
        element={
          <RequireAuth>
            <QuotationDetailPage />
          </RequireAuth>
        }
      />
    </Routes>
  );
}

export default AppRoutes;
