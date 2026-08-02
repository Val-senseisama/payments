1. System Summary (Non-Technical, Precise)

You are building:

A multi-tenant financial operating system for companies to control, move, and account for money across employees, vendors, and customers.

Core Capabilities
Companies own and control money
Users operate on behalf of companies (with roles/permissions)
Profiles represent financial actors (employees, vendors, customers)
Accounts + Entries form a double-entry ledger (source of truth)
Transactions represent intent (payroll, payout, payment, etc.)
Payment Attempts handle real-world execution via PSPs
Approvals enforce organizational control
Ledger posting guarantees financial correctness
Mental Model (Critical)
Layer	Purpose
Transactions	What should happen
Payment Attempts	Try to make it happen
Entries (Ledger)	What actually happened
Key Principle

Money is only “real” when recorded in the ledger (entries), not when a PSP says success.

Good call—**locking to NGN for v1** simplifies a *lot* of risk surfaces (FX, rounding, reconciliation complexity). For your current stage, this is the right constraint.

Let me tighten your schema with that assumption and resolve the gaps you identified.

---

# 🧾 Updated Design Principles (NGN-Only)

* All monetary values are **in kobo (₦ × 100)** → `BIGINT`
* No FX, no currency columns (for now)
* Ledger must **always balance**
* Posting must be **atomic + idempotent**
* Accounts are implicitly NGN

---

# ✅ Key Fixes Applied

### 1. **Remove currency entirely (for now)**

You don’t need it yet—less room for inconsistency.

---

### 2. **Ledger posting = single atomic operation**

We eliminate `ledger_posted` boolean and rely purely on:

```sql
ledger_postings(transaction_id PRIMARY KEY)
```

If it exists → posted
If not → not posted

---

### 3. **Enforce idempotent posting at DB level**

* `entries.transaction_id` indexed
* `ledger_postings.transaction_id` is UNIQUE
* Insert into `ledger_postings` FIRST (lock), then entries

---

### 4. **Fix approval model (deterministic)**

* One row per transaction defining requirement
* Separate table for actual approvals

---

### 5. **Add audit log (non-negotiable)**

---

### 6. **Balance caching = optional but safe**

We’ll include it, but assume:

* updated via trigger OR
* recomputed if needed

---

# 🗄️ FINAL POSTGRES SCHEMA (NGN-ONLY, COMMENTED)

