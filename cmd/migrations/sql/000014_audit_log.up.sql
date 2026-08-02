CREATE TABLE IF NOT EXISTS audit_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID REFERENCES companies(id),
  entity_type TEXT NOT NULL,
  entity_id UUID,
  action TEXT NOT NULL,
  changed_by UUID REFERENCES users(id),
  new_values JSONB,
  created_at TIMESTAMP DEFAULT now()
);
