CREATE TABLE approval_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID REFERENCES companies(id),
  transaction_type TEXT NOT NULL,
  min_amount BIGINT,
  max_amount BIGINT,
  required_approvals INT NOT NULL,
  effective_from TIMESTAMP DEFAULT now(),
  effective_to TIMESTAMP
);
