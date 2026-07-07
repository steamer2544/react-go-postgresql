// Test cases: TC-FE-FORM-PT-01, TC-FE-FORM-PT-02, TC-FE-FORM-PT-03
//
// Assumed contract (documented - not yet in the codebase):
//   - "@/features/quotation/pages/QuotationFormPage" renders payment terms:
//     * Each term row wrapped in data-testid="payment-term-row"
//     * Description input: id={`payment-term-description-${index}`}
//     * Amount input (type number): id={`payment-term-amount-${index}`}
//     * Add Term button with accessible name matching /add term|เพิ่มงวด/i
//     * data-testid="payment-terms-sum" showing total formatted with toLocaleString 2 decimals
//     * data-testid="payment-terms-warning" when sum !== total (and at least 1 term)
//     * All inputs/buttons in this section disabled={isLocked} (non-draft)
//   - Uses renderQuotationFormPage helper and MSW server from '@/test/mswServer'

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

// TC-FE-FORM-PT-01: create mode - fill 2 items (total=2675.00) + 3 matching terms => no warning, save enabled
it('TC-FE-FORM-PT-01: create mode fills 2 items + 3 matching payment terms, no warning, save enabled', async () => {
  // Arrange: no API calls expected in create mode
  renderQuotationFormPage('/quotations/new');

  const user = userEvent.setup();

  // Fill row 1: unit_price=1000, qty=2 => line_total=2000
  await user.type(screen.getAllByLabelText(/unit price/i)[0], '1000');
  await user.type(screen.getAllByLabelText(/qty|quantity/i)[0], '2');

  // Click "Add Item" to add row 2
  await user.click(screen.getByRole('button', { name: /add item|เพิ่มรายการ/i }));

  // Fill row 2: unit_price=500, qty=1 => line_total=500
  await user.type(screen.getAllByLabelText(/unit price/i)[1], '500');
  await user.type(screen.getAllByLabelText(/qty|quantity/i)[1], '1');

  // Add 3 payment term rows
  await user.click(screen.getByRole('button', { name: /add term|เพิ่มงวด/i }));
  await user.click(screen.getByRole('button', { name: /add term|เพิ่มงวด/i }));
  await user.click(screen.getByRole('button', { name: /add term|เพิ่มงวด/i }));

  // Fill term 0: Deposit, 891.67
  // NOTE: uses "term description"/"term amount" label text (NOT "description"/"amount" alone)
  // because the item rows above already have 2 inputs labelled "Description" — a bare
  // /description/i or /amount/i query would match multiple elements and throw.
  await user.type(screen.getAllByLabelText(/term description/i)[0], 'Deposit');
  await user.clear(screen.getAllByLabelText(/term amount/i)[0]);
  await user.type(screen.getAllByLabelText(/term amount/i)[0], '891.67');

  // Fill term 1: Progress, 891.67
  await user.type(screen.getAllByLabelText(/term description/i)[1], 'Progress');
  await user.clear(screen.getAllByLabelText(/term amount/i)[1]);
  await user.type(screen.getAllByLabelText(/term amount/i)[1], '891.67');

  // Fill term 2: Final, 891.66
  await user.type(screen.getAllByLabelText(/term description/i)[2], 'Final');
  await user.clear(screen.getAllByLabelText(/term amount/i)[2]);
  await user.type(screen.getAllByLabelText(/term amount/i)[2], '891.66');

  // Assert: no payment-terms-warning
  expect(screen.queryByTestId('payment-terms-warning')).toBeNull();

  // Assert: save button is NOT disabled
  const saveBtn = screen.getByRole('button', { name: /save|submit|บันทึก/i });
  expect(saveBtn).not.toBeDisabled();

  // Assert: payment-terms-sum shows "2,675.00"
  expect(await screen.findByTestId('payment-terms-sum')).toHaveTextContent('2,675.00');
});

