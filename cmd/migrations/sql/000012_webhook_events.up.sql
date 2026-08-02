CREATE TABLE webhook_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  psp TEXT NOT NULL,
  event_type TEXT NOT NULL,
  external_reference TEXT NOT NULL,
  payload JSONB,
  processed BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT now()
);

CREATE UNIQUE INDEX idx_webhook_idempotent
ON webhook_events(psp, event_type, external_reference)
WHERE processed = TRUE;
