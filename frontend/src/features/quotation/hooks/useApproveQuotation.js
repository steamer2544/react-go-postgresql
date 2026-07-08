import { useMutation, useQueryClient } from '@tanstack/react-query';
import { approve } from '@/features/quotation/services/quotationService';

export function useApproveQuotation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id) => approve(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['quotation', id] });
      queryClient.invalidateQueries({ queryKey: ['quotations'] });
    },
    retry: false,
  });
}
