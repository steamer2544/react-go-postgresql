CREATE TABLE IF NOT EXISTS payment_terms (
    id BIGSERIAL PRIMARY KEY,
    quotation_id BIGINT NOT NULL REFERENCES quotations(id) ON DELETE CASCADE,
    term_no INTEGER NOT NULL,
    description TEXT,
    amount NUMERIC(12,2) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE (quotation_id, term_no)
);
