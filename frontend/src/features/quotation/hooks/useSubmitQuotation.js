import { useMutation, useQueryClient } from '@tanstack/react-query';
import { submit } from '@/features/quotation/services/quotationService';

export function useSubmitQuotation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id) => submit(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['quotation', id] });
      queryClient.invalidateQueries({ queryKey: ['quotations'] });
    },
    retry: false,
  });
}
