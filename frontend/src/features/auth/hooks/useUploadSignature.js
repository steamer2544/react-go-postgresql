import { useMutation, useQueryClient } from '@tanstack/react-query';
import { uploadSignature } from '@/features/auth/services/authService';

export function useUploadSignature() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (file) => uploadSignature(file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['me'] });
    },
    retry: false,
  });
}
