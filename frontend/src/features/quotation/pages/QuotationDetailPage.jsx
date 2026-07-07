import { useParams, Link } from 'react-router-dom';
import { useQuotation } from '@/features/quotation/hooks/useQuotation';

function QuotationDetailPage() {
  const { id } = useParams();
  const { data: quotation, isLoading } = useQuotation(id);

  if (isLoading) return <p>Loading...</p>;
  if (!quotation) return <p>Quotation not found</p>;

  return (
    <div>
      <h1>Quotation Detail</h1>
      <div>
        <p>
          <strong>Reference No:</strong> {quotation.reference_no}
        </p>
        <p>
          <strong>Status:</strong> {quotation.status}
        </p>
        <p>
          <strong>Attention:</strong> {quotation.attention}
        </p>
        <p>
          <strong>Company:</strong> {quotation.company}
        </p>
        <p>
          <strong>Project:</strong> {quotation.project}
        </p>
        <p>
          <strong>Telephone:</strong> {quotation.telephone}
        </p>
        <p>
          <strong>Email:</strong> {quotation.email}
        </p>
        <p>
          <strong>Date:</strong> {quotation.date}
        </p>
        <p>
          <strong>Valid Until:</strong> {quotation.valid_until}
        </p>
      </div>

      {/* Items table */}
      <h2>Items</h2>
      <table>
        <thead>
          <tr>
            <th>Service Type</th>
            <th>Description</th>
            <th>Unit Price</th>
            <th>Qty</th>
            <th>Line Total</th>
          </tr>
        </thead>
        <tbody>
          {(quotation.items || []).map((item, index) => (
            <tr key={index}>
              <td>{item.service_type}</td>
              <td>{item.description}</td>
              <td>
                {Number(item.unit_price).toLocaleString('en-US', {
                  minimumFractionDigits: 2,
                  maximumFractionDigits: 2,
                })}
              </td>
              <td>{item.qty}</td>
              <td>
                {Number(item.line_total).toLocaleString('en-US', {
                  minimumFractionDigits: 2,
                  maximumFractionDigits: 2,
                })}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {/* Summary */}
      <h2>Summary</h2>
      <div data-testid="summary-subtotal">
        {Number(quotation.subtotal).toLocaleString('en-US', {
          minimumFractionDigits: 2,
          maximumFractionDigits: 2,
        })}
      </div>
      <div data-testid="summary-discount">
        {Number(quotation.discount_amount).toLocaleString('en-US', {
          minimumFractionDigits: 2,
          maximumFractionDigits: 2,
        })}
      </div>
      <div data-testid="summary-vat">
        {Number(quotation.vat_amount).toLocaleString('en-US', {
          minimumFractionDigits: 2,
          maximumFractionDigits: 2,
        })}
      </div>
      <div data-testid="summary-total">
        {Number(quotation.total).toLocaleString('en-US', {
          minimumFractionDigits: 2,
          maximumFractionDigits: 2,
        })}
      </div>

      {/* Signees */}
      <h2>Signees</h2>
      <div>
        <p>
          <strong>Customer Signee:</strong> {quotation.customer_signee_name}
        </p>
        <p>
          <strong>Position:</strong> {quotation.customer_signee_position}
        </p>
        <p>
          <strong>Date:</strong> {quotation.customer_signee_date}
        </p>
        <p>
          <strong>Company Signee:</strong> {quotation.company_signee_name}
        </p>
        <p>
          <strong>Position:</strong> {quotation.company_signee_position}
        </p>
      </div>

      {/* Edit link for draft */}
      {quotation.status === 'draft' && <Link to={`/quotations/${id}/edit`}>Edit</Link>}
    </div>
  );
}

export default QuotationDetailPage;
