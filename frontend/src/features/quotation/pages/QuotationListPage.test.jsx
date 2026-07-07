// Test cases: TC-FE-LIST-01, TC-FE-LIST-02
//
// Assumed contract (documented - not yet in the codebase):
//   - "@/features/quotation/pages/QuotationListPage" renders a list of quotations
//     showing each quotation's reference_no as visible text.
//   - It issues GET /quotations with page/sort/status params via react-query hook
//     useQuotations.
//   - Pagination/sort params follow list-query.md: page, page_size, sort, etc.
//   - QuotationFormPage at "/quotations/new" creates a quotation and navigates
//     back to "/quotations" after successful submit (201), triggering react-query
//     invalidate on the list.
//
// Assumed contract for QuotationFormPage (same as QuotationFormPage.test.jsx):
//   - inputs findable via getByLabelText(/unit price/i) and /qty|quantity/i,
//     getByLabelText(/attention/i), /company/i, /email/i
//   - data-testid="item-row", "discount-input", "summary-*"
//   - submit button matching /save|submit|บันทึก/i

import { expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';

import { server } from '@/test/mswServer';
import QuotationListPage from '@/features/quotation/pages/QuotationListPage';
import QuotationFormPage from '@/features/quotation/pages/QuotationFormPage';

const API_URL = import.meta.env.VITE_API_URL;

// TC-FE-LIST-01 (AC17): create from new-page, navigate back, list refetches and shows new item
it('TC-FE-LIST-01: creates a quotation from the new page and the list refetches to show it', async () => {
  // Arrange — mutable array shared between MSW handlers so that POST appends
  // to the list that GET returns, simulating the invalidate-then-refetch flow.
  let quotations = [];

  server.use(
    http.get(`${API_URL}/quotations`, () =>
      HttpResponse.json({
        data: quotations,
        meta: { page: 1, page_size: 20, total: quotations.length },
      }),
    ),
    http.post(`${API_URL}/quotations`, async ({ request }) => {
      const body = await request.json();
      const created = {
        id: 99,
        reference_no: 'QT2607099',
        company: body.company,
        status: 'draft',
      };
      quotations = [...quotations, created];
      return HttpResponse.json({ data: created, message: 'created' }, { status: 201 });
    }),
  );

  // Shared QueryClient is critical - both ListPage and FormPage must share the
  // same cache so that the mutation's invalidate is visible to the list query.
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/quotations']}>
        <Routes>
          <Route path="/quotations" element={<QuotationListPage />} />
          <Route path="/quotations/new" element={<QuotationFormPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

  const user = userEvent.setup();

  // Act — navigate to the "new" form
  const newLink = screen.getByRole('link', { name: /new|create|เพิ่ม/i });
  await user.click(newLink);

  // The form should now be visible - wait for a form field to confirm navigation
  await screen.findByTestId('item-row');

  // Fill the form with a single item row + required header fields
  await user.type(screen.getAllByLabelText(/unit price/i)[0], '1000');
  await user.type(screen.getAllByLabelText(/qty|quantity/i)[0], '1');
  await user.type(screen.getByLabelText(/attention/i), 'Test Customer');
  await user.type(screen.getByLabelText(/company/i), 'Test Co.');
  await user.type(screen.getByLabelText(/email/i), 'test@test.com');

  // Act — submit
  await user.click(screen.getByRole('button', { name: /save|submit|บันทึก/i }));

  // Assert — after successful submit, the form navigates back to /quotations
  // and the list refetches (react-query invalidate) to show the new quotation.
  expect(await screen.findByText('QT2607099')).toBeInTheDocument();
});

// TC-FE-LIST-02 (list-query.md): smoke test - verify list endpoint receives standard query params
it('TC-FE-LIST-02: list page sends page and page_size query params on mount', async () => {
  // Arrange — capture the URL of the GET /quotations call made by MSW.
  let capturedURL = '';

  server.use(
    http.get(`${API_URL}/quotations`, ({ request }) => {
      capturedURL = request.url;
      return HttpResponse.json({
        data: [],
        meta: { page: 1, page_size: 20, total: 0 },
      });
    }),
  );

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  // Act
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/quotations']}>
        <Routes>
          <Route path="/quotations" element={<QuotationListPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

  // Assert — wait deterministically for the mocked endpoint to have been hit
  // (rather than guessing at loading/empty-state UI text, which the plan does
  // not pin down), then check the standard list-query params are present.
  // This is intentionally a lightweight check - the plan does not pin specific
  // pagination UI controls, so we verify the query params are present without
  // asserting exact values beyond page=1 and page_size being set.
  await waitFor(() => expect(capturedURL).not.toBe(''));
  expect(capturedURL).toContain('page=');
  expect(capturedURL).toContain('page_size=');
});
