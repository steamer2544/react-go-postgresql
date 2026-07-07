import { useMutation, useQueryClient } from '@tanstack/react-query';
import { updateProfile } from '@/features/auth/services/authService';

export function useUpdateProfile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload) => updateProfile(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['me'] });
    },
    retry: false,
  });
}
