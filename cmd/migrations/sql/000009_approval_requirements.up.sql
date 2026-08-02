CREATE TABLE approval_requirements (
  transaction_id UUID PRIMARY KEY REFERENCES transactions(id),
  required_count INT NOT NULL
);
