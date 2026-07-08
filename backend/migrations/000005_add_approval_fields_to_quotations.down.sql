ALTER TABLE quotations DROP COLUMN IF EXISTS approved_signature_path;
ALTER TABLE quotations DROP COLUMN IF EXISTS approved_signee_position;
ALTER TABLE quotations DROP COLUMN IF EXISTS approved_signee_name;
ALTER TABLE quotations DROP COLUMN IF EXISTS approved_at;
ALTER TABLE quotations DROP COLUMN IF EXISTS approver_id;

ALTER TABLE quotations DROP CONSTRAINT quotations_status_check;
ALTER TABLE quotations ADD CONSTRAINT quotations_status_check
    CHECK (status IN ('draft', 'sent', 'approved', 'rejected'));
