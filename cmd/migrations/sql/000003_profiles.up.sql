CREATE TABLE profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID REFERENCES companies(id),
  type TEXT CHECK (type IN ('customer', 'vendor', 'employee')) NOT NULL,
  name TEXT NOT NULL,
  email TEXT,
  phone TEXT,
  avatar_url TEXT,
  metadata JSONB,
  created_at TIMESTAMP DEFAULT now(),
  deleted_at TIMESTAMP
);
