import { useMutation, useQueryClient } from '@tanstack/react-query';
import { reject } from '@/features/quotation/services/quotationService';

export function useRejectQuotation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id) => reject(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['quotation', id] });
      queryClient.invalidateQueries({ queryKey: ['quotations'] });
    },
    retry: false,
  });
}
