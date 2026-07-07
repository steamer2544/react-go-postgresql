import { useContext } from 'react';
import { Navigate } from 'react-router-dom';
import { AuthContext } from '@/contexts/AuthContext';

function RequireAuth({ children }) {
  const { isAuthenticated } = useContext(AuthContext);
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
}

export default RequireAuth;
