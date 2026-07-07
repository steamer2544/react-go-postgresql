// Test cases: TC-FE-FORM-01, TC-FE-FORM-02, TC-FE-FORM-03
//
// Assumed contract (documented - not yet in the codebase):
//   - "@/features/quotation/pages/QuotationFormPage" renders:
//     * Each item row wrapped in data-testid="item-row" containing inputs for
//       unit price (findable via getByLabelText(/unit price/i)) and qty
//       (findable via getByLabelText(/qty|quantity/i)).
//     * An "add item" button with accessible name matching /add item|เพิ่มรายการ/i.
//     * A discount input with data-testid="discount-input".
//     * An error element under discount with data-testid="discount-error".
//     * Summary block: data-testid="summary-subtotal", "summary-discount",
//       "summary-vat", "summary-total" - text formatted as "2,751.50"
//       (toLocaleString with 2 decimals, comma thousands separator).
//     * A submit button with accessible name matching /save|submit|บันทึก/i.
//     * Edit mode: when quotation status !== 'draft', all inputs are disabled
//       and no save/delete buttons appear.
//   - Edit mode loads data via GET /quotations/:id (mocked by MSW).

import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';

import { server } from '@/test/mswServer';
import QuotationFormPage from '@/features/quotation/pages/QuotationFormPage';

const API_URL = import.meta.env.VITE_API_URL;

function renderQuotationFormPage(initialPath) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/quotations/new" element={<QuotationFormPage />} />
          <Route path="/quotations/:id/edit" element={<QuotationFormPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

// TC-FE-FORM-01: create mode - fill 2 rows + discount, assert summary values
it('TC-FE-FORM-01: create mode fills 2 rows + discount and shows correct summary totals', async () => {
  // Arrange: no API calls expected in create mode
  renderQuotationFormPage('/quotations/new');

  const user = userEvent.setup();

  // Fill row 1: unit_price=1000, qty=2
  await user.type(screen.getAllByLabelText(/unit price/i)[0], '1000');
  await user.type(screen.getAllByLabelText(/qty|quantity/i)[0], '2');

  // Click "add item" to add row 2
  await user.click(screen.getByRole('button', { name: /add item|เพิ่มรายการ/i }));

  // Fill row 2: unit_price=250.50, qty=3
  await user.type(screen.getAllByLabelText(/unit price/i)[1], '250.50');
  await user.type(screen.getAllByLabelText(/qty|quantity/i)[1], '3');

  // Fill discount
  const discountInput = screen.getByTestId('discount-input');
  await user.clear(discountInput);
  await user.type(discountInput, '151.50');

  // Assert: summary values appear after debounce/re-render
  expect(await screen.findByTestId('summary-subtotal')).toHaveTextContent('2,751.50');
  expect(await screen.findByTestId('summary-discount')).toHaveTextContent('151.50');
  expect(await screen.findByTestId('summary-vat')).toHaveTextContent('182.00');
  expect(await screen.findByTestId('summary-total')).toHaveTextContent('2,782.00');
});

// TC-FE-FORM-02: discount exceeds subtotal => error + disabled submit
it('TC-FE-FORM-02: discount > subtotal shows discount-error and disables submit button', async () => {
  // Arrange
  renderQuotationFormPage('/quotations/new');

  const user = userEvent.setup();

  // Fill 1 row: unit_price=100, qty=1 => subtotal=100
  await user.type(screen.getAllByLabelText(/unit price/i)[0], '100');
  await user.type(screen.getAllByLabelText(/qty|quantity/i)[0], '1');

  // Enter discount exceeding subtotal
  const discountInput = screen.getByTestId('discount-input');
  await user.clear(discountInput);
  await user.type(discountInput, '500');

  // Assert: discount error appears
  expect(await screen.findByTestId('discount-error')).toBeInTheDocument();

  // Assert: submit button is disabled
  const submitBtn = screen.getByRole('button', { name: /save|submit|บันทึก/i });
  expect(submitBtn).toBeDisabled();
});

// TC-FE-FORM-03: edit mode with non-draft status => fields disabled, no save/delete
it('TC-FE-FORM-03: edit mode with status=sent disables all inputs and hides save/delete buttons', async () => {
  // Arrange: mock GET /quotations/99 returning a non-draft quotation
  server.use(
    http.get(`${API_URL}/quotations/99`, () =>
      HttpResponse.json({
        data: {
          id: 99,
          reference_no: 'QT2607099',
          company: 'Acme',
          status: 'sent',
          attention: 'Mr.X',
          email: 'a@a.com',
          date: '2026-07-01',
          valid_until: '2026-07-31',
          discount_amount: 0,
          subtotal: 100,
          vat_amount: 7,
          total: 107,
          items: [
            {
              service_type: 'A',
              description: 'a',
              unit_price: 100,
              qty: 1,
              line_total: 100,
              sort_order: 1,
            },
          ],
          customer_signee_name: null,
          customer_signee_position: null,
          customer_signee_date: null,
          company_signee_name: 'Somchai',
          company_signee_position: 'Manager',
          created_by: 1,
        },
        message: 'ok',
      }),
    ),
  );

  renderQuotationFormPage('/quotations/99/edit');

  // Wait for form to load
  await screen.findByDisplayValue('Mr.X');

  // Assert: attention input is disabled
  const attentionInput = screen.getByDisplayValue('Mr.X');
  expect(attentionInput).toBeDisabled();

  // Assert: no save button visible
  expect(screen.queryByRole('button', { name: /save|submit|บันทึก/i })).toBeNull();

  // Assert: no delete button visible
  expect(screen.queryByRole('button', { name: /delete|ลบ/i })).toBeNull();
});
