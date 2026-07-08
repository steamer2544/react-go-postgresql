// Test cases: TC-FE-DETAIL-A01..A09 (AC9 — Submit/Approve/Reject buttons +
// approved section)
//
// Contract required from dev (frontend/src/features/quotation/pages/QuotationDetailPage.jsx):
//   - Uses useMe() (frontend/src/features/auth/hooks/useMe.js, already exists) to
//     read the current { id, role }.
//   - Submit button (accessible name /submit/i) renders only when
//     quotation.status === 'draft' AND (me.role === 'admin' OR me.id === quotation.created_by).
//     Clicking it calls useSubmitQuotation().mutate(id) -> POST /quotations/:id/submit.
//   - Approve button (accessible name /approve/i) and Reject button (accessible
//     name /reject/i) render only when quotation.status === 'pending_approval'
//     AND me.role === 'approver'. Clicking Approve -> POST /quotations/:id/approve;
//     clicking Reject -> POST /quotations/:id/reject.
//   - Mutation errors render as `<p role="alert">{mutation.error.message}</p>`
//     underneath the relevant button.
//   - When quotation.status === 'approved', an "Approved" section renders
//     quotation.approved_signee_name, quotation.approved_signee_position,
//     quotation.approved_at as visible text, and — when
//     quotation.has_approved_signature is true — an <img data-testid="approved-signature">
//     whose src is a blob: URL fetched from GET /quotations/:id/approval-signature.
//
// New files required from dev:
//   frontend/src/features/quotation/hooks/useSubmitQuotation.js
//   frontend/src/features/quotation/hooks/useApproveQuotation.js
//   frontend/src/features/quotation/hooks/useRejectQuotation.js
//   frontend/src/features/quotation/hooks/useApprovalSignatureUrl.js
//   quotationService.js additions: submit(id), approve(id), reject(id), getApprovalSignatureUrl(id)

import { expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';

import { server } from '@/test/mswServer';
import QuotationDetailPage from '@/features/quotation/pages/QuotationDetailPage';

const API_URL = import.meta.env.VITE_API_URL;

function mockMe(me) {
  server.use(http.get(`${API_URL}/me`, () => HttpResponse.json({ data: me, message: 'ok' })));
}

function mockQuotation(quotation) {
  server.use(
    http.get(`${API_URL}/quotations/1`, () =>
      HttpResponse.json({ data: quotation, message: 'ok' }),
    ),
  );
}

function baseQuotation(overrides = {}) {
  return {
    id: 1,
    reference_no: 'QT2607099',
    company: 'Acme Corp',
    status: 'draft',
    attention: 'Mr. John',
    project: 'Project Alpha',
    telephone: '02-123-4567',
    email: 'john@acme.com',
    date: '2026-07-01',
    valid_until: '2026-07-31',
    subtotal: 1000,
    discount_amount: 0,
    vat_amount: 70,
    total: 1070,
    items: [],
    customer_signee_name: null,
    customer_signee_position: null,
    customer_signee_date: null,
    company_signee_name: 'Jane Smith',
    company_signee_position: 'Director',
    created_by: 7,
    approver_id: null,
    approved_at: null,
    approved_signee_name: null,
    approved_signee_position: null,
    has_approved_signature: false,
    ...overrides,
  };
}

function renderDetailPage() {
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
}

// TC-FE-DETAIL-A01: Submit button visible for the creator who owns the draft quotation
it('TC-FE-DETAIL-A01: shows Submit button for the owning creator on a draft quotation', async () => {
  mockMe({ id: 7, email: 'c@x.com', role: 'creator', full_name: 'Creator', position: 'Staff' });
  mockQuotation(baseQuotation({ status: 'draft', created_by: 7 }));

  renderDetailPage();

  expect(await screen.findByRole('button', { name: /submit/i })).toBeInTheDocument();
});

// TC-FE-DETAIL-A02: Submit button hidden for a creator who does NOT own the quotation
it('TC-FE-DETAIL-A02: hides Submit button for a non-owning creator', async () => {
  mockMe({
    id: 8,
    email: 'c2@x.com',
    role: 'creator',
    full_name: 'Other Creator',
    position: 'Staff',
  });
  mockQuotation(baseQuotation({ status: 'draft', created_by: 7 }));

  renderDetailPage();

  await screen.findByText('QT2607099');
  expect(screen.queryByRole('button', { name: /submit/i })).not.toBeInTheDocument();
});

// TC-FE-DETAIL-A03: Submit button visible for admin regardless of ownership
it('TC-FE-DETAIL-A03: shows Submit button for admin even without ownership', async () => {
  mockMe({ id: 1, email: 'a@x.com', role: 'admin', full_name: 'Admin', position: 'Manager' });
  mockQuotation(baseQuotation({ status: 'draft', created_by: 7 }));

  renderDetailPage();

  expect(await screen.findByRole('button', { name: /submit/i })).toBeInTheDocument();
});

// TC-FE-DETAIL-A04: clicking Submit calls POST /quotations/1/submit and the UI reflects the new status
it('TC-FE-DETAIL-A04: clicking Submit sends POST /quotations/1/submit and updates status', async () => {
  mockMe({ id: 7, email: 'c@x.com', role: 'creator', full_name: 'Creator', position: 'Staff' });
  let currentStatus = 'draft';
  server.use(
    http.get(`${API_URL}/quotations/1`, () =>
      HttpResponse.json({
        data: baseQuotation({ status: currentStatus, created_by: 7 }),
        message: 'ok',
      }),
    ),
  );
  let submitCalled = false;
  server.use(
    http.post(`${API_URL}/quotations/1/submit`, () => {
      submitCalled = true;
      currentStatus = 'pending_approval';
      return HttpResponse.json({
        data: baseQuotation({ status: 'pending_approval', created_by: 7 }),
        message: 'submitted',
      });
    }),
  );

  renderDetailPage();
  const user = userEvent.setup();
  const submitBtn = await screen.findByRole('button', { name: /submit/i });

  await user.click(submitBtn);

  await waitFor(() => expect(submitCalled).toBe(true));
  await waitFor(() => expect(screen.getByText('pending_approval')).toBeInTheDocument());
});

// TC-FE-DETAIL-A05: Approve/Reject buttons visible for approver on pending_approval quotation
it('TC-FE-DETAIL-A05: shows Approve and Reject buttons for an approver on a pending_approval quotation', async () => {
  mockMe({ id: 9, email: 'ap@x.com', role: 'approver', full_name: 'Approver', position: 'CFO' });
  mockQuotation(baseQuotation({ status: 'pending_approval', created_by: 7 }));

  renderDetailPage();

  expect(await screen.findByRole('button', { name: /approve/i })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /reject/i })).toBeInTheDocument();
});

