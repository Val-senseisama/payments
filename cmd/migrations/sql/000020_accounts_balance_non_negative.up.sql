ALTER TABLE accounts ADD CONSTRAINT accounts_cached_balance_non_negative CHECK (cached_balance >= 0) NOT VALID;
ALTER TABLE accounts VALIDATE CONSTRAINT accounts_cached_balance_non_negative;
