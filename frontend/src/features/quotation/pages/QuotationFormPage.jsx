import { useEffect, useMemo } from 'react';
import { useForm, Controller } from 'react-hook-form';
import DatePicker from 'react-datepicker';
import dayjs from 'dayjs';
import 'react-datepicker/dist/react-datepicker.css';
import { useParams, useNavigate } from 'react-router-dom';
import { calcTotals } from '@/features/quotation/utils/calcQuotation';
import { useQuotation } from '@/features/quotation/hooks/useQuotation';
import { useCreateQuotation } from '@/features/quotation/hooks/useCreateQuotation';
import { useUpdateQuotation } from '@/features/quotation/hooks/useUpdateQuotation';

function QuotationFormPage() {
  const { id } = useParams();
  const isEditMode = !!id;
  const navigate = useNavigate();

  const { data: quotation, isLoading } = useQuotation(id);
  const createMutation = useCreateQuotation();
  const updateMutation = useUpdateQuotation();

  const { register, handleSubmit, setValue, watch, reset, control } = useForm({
    defaultValues: {
      items: [{ service_type: '', description: '', unit_price: 0, qty: 0, sort_order: 1 }],
      discount_amount: 0,
    },
  });

  // Track form fields via watch
  const watchedItems = watch('items');
  const watchedDiscount = watch('discount_amount');

  // Calculate totals from current form state
  // Transform snake_case form fields to calcTotals expected shape {unitPrice, qty}
  const totals = useMemo(
    () =>
      calcTotals(
        (watchedItems || []).map((item) => ({
          unitPrice: Number(item.unit_price) || 0,
          qty: Number(item.qty) || 0,
        })),
        Number(watchedDiscount) || 0,
      ),
    [watchedItems, watchedDiscount],
  );

  // Load existing data in edit mode
  useEffect(() => {
    if (quotation && isEditMode) {
      reset({
        attention: quotation.attention || '',
        company: quotation.company || '',
        project: quotation.project || '',
        telephone: quotation.telephone || '',
        email: quotation.email || '',
        date: quotation.date || '',
        valid_until: quotation.valid_until || '',
        discount_amount: quotation.discount_amount || 0,
        items:
          quotation.items?.map((item) => ({
            service_type: item.service_type || '',
            description: item.description || '',
            unit_price: item.unit_price || 0,
            qty: item.qty || 0,
            sort_order: item.sort_order || 0,
          })) || [],
        customer_signee_name: quotation.customer_signee_name || '',
        customer_signee_position: quotation.customer_signee_position || '',
        customer_signee_date: quotation.customer_signee_date || '',
        company_signee_name: quotation.company_signee_name || '',
        company_signee_position: quotation.company_signee_position || '',
      });
    }
  }, [quotation, isEditMode, reset]);

  // Check if form is locked (non-draft in edit mode)
  const isLocked = isEditMode && quotation && quotation.status !== 'draft';

  const addItem = () => {
    const items = watch('items', []);
    setValue('items', [
      ...items,
      { service_type: '', description: '', unit_price: 0, qty: 0, sort_order: items.length + 1 },
    ]);
  };

  const onSubmit = (data) => {
    const payload = {
      attention: data.attention,
      company: data.company,
      project: data.project,
      telephone: data.telephone,
      email: data.email,
      date: data.date,
      valid_until: data.valid_until,
      discount_amount: Number(data.discount_amount) || 0,
      items: (data.items || []).map((item, index) => ({
        service_type: item.service_type,
        description: item.description,
        unit_price: Number(item.unit_price) || 0,
        qty: Number(item.qty) || 0,
        sort_order: item.sort_order || index + 1,
      })),
      customer_signee_name: data.customer_signee_name,
      customer_signee_position: data.customer_signee_position,
      customer_signee_date: data.customer_signee_date,
    };

    if (isEditMode) {
      updateMutation.mutate(
        { id, payload },
        {
          onSuccess: () => {
            navigate('/quotations');
          },
        },
      );
    } else {
      createMutation.mutate(payload, {
        onSuccess: () => {
          navigate('/quotations');
        },
      });
    }
  };

  if (isLoading) return <p>Loading...</p>;

  return (
    <div>
      <h1>{isEditMode ? 'Edit Quotation' : 'New Quotation'}</h1>
      <form onSubmit={handleSubmit(onSubmit)}>
        {/* Header fields */}
        <label htmlFor="attention">Attention</label>
        <input id="attention" {...register('attention')} disabled={isLocked} />

        <label htmlFor="company">Company</label>
        <input id="company" {...register('company')} disabled={isLocked} />

        <label htmlFor="project">Project</label>
        <input id="project" {...register('project')} disabled={isLocked} />

        <label htmlFor="telephone">Telephone</label>
        <input id="telephone" {...register('telephone')} disabled={isLocked} />

        <label htmlFor="email">Email</label>
        <input id="email" {...register('email')} disabled={isLocked} />

        <label htmlFor="date">Date</label>
        <Controller
          name="date"
          control={control}
          render={({ field }) => (
            <DatePicker
              id="date"
              selected={field.value ? dayjs(field.value).toDate() : null}
              onChange={(date) => field.onChange(date ? dayjs(date).format('YYYY-MM-DD') : '')}
              disabled={isLocked}
              dateFormat="yyyy-MM-dd"
              placeholderText="Select date"
            />
          )}
        />

        <label htmlFor="validUntil">Valid Until</label>
        <Controller
          name="valid_until"
          control={control}
          render={({ field }) => (
            <DatePicker
              id="validUntil"
              selected={field.value ? dayjs(field.value).toDate() : null}
              onChange={(date) => field.onChange(date ? dayjs(date).format('YYYY-MM-DD') : '')}
              disabled={isLocked}
              dateFormat="yyyy-MM-dd"
              placeholderText="Select date"
            />
          )}
        />

        {/* Items table */}
        <h2>Items</h2>
        {watchedItems.map((item, index) => (
          <div key={index} data-testid="item-row">
            <label htmlFor={`service-type-${index}`}>Service Type</label>
            <input
              id={`service-type-${index}`}
              {...register(`items.${index}.service_type`)}
              disabled={isLocked}
            />

            <label htmlFor={`description-${index}`}>Description</label>
            <input
              id={`description-${index}`}
              {...register(`items.${index}.description`)}
              disabled={isLocked}
            />

            <label htmlFor={`unit-price-${index}`}>Unit Price</label>
            <input
              id={`unit-price-${index}`}
              type="number"
              step="0.01"
              {...register(`items.${index}.unit_price`, { valueAsNumber: true })}
              disabled={isLocked}
            />

            <label htmlFor={`qty-${index}`}>Qty</label>
            <input
              id={`qty-${index}`}
              type="number"
              {...register(`items.${index}.qty`, { valueAsNumber: true })}
              disabled={isLocked}
            />
          </div>
        ))}

        <button type="button" onClick={addItem} disabled={isLocked}>
          Add Item
        </button>

        {/* Discount */}
        <label htmlFor="discount">Discount</label>
        <input
          id="discount"
          data-testid="discount-input"
          type="number"
          step="0.01"
          {...register('discount_amount', { valueAsNumber: true })}
          disabled={isLocked}
        />

        {/* Discount error */}
        {totals.error === 'DISCOUNT_EXCEEDS_SUBTOTAL' && (
          <div data-testid="discount-error">Discount exceeds subtotal</div>
        )}

        {/* Summary block */}
        <h2>Summary</h2>
        <div data-testid="summary-subtotal">
          {totals.subtotal.toLocaleString('en-US', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          })}
        </div>
        <div data-testid="summary-discount">
          {totals.discountAmount.toLocaleString('en-US', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          })}
        </div>
        <div data-testid="summary-vat">
          {totals.vatAmount !== null
            ? totals.vatAmount.toLocaleString('en-US', {
                minimumFractionDigits: 2,
                maximumFractionDigits: 2,
              })
            : '0.00'}
        </div>
        <div data-testid="summary-total">
          {totals.total !== null
            ? totals.total.toLocaleString('en-US', {
                minimumFractionDigits: 2,
                maximumFractionDigits: 2,
              })
            : '0.00'}
        </div>

        {/* Submit */}
        {!isLocked && (
          <button
            type="submit"
            disabled={
              totals.error === 'DISCOUNT_EXCEEDS_SUBTOTAL' ||
              createMutation.isPending ||
              updateMutation.isPending
            }
          >
            Save
          </button>
        )}
      </form>
    </div>
  );
}

export default QuotationFormPage;
