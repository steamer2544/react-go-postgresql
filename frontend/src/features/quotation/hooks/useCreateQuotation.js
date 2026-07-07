import { useMutation, useQueryClient } from '@tanstack/react-query';
import { create } from '@/features/quotation/services/quotationService';

export function useCreateQuotation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload) => create(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quotations'] });
    },
    retry: false,
  });
}
