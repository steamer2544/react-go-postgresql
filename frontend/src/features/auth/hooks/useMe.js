import { useQuery } from '@tanstack/react-query';
import { getMe } from '@/features/auth/services/authService';

export function useMe() {
  return useQuery({
    queryKey: ['me'],
    queryFn: getMe,
    retry: false,
  });
}
