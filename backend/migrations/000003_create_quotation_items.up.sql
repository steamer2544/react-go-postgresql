CREATE TABLE IF NOT EXISTS quotation_items (
    id BIGSERIAL PRIMARY KEY,
    quotation_id BIGINT NOT NULL REFERENCES quotations(id) ON DELETE CASCADE,
    service_type TEXT,
    description TEXT,
    unit_price NUMERIC(12,2) NOT NULL,
    qty INTEGER NOT NULL,
    line_total NUMERIC(12,2) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);
