import { useContext } from 'react';
import { Navigate } from 'react-router-dom';
import { AuthContext } from '@/contexts/AuthContext';

function RequireRole({ allowed, children }) {
  const { user } = useContext(AuthContext);
  const role = user?.role;
  if (!allowed.includes(role)) {
    return <Navigate to="/403" replace />;
  }
  return <>{children}</>;
}

export default RequireRole;
