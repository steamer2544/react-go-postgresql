import { useMutation, useQueryClient } from '@tanstack/react-query';
import { remove } from '@/features/quotation/services/quotationService';

export function useDeleteQuotation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id) => remove(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quotations'] });
    },
    retry: false,
  });
}
