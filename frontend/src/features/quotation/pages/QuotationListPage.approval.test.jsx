// Test cases: TC-FE-LIST-A01 (Decision 1 — status filter label change)
//
// Contract required from dev (frontend/src/features/quotation/pages/QuotationListPage.jsx):
//   STATUS_OPTIONS replaces { value: 'sent', label: 'Sent' } with
//   { value: 'pending_approval', label: 'Pending Approval' }.

import { expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';

import { server } from '@/test/mswServer';
import QuotationListPage from '@/features/quotation/pages/QuotationListPage';

const API_URL = import.meta.env.VITE_API_URL;

function renderList() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/quotations']}>
        <Routes>
          <Route path="/quotations" element={<QuotationListPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

// TC-FE-LIST-A01: status filter option is "Pending Approval" (not "Sent") and
// selecting it requests GET /quotations?...status=pending_approval
it('TC-FE-LIST-A01: status filter option is "Pending Approval" and filters by pending_approval', async () => {
  let lastStatusParam;
  server.use(
    http.get(`${API_URL}/quotations`, ({ request }) => {
      const url = new URL(request.url);
      lastStatusParam = url.searchParams.get('status');
      return HttpResponse.json({ data: [], meta: { page: 1, page_size: 20, total: 0 } });
    }),
  );

  renderList();
  const user = userEvent.setup();

  await screen.findByText('Quotations');

  // Open the react-select dropdown by clicking the currently displayed value ("All").
  await user.click(screen.getByText('All'));

  // Old label must be gone; new label must be present.
  expect(screen.queryByText('Sent')).not.toBeInTheDocument();
  const pendingOption = await screen.findByText('Pending Approval');
  await user.click(pendingOption);

  await waitFor(() => expect(lastStatusParam).toBe('pending_approval'));
});