// TC-FE-DETAIL-A06: Approve/Reject buttons hidden for a creator (RBAC — approver-only action)
it('TC-FE-DETAIL-A06: hides Approve and Reject buttons for a creator', async () => {
  mockMe({ id: 7, email: 'c@x.com', role: 'creator', full_name: 'Creator', position: 'Staff' });
  mockQuotation(baseQuotation({ status: 'pending_approval', created_by: 7 }));

  renderDetailPage();

  await screen.findByText('QT2607099');
  expect(screen.queryByRole('button', { name: /approve/i })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /reject/i })).not.toBeInTheDocument();
});

// TC-FE-DETAIL-A07: clicking Approve sends POST /quotations/1/approve
it('TC-FE-DETAIL-A07: clicking Approve sends POST /quotations/1/approve', async () => {
  mockMe({ id: 9, email: 'ap@x.com', role: 'approver', full_name: 'Approver', position: 'CFO' });
  mockQuotation(baseQuotation({ status: 'pending_approval', created_by: 7 }));
  let approveCalled = false;
  server.use(
    http.post(`${API_URL}/quotations/1/approve`, () => {
      approveCalled = true;
      return HttpResponse.json({
        data: baseQuotation({
          status: 'approved',
          created_by: 7,
          approver_id: 9,
          approved_at: '2026-07-15T10:00:00Z',
          approved_signee_name: 'Approver',
          approved_signee_position: 'CFO',
          has_approved_signature: true,
        }),
        message: 'approved',
      });
    }),
  );

  renderDetailPage();
  const user = userEvent.setup();
  await user.click(await screen.findByRole('button', { name: /approve/i }));

  await waitFor(() => expect(approveCalled).toBe(true));
});

// TC-FE-DETAIL-A08: error from a failed Approve mutation is shown as an alert
it('TC-FE-DETAIL-A08: shows the mutation error message when Approve fails', async () => {
  mockMe({ id: 9, email: 'ap@x.com', role: 'approver', full_name: 'Approver', position: 'CFO' });
  mockQuotation(baseQuotation({ status: 'pending_approval', created_by: 7 }));
  server.use(
    http.post(`${API_URL}/quotations/1/approve`, () =>
      HttpResponse.json(
        { error: { code: 'VALIDATION_ERROR', message: 'approver has no signature on file' } },
        { status: 400 },
      ),
    ),
  );

  renderDetailPage();
  const user = userEvent.setup();
  await user.click(await screen.findByRole('button', { name: /approve/i }));

  expect(await screen.findByRole('alert')).toHaveTextContent('approver has no signature on file');
});

// TC-FE-DETAIL-A09: approved section shows signee name/position/date and the signature image
it('TC-FE-DETAIL-A09: shows signee name, position, date and signature image when approved', async () => {
  mockMe({ id: 7, email: 'c@x.com', role: 'creator', full_name: 'Creator', position: 'Staff' });
  mockQuotation(
    baseQuotation({
      status: 'approved',
      created_by: 7,
      approver_id: 9,
      approved_at: '2026-07-15T10:00:00Z',
      approved_signee_name: 'Approver Name',
      approved_signee_position: 'CFO',
      has_approved_signature: true,
    }),
  );
  server.use(
    http.get(`${API_URL}/quotations/1/approval-signature`, () =>
      HttpResponse.arrayBuffer(new Uint8Array([0x89, 0x50, 0x4e, 0x47]).buffer, {
        headers: { 'Content-Type': 'image/png' },
      }),
    ),
  );

  renderDetailPage();

  expect(await screen.findByText('Approver Name')).toBeInTheDocument();
  expect(screen.getByText('CFO')).toBeInTheDocument();
  expect(screen.getByText('2026-07-15T10:00:00Z')).toBeInTheDocument();
  const img = await screen.findByTestId('approved-signature');
  expect(img.getAttribute('src')).toEqual(expect.stringContaining('blob:'));
});