```sql
-- Enable UUID support
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =========================
-- COMPANIES (MULTI-TENANT ROOT)
-- =========================
CREATE TABLE companies (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT now()
);

-- =========================
-- USERS (ADMINS / STAFF)
-- =========================
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  company_id UUID REFERENCES companies(id),
  email TEXT UNIQUE NOT NULL,
  role TEXT CHECK (role IN ('admin', 'finance', 'viewer')) NOT NULL,
  created_at TIMESTAMP DEFAULT now(),
  deleted_at TIMESTAMP
);

-- =========================
-- PROFILES (BUSINESS ENTITIES)
-- customers, vendors, employees
-- =========================
CREATE TABLE profiles (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  company_id UUID REFERENCES companies(id),
  type TEXT CHECK (type IN ('customer', 'vendor', 'employee')) NOT NULL,
  name TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT now(),
  deleted_at TIMESTAMP
);

-- =========================
-- ACCOUNTS (LEDGER ACCOUNTS)
-- Each profile can have multiple accounts
-- =========================
CREATE TABLE accounts (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  company_id UUID REFERENCES companies(id),
  profile_id UUID REFERENCES profiles(id),
  type TEXT CHECK (type IN ('asset', 'liability', 'revenue', 'expense')) NOT NULL,
  name TEXT NOT NULL,

  -- Cached balance in kobo (₦)
  cached_balance BIGINT DEFAULT 0,

  created_at TIMESTAMP DEFAULT now()
);

-- =========================
-- TRANSACTIONS (INTENT LAYER)
-- High-level business action
-- =========================
CREATE TABLE transactions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  company_id UUID REFERENCES companies(id),

  reference TEXT UNIQUE NOT NULL, -- idempotency / external reference

  type TEXT CHECK (type IN ('payment_in', 'payment_out', 'transfer')) NOT NULL,

  amount BIGINT NOT NULL, -- in kobo

  status TEXT CHECK (
    status IN ('pending', 'pending_approval', 'approved', 'processing', 'completed', 'failed')
  ) NOT NULL,

  created_by UUID REFERENCES users(id),

  created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX idx_transactions_company_time
ON transactions(company_id, created_at);

-- =========================
-- LEDGER POSTINGS (SOURCE OF TRUTH FOR POSTED STATE)
-- If a row exists → transaction has been posted
-- =========================
CREATE TABLE ledger_postings (
  transaction_id UUID PRIMARY KEY REFERENCES transactions(id),
  posted_at TIMESTAMP DEFAULT now()
);

-- =========================
-- ENTRIES (DOUBLE-ENTRY LEDGER)
-- Immutable financial truth
-- =========================
CREATE TABLE entries (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  transaction_id UUID REFERENCES transactions(id),
  account_id UUID REFERENCES accounts(id),

  amount BIGINT NOT NULL, -- kobo

  direction TEXT CHECK (direction IN ('debit', 'credit')) NOT NULL,

  created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX idx_entries_account_time
ON entries(account_id, created_at);

CREATE INDEX idx_entries_transaction
ON entries(transaction_id);

-- =========================
-- PAYMENT ATTEMPTS (PSP INTERACTIONS)
-- External payment processing attempts
-- =========================
CREATE TABLE payment_attempts (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  transaction_id UUID REFERENCES transactions(id),

  psp TEXT NOT NULL, -- paystack, flutterwave, etc.

  status TEXT CHECK (
    status IN ('pending', 'success', 'failed')
  ) NOT NULL,

  external_reference TEXT, -- PSP reference

  retry_count INT DEFAULT 0,
  next_retry_at TIMESTAMP,

  response JSONB,

  created_at TIMESTAMP DEFAULT now()
);

-- =========================
-- APPROVAL REQUIREMENTS (ONE PER TRANSACTION)
-- Defines how many approvals are needed
-- =========================
CREATE TABLE approval_requirements (
  transaction_id UUID PRIMARY KEY REFERENCES transactions(id),
  required_count INT NOT NULL
);

-- =========================
-- APPROVALS (WHO APPROVED)
-- =========================
CREATE TABLE approvals (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  transaction_id UUID REFERENCES transactions(id),
  approved_by UUID REFERENCES users(id),

  status TEXT CHECK (status IN ('approved', 'rejected')) NOT NULL,

  created_at TIMESTAMP DEFAULT now()
);

-- =========================
-- APPROVAL RULES (CONFIG)
-- =========================
CREATE TABLE approval_rules (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  company_id UUID REFERENCES companies(id),

  transaction_type TEXT NOT NULL,

  min_amount BIGINT,
  max_amount BIGINT,

  required_approvals INT NOT NULL,

  effective_from TIMESTAMP DEFAULT now(),
  effective_to TIMESTAMP
);

-- =========================
-- WEBHOOK EVENTS (IDEMPOTENT PROCESSING)
-- =========================
CREATE TABLE webhook_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  psp TEXT NOT NULL,
  event_type TEXT NOT NULL,
  external_reference TEXT NOT NULL,

  payload JSONB,

  processed BOOLEAN DEFAULT FALSE,

  created_at TIMESTAMP DEFAULT now()
);

-- Prevent duplicate processing of same PSP event
CREATE UNIQUE INDEX idx_webhook_idempotent
ON webhook_events(psp, event_type, external_reference)
WHERE processed = TRUE;

-- =========================
-- IDEMPOTENCY KEYS (API SAFETY)
-- =========================
CREATE TABLE idempotency_keys (
  key TEXT PRIMARY KEY,
  response JSONB,
  created_at TIMESTAMP DEFAULT now()
);

-- =========================
-- AUDIT LOG (CRITICAL FOR FINTECH)
-- =========================
CREATE TABLE audit_log (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  entity_type TEXT NOT NULL,
  entity_id UUID,

  action TEXT NOT NULL,

  changed_by UUID REFERENCES users(id),

  old_values JSONB,
  new_values JSONB,

  created_at TIMESTAMP DEFAULT now()
);
```

---

# ⚙️ How Posting Should Work (Important)

Your **ledger posting flow must be exactly this**:

```sql
BEGIN;

-- 1. Lock transaction
SELECT * FROM transactions WHERE id = $1 FOR UPDATE;

-- 2. Prevent double posting
INSERT INTO ledger_postings (transaction_id)
VALUES ($1);

-- If this fails → already posted → exit

-- 3. Insert entries (must balance)
INSERT INTO entries (...);
INSERT INTO entries (...);

-- 4. Mark transaction completed
UPDATE transactions SET status = 'completed' WHERE id = $1;

COMMIT;
```

---

# 🧠 Final Assessment

With NGN-only:

✅ No FX bugs
✅ Simpler ledger invariants
✅ Easier reconciliation
✅ Faster iteration

You’ve now got a **serious fintech-grade core**:

* clean separation of intent vs truth
* safe posting model
* retryable PSP layer
* auditable system

---

# Next Step (Recommended)

You’re ready to define:

👉 **Transaction → Ledger mapping rules**

Example:

* `payment_in` → debit cash, credit revenue
* `payment_out` → debit expense, credit cash

That mapping layer is where most systems either become elegant… or chaotic.

Say the word and we’ll design that cleanly.

