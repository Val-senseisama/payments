CREATE TABLE accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID REFERENCES companies(id),
  profile_id UUID REFERENCES profiles(id),
  type TEXT CHECK (type IN ('asset', 'liability', 'revenue', 'expense')) NOT NULL,
  name TEXT NOT NULL,
  cached_balance BIGINT DEFAULT 0,
  created_at TIMESTAMP DEFAULT now()
);