// TC-FE-FORM-PT-02: create mode - 2 items (total=2675.00) + 2 terms summing to 2000 => warning + save disabled
it('TC-FE-FORM-PT-02: mismatch payment terms show warning and disable save', async () => {
  // Arrange: no API calls expected in create mode
  renderQuotationFormPage('/quotations/new');

  const user = userEvent.setup();

  // Fill row 1: unit_price=1000, qty=2 => line_total=2000
  await user.type(screen.getAllByLabelText(/unit price/i)[0], '1000');
  await user.type(screen.getAllByLabelText(/qty|quantity/i)[0], '2');

  // Click "Add Item" to add row 2
  await user.click(screen.getByRole('button', { name: /add item|เพิ่มรายการ/i }));

  // Fill row 2: unit_price=500, qty=1 => line_total=500
  await user.type(screen.getAllByLabelText(/unit price/i)[1], '500');
  await user.type(screen.getAllByLabelText(/qty|quantity/i)[1], '1');

  // Add 2 payment term rows
  await user.click(screen.getByRole('button', { name: /add term|เพิ่มงวด/i }));
  await user.click(screen.getByRole('button', { name: /add term|เพิ่มงวด/i }));

  // Fill term 0: 1000
  // NOTE: "term amount" label text (not bare /amount/i) to avoid ambiguity — see TC-FE-FORM-PT-01.
  await user.clear(screen.getAllByLabelText(/term amount/i)[0]);
  await user.type(screen.getAllByLabelText(/term amount/i)[0], '1000');

  // Fill term 1: 1000 (total=2000 ≠ 2675)
  await user.clear(screen.getAllByLabelText(/term amount/i)[1]);
  await user.type(screen.getAllByLabelText(/term amount/i)[1], '1000');

  // Assert: payment-terms-warning appears
  expect(await screen.findByTestId('payment-terms-warning')).toBeInTheDocument();

  // Assert: save button is disabled
  const saveBtn = screen.getByRole('button', { name: /save|submit|บันทึก/i });
  expect(saveBtn).toBeDisabled();
});

// TC-FE-FORM-PT-03: edit mode with status=sent => payment term inputs + add button disabled
it('TC-FE-FORM-PT-03: edit mode with status=sent disables payment term inputs and add button', async () => {
  // Arrange: mock GET /quotations/77 returning a non-draft quotation with 1 payment term
  server.use(
    http.get(`${API_URL}/quotations/77`, () =>
      HttpResponse.json({
        data: {
          id: 77,
          reference_no: 'QT2607077',
          company: 'Acme',
          status: 'sent',
          attention: 'Mr.X',
          email: 'a@a.com',
          date: '2026-07-01',
          valid_until: '2026-07-31',
          discount_amount: 0,
          subtotal: 2500,
          vat_amount: 175,
          total: 2675,
          items: [
            {
              service_type: 'Design',
              description: 'Website design',
              unit_price: 1000,
              qty: 2,
              line_total: 2000,
              sort_order: 1,
            },
            {
              service_type: 'Development',
              description: 'Backend dev',
              unit_price: 500,
              qty: 1,
              line_total: 500,
              sort_order: 2,
            },
          ],
          payment_terms: [
            { id: 1, term_no: 1, description: 'Deposit', amount: 891.67, sort_order: 1 },
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

  renderQuotationFormPage('/quotations/77/edit');

  // Wait for form to load
  await screen.findByDisplayValue('Deposit');

  // Assert: payment term description input is disabled
  const descriptionInput = screen.getByDisplayValue('Deposit');
  expect(descriptionInput).toBeDisabled();

  // Assert: payment term amount input is disabled
  const amountInput = screen.getByDisplayValue('891.67');
  expect(amountInput).toBeDisabled();

  // Assert: Add Term button is disabled
  const addTermBtn = screen.getByRole('button', { name: /add term|เพิ่มงวด/i });
  expect(addTermBtn).toBeDisabled();
});
