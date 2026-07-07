import { useMutation } from '@tanstack/react-query';
import { login } from '@/features/auth/services/authService';
import { useAuth } from '@/hooks/useAuth';

export function useLogin() {
  const { login: authLogin } = useAuth();

  return useMutation({
    mutationFn: ({ email, password }) => login(email, password),
    onSuccess: (data) => {
      authLogin(data.access_token);
    },
    retry: false,
  });
}
