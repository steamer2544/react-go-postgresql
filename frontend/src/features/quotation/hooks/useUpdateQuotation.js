import { useMutation, useQueryClient } from '@tanstack/react-query';
import { update } from '@/features/quotation/services/quotationService';

export function useUpdateQuotation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, payload }) => update(id, payload),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['quotations'] });
      queryClient.invalidateQueries({ queryKey: ['quotation', variables.id] });
    },
    retry: false,
  });
}
