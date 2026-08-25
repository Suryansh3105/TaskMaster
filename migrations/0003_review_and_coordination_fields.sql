ALTER TABLE tasks ADD COLUMN needs_review_at TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN dispatch_attempted_at TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN claim_renewed_at TIMESTAMPTZ;