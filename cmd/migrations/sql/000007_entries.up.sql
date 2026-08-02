CREATE TABLE entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id UUID REFERENCES transactions(id),
  account_id UUID REFERENCES accounts(id),
  amount BIGINT NOT NULL,
  direction TEXT CHECK (direction IN ('debit', 'credit')) NOT NULL,
  created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX idx_entries_account_time ON entries(account_id, created_at);
CREATE INDEX idx_entries_transaction ON entries(transaction_id);
