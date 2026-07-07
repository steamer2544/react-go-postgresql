import { createContext, useState, useCallback } from 'react';

export const AuthContext = createContext(null);

function decodeRole(token) {
  try {
    const payload = token.split('.')[1];
    const json = atob(payload);
    const data = JSON.parse(json);
    return data.role || null;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }) {
  const [token, setTokenState] = useState(() => localStorage.getItem('accessToken'));
  const [user, setUserState] = useState(() => {
    const t = localStorage.getItem('accessToken');
    if (!t) return null;
    const role = decodeRole(t);
    return role ? { role } : null;
  });

  const login = useCallback((accessToken) => {
    localStorage.setItem('accessToken', accessToken);
    setTokenState(accessToken);
    const role = decodeRole(accessToken);
    setUserState(role ? { role } : null);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('accessToken');
    setTokenState(null);
    setUserState(null);
  }, []);

  return (
    <AuthContext.Provider
      value={{
        token,
        user,
        isAuthenticated: !!token,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
