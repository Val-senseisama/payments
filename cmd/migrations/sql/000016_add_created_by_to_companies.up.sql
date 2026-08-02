ALTER TABLE companies ADD COLUMN created_by UUID REFERENCES users(id);
