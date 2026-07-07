// Test cases: TC-FE-LIST-DELETE-01, TC-FE-LIST-DELETE-02
//
// Verifies the Delete button behavior in QuotationListPage:
//   a) draft quotations show a Delete button; clicking it with confirm=true
//      sends a DELETE request and the quotation disappears after refetch.
//   b) non-draft quotations do NOT show a Delete button.

import { expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';

import { server } from '@/test/mswServer';
import QuotationListPage from '@/features/quotation/pages/QuotationListPage';

const API_URL = import.meta.env.VITE_API_URL;

function renderQuotationList() {
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
  return queryClient;
}

// TC-FE-LIST-DELETE-01: draft quotation shows Delete button; confirm=true => DELETE sent, list updates
it('TC-FE-LIST-DELETE-01: draft quotation shows Delete button and removes after confirm', async () => {
  // Arrange — track whether DELETE was called
  let deleteCalled = false;
  let quotations = [{ id: 1, reference_no: 'QT001', company: 'Acme Co', status: 'draft' }];

  server.use(
    http.get(`${API_URL}/quotations`, () =>
      HttpResponse.json({
        data: quotations,
        meta: { page: 1, page_size: 20, total: quotations.length },
      }),
    ),
    http.delete(`${API_URL}/quotations/:id`, async ({ params }) => {
      deleteCalled = true;
      quotations = quotations.filter((q) => q.id !== parseInt(params.id, 10));
      return HttpResponse.json(
        { data: { id: parseInt(params.id, 10) }, message: 'deleted' },
        { status: 200 },
      );
    }),
  );

  // Mock window.confirm to return true
  const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);

  renderQuotationList();

  const user = userEvent.setup();

  // Wait for the list to load before asserting
  await waitFor(() => {
    expect(screen.getByText('QT001')).toBeInTheDocument();
  });

  // Assert — Delete button is visible for draft quotation
  const deleteBtn = screen.getByRole('button', { name: /delete|ลบ/i });
  expect(deleteBtn).toBeInTheDocument();

  // Act — click Delete
  await user.click(deleteBtn);

  // Assert — mutation was triggered
  expect(deleteCalled).toBe(true);
  expect(confirmSpy).toHaveBeenCalledWith('Delete this quotation?');

  // Assert — after refetch (mock returns empty list), the quotation is gone
  server.use(
    http.get(`${API_URL}/quotations`, () =>
      HttpResponse.json({ data: [], meta: { page: 1, page_size: 20, total: 0 } }),
    ),
  );

  await waitFor(() => {
    expect(screen.queryByText('QT001')).not.toBeInTheDocument();
  });

  confirmSpy.mockRestore();
});

// TC-FE-LIST-DELETE-02: non-draft quotation does NOT show Delete button
it('TC-FE-LIST-DELETE-02: non-draft quotation does not show Delete button', async () => {
  const quotations = [{ id: 2, reference_no: 'QT002', company: 'Beta Ltd', status: 'sent' }];

  server.use(
    http.get(`${API_URL}/quotations`, () =>
      HttpResponse.json({
        data: quotations,
        meta: { page: 1, page_size: 20, total: quotations.length },
      }),
    ),
  );

  renderQuotationList();

  // Wait for the list to load
  await waitFor(() => {
    expect(screen.getByText('QT002')).toBeInTheDocument();
  });

  // Assert — no Delete button for non-draft
  expect(screen.queryByRole('button', { name: /delete|ลบ/i })).not.toBeInTheDocument();
});
