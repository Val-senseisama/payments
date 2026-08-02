CREATE TABLE transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID REFERENCES companies(id),
  reference TEXT UNIQUE NOT NULL,
  type TEXT CHECK (type IN ('payment_in', 'payment_out', 'transfer')) NOT NULL,
  amount BIGINT NOT NULL,
  status TEXT CHECK (
    status IN ('pending', 'pending_approval', 'approved', 'processing', 'completed', 'failed')
  ) NOT NULL,
  created_by UUID REFERENCES users(id),
  created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX idx_transactions_company_time ON transactions(company_id, created_at);
