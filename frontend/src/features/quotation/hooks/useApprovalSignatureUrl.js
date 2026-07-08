import { useQuery } from '@tanstack/react-query';
import { getApprovalSignatureUrl } from '@/features/quotation/services/quotationService';

export function useApprovalSignatureUrl(id, hasApprovedSignature) {
  return useQuery({
    queryKey: ['quotation', id, 'approval-signature'],
    queryFn: () => getApprovalSignatureUrl(id),
    enabled: !!hasApprovedSignature,
    retry: false,
    staleTime: Infinity,
  });
}
