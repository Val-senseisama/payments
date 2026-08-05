# Multi-Tenant Payments Platform API

A robust, production-grade Go REST API for multi-tenant payment processing, organization management, and audit tracking.

## 🚀 Features

- **Multi-Tenant Architecture**: Complete tenant isolation for companies, users, and financial records.
- **Authentication & RBAC**: JWT Access Tokens, HttpOnly Refresh Token Cookies with **Redis Whitelisting**, and Role-Based Access Control (`super_admin`, `admin`, `finance`, `viewer`).
- **Audit Logging**: Comprehensive, non-repudiable audit trails for all mutating operations (`company.created`, `company.updated`, `company.deleted`, `user.created`, etc.).
- **Soft Deletion**: Historical financial data and audit records are preserved via `deleted_at` timestamps.
- **High-Performance Routing**: Built with `go-chi/chi/v5` and custom middleware.

---

## 🛠️ Tech Stack

- **Language**: Go (1.22+)
- **Router**: `go-chi/chi/v5`
- **Database**: PostgreSQL
- **Session Cache**: Redis (`go-redis/v9`)
- **Authentication**: JWT (`golang-jwt/jwt/v5`) & `HttpOnly` Cookies
- **Migrations**: `golang-migrate`

---

## 📁 Project Structure

```text
.
├── cmd/
│   ├── api/          # HTTP Server initialization and route mounting
│   ├── config/       # Environment configuration loader
│   ├── db/           # PostgreSQL database connection builder
│   ├── migrations/   # SQL migration files & CLI runner
│   └── main.go       # Application entry point
├── internal/
│   ├── common/       # Shared middleware, auth helpers, and utils
│   │   ├── auth/     # JWT validation, identity context, and RBAC helpers
│   │   └── redis/    # Redis store implementation for refresh token session management
│   └── domain/       # Core business domains
│       ├── account/     # Chart of accounts management & balance tracking
│       ├── audit/       # Background audit log worker and query store
│       ├── company/     # Multi-tenant company management and onboarding
│       ├── ledger/      # Double-entry posting engine & journal entry validation
│       ├── payments/    # PSP adapters, charge execution, and payment attempts
│       ├── profiles/    # Counterparty profiles & invitation token management
│       ├── transactions/# Transaction engine & SSE streaming API
│       └── users/       # User authentication, OTP, and session management
└── types/            # Shared data models, structs, and interfaces
```

---

## ⚡ Getting Started

### Prerequisites

- Go `1.22+`
- PostgreSQL
- Redis

### Environment Setup

Create a `.env` file in the root directory:

```env
PORT=8080
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=payments
DB_PORT=5432
DB_HOST=127.0.0.1
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0
JWT_SECRET=super-secret-access-key
JWT_REFRESH_SECRET=super-secret-refresh-key
JWT_ACCESS_EXPIRATION_SECONDS=900
JWT_REFRESH_EXPIRATION_SECONDS=604800
```

### Running Database Migrations

Apply database migrations:

```bash
make migrate-up
```

To rollback migrations:

```bash
make migrate-down
```

### Building & Running Locally

Run with live-reloading (using `air`):

```bash
air
```

Or build and run using `make`:

```bash
make run
```

---

## 📡 API Reference

### Health Check
- `GET /health` - Service health status

### Authentication & Users (`/api/v1/users`)
- `POST /api/v1/users/register` - Register a new user
- `POST /api/v1/users/request-otp` - Request login OTP
- `POST /api/v1/users/verify-otp` - Verify OTP and receive tokens
- `POST /api/v1/users/refresh` - Refresh access token via HttpOnly cookie

### Companies (`/api/v1/companies`)
- `POST /api/v1/companies` - Create a new company (Auth required)
- `GET /api/v1/companies` - List all companies (SuperAdmin only)
- `GET /api/v1/companies/{id}` - Get company details (Member / Admin / SuperAdmin)
- `PUT /api/v1/companies/{id}` - Update company details (Admin / SuperAdmin)
- `DELETE /api/v1/companies/{id}` - Soft delete company (Admin / SuperAdmin)

### Profiles & Invites (`/api/v1/profiles`)
- `POST /api/v1/profiles` - Create a counterparty profile (customer, vendor, employee)
- `GET /api/v1/profiles` - List company profiles
- `POST /api/v1/companies/invite` - Send company invitation token
- `POST /api/v1/companies/accept-invite` - Accept invitation token

### Accounts (`/api/v1/accounts`)
- `POST /api/v1/accounts` - Create a chart of accounts entry (asset, liability, revenue, expense)
- `GET /api/v1/accounts` - List company accounts
- `GET /api/v1/accounts/{id}` - Get account details

### Transactions (`/api/v1/transactions`)
- `POST /api/v1/transactions` - Create a new transaction (payment_in, payment_out, transfer)
- `GET /api/v1/transactions` - List company transactions
- `GET /api/v1/transactions/stream` - SSE endpoint for live transaction updates

### Ledger (`/api/v1/ledger`)
- `POST /api/v1/ledger/post` - Post balanced journal entries for a completed transaction
- `GET /api/v1/ledger/entries/{txn_id}` - Get double-entry ledger records for a transaction

### Payment Processing & Attempts (`/api/v1/payments`)
- `POST /api/v1/payments/process` - Execute payment charge through a PSP adapter with Redis distributed locking
- `GET /api/v1/payments/attempts/{txn_id}` - Retrieve all payment attempts and gateway response payloads for a transaction

