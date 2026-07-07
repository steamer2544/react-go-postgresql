import { useQuery } from '@tanstack/react-query';
import { list } from '@/features/quotation/services/quotationService';

export function useQuotations(params) {
  return useQuery({
    queryKey: ['quotations', params],
    queryFn: () => list(params),
    retry: false,
  });
}
