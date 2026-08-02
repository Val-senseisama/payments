CREATE TABLE payment_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id UUID REFERENCES transactions(id),
  psp TEXT NOT NULL,
  status TEXT CHECK (
    status IN ('pending', 'success', 'failed')
  ) NOT NULL,
  external_reference TEXT,
  retry_count INT DEFAULT 0,
  next_retry_at TIMESTAMP,
  response JSONB,
  created_at TIMESTAMP DEFAULT now()
);
