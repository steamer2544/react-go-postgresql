// Test cases: TC-FE-DETAIL-01
//
// Smoke test for QuotationDetailPage: mock GET /quotations/1, render via
// MemoryRouter at /quotations/1, assert reference_no and summary-total.

import { expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';

import { server } from '@/test/mswServer';
import QuotationDetailPage from '@/features/quotation/pages/QuotationDetailPage';

const API_URL = import.meta.env.VITE_API_URL;

// TC-FE-DETAIL-01: renders quotation fields and summary-total matches mock data
it('TC-FE-DETAIL-01: renders quotation detail with reference_no and correct summary-total', async () => {
  const mockQuotation = {
    id: 1,
    reference_no: 'QT2607099',
    company: 'Acme Corp',
    status: 'sent',
    attention: 'Mr. John',
    project: 'Project Alpha',
    telephone: '02-123-4567',
    email: 'john@acme.com',
    date: '2026-07-01',
    valid_until: '2026-07-31',
    subtotal: 1000,
    discount_amount: 100,
    vat_amount: 63,
    total: 963,
    items: [
      {
        service_type: 'Service',
        description: 'Consulting',
        unit_price: 1000,
        qty: 1,
        line_total: 1000,
        sort_order: 1,
      },
    ],
    customer_signee_name: 'John Doe',
    customer_signee_position: 'Manager',
    customer_signee_date: '2026-07-01',
    company_signee_name: 'Jane Smith',
    company_signee_position: 'Director',
  };

  server.use(
    http.get(`${API_URL}/quotations/1`, () =>
      HttpResponse.json({ data: mockQuotation, message: 'ok' }),
    ),
  );

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/quotations/1']}>
        <Routes>
          <Route path="/quotations/:id" element={<QuotationDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

  // Assert — reference_no is visible
  await waitFor(() => {
    expect(screen.getByText('QT2607099')).toBeInTheDocument();
  });

  // Assert — summary-total matches mock data (963.00)
  expect(await screen.findByTestId('summary-total')).toHaveTextContent('963.00');
});
