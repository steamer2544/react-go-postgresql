import { useQuery } from '@tanstack/react-query';
import { getSignatureUrl } from '@/features/auth/services/authService';

export function useSignatureUrl(hasSignature) {
  return useQuery({
    queryKey: ['me', 'signature'],
    queryFn: getSignatureUrl,
    enabled: !!hasSignature,
    retry: false,
    staleTime: Infinity,
  });
}
