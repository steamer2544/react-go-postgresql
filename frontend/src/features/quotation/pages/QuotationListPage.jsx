import {
  useReactTable,
  getCoreRowModel,
  createColumnHelper,
  flexRender,
} from '@tanstack/react-table';
import { useState, useMemo } from 'react';
import Select from 'react-select';
import { useQuotations } from '@/features/quotation/hooks/useQuotations';
import { useDeleteQuotation } from '@/features/quotation/hooks/useDeleteQuotation';
import { Link } from 'react-router-dom';

const columnHelper = createColumnHelper();

const STATUS_OPTIONS = [
  { value: '', label: 'All' },
  { value: 'draft', label: 'Draft' },
  { value: 'pending_approval', label: 'Pending Approval' },
  { value: 'approved', label: 'Approved' },
  { value: 'rejected', label: 'Rejected' },
];

function QuotationListPage() {
  const [statusFilter, setStatusFilter] = useState('');

  const listParams = useMemo(
    () => ({
      page: 1,
      page_size: 20,
      sort: '-created_at',
      ...(statusFilter ? { status: statusFilter } : {}),
    }),
    [statusFilter],
  );

  const { data: quotations } = useQuotations(listParams);

  const deleteMutation = useDeleteQuotation();

  const handleDelete = (id) => {
    if (window.confirm('Delete this quotation?')) {
      deleteMutation.mutate(id);
    }
  };

  const columns = [
    columnHelper.accessor('reference_no', {
      header: 'Reference No',
    }),
    columnHelper.accessor('company', {
      header: 'Company',
    }),
    columnHelper.accessor('status', {
      header: 'Status',
    }),
    columnHelper.display({
      id: 'actions',
      header: '',
      cell: (info) => {
        const row = info.row.original;
        if (row.status === 'draft') {
          return (
            <button aria-label="Delete" onClick={() => handleDelete(row.id)}>
              Delete
            </button>
          );
        }
        return null;
      },
    }),
  ];

  const table = useReactTable({
    data: quotations || [],
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <div>
      <h1>Quotations</h1>
      <Link to="/quotations/new">New Quotation</Link>
      <div style={{ marginBottom: '1rem' }}>
        <label htmlFor="status-filter">Filter by Status: </label>
        <Select
          id="status-filter"
          value={STATUS_OPTIONS.find((o) => o.value === statusFilter) || null}
          onChange={(option) => setStatusFilter(option?.value || '')}
          options={STATUS_OPTIONS}
          placeholder="All"
          isClearable
        />
      </div>
      <table>
        <thead>
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th key={header.id}>
                  {flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id}>
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default QuotationListPage;
