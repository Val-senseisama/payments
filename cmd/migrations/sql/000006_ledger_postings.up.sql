CREATE TABLE ledger_postings (
  transaction_id UUID PRIMARY KEY REFERENCES transactions(id),
  posted_at TIMESTAMP DEFAULT now()
);
