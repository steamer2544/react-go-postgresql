import { useQuery } from '@tanstack/react-query';
import { getById } from '@/features/quotation/services/quotationService';

export function useQuotation(id) {
  return useQuery({
    queryKey: ['quotation', id],
    queryFn: () => getById(id),
    enabled: !!id,
    retry: false,
  });
}
