CREATE TABLE approvals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id UUID REFERENCES transactions(id),
  approved_by UUID REFERENCES users(id),
  status TEXT CHECK (status IN ('approved', 'rejected')) NOT NULL,
  created_at TIMESTAMP DEFAULT now()
);
