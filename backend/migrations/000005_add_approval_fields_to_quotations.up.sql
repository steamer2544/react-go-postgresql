-- Backfill safety: no code path ever set 'sent', but guard against stray prod rows
-- before tightening the CHECK constraint.
UPDATE quotations SET status = 'pending_approval' WHERE status = 'sent';

ALTER TABLE quotations DROP CONSTRAINT quotations_status_check;
ALTER TABLE quotations ADD CONSTRAINT quotations_status_check
    CHECK (status IN ('draft', 'pending_approval', 'approved', 'rejected'));

ALTER TABLE quotations ADD COLUMN approver_id BIGINT REFERENCES users(id);
ALTER TABLE quotations ADD COLUMN approved_at TIMESTAMPTZ;
ALTER TABLE quotations ADD COLUMN approved_signee_name TEXT;
ALTER TABLE quotations ADD COLUMN approved_signee_position TEXT;
ALTER TABLE quotations ADD COLUMN approved_signature_path TEXT;
