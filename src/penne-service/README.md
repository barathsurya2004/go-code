# Penne Service

`penne-service` is the primary REST API backend microservice for the Penne application stack. Built with Go 1.25, it handles user identity, authentication token management, and financial transaction record keeping. The service leverages **Uber FX** for modular dependency injection, **Gorilla Mux** for HTTP routing, and **PostgreSQL** for relational persistence.

---

## Table of Contents

- [Architecture & Directory Structure](#architecture--directory-structure)
- [Core Features](#core-features)
- [API Reference](#api-reference)
- [Authentication & Middleware](#authentication--middleware)
- [Database & Migrations](#database--migrations)
- [Dependency Injection Setup](#dependency-injection-setup)
- [Configuration](#configuration)
- [Getting Started & Local Development](#getting-started--local-development)
- [Testing & Code Coverage](#testing--code-coverage)
- [Docker Support](#docker-support)

---

## Architecture & Directory Structure

The project follows standard Go project layout conventions cleanly separating entrypoints, business interfaces, persistence layers, and HTTP handlers:

```text
src/penne-service/
├── cmd/
│   └── server/
│       ├── handlers/              # HTTP endpoint handlers & handler module
│       │   ├── module.go          # FX module providing HTTP handlers
│       │   ├── module_test.go
│       │   ├── transaction-service.go
│       │   ├── transaction-service_test.go
│       │   ├── user-service.go
│       │   └── user-service_test.go
│       ├── main.go                # Application bootstrapper with Uber FX
│       ├── main_test.go
│       ├── server.go              # Router setup, HTTP server, & AuthMiddleware
│       └── server_test.go
├── internal/
│   ├── core/                      # Domain models & repository interfaces
│   │   ├── models.go
│   │   └── models_test.go
│   └── db/                        # PostgreSQL database implementations
│       ├── db.go                  # Connection initialization & configuration
│       ├── db_test.go
│       ├── module.go              # FX module providing database repositories
│       ├── module_test.go
│       ├── token-repository.go
│       ├── token-repository_test.go
│       ├── transaction-repository.go
│       ├── transaction-repository_test.go
│       ├── user-repository.go
│       └── user-repository_test.go
├── migrations/                    # SQL migration scripts
│   ├── 000001_create_users_table.up.sql
│   ├── 000001_create_users_table.down.sql
│   ├── 000002_create_txn_table.up.sql
│   ├── 000002_create_txn_table.down.sql
│   ├── 000003_create_token_table.up.sql
│   └── 000003_create_token_table.down.sql
├── Dockerfile                     # Multi-stage Docker build file
├── coverage.out                   # Go test coverage report output
├── go.mod                         # Module definition
└── go.sum                         # Dependency checksums
```

---

## Core Features

- **User Management**: User creation with automatic UUID assignment and query by UUID.
- **Automatic Token Provisioning**: Registration of a new user automatically provisions an authentication token and returns its UUID.
- **Transaction Records**: Complete CRUD operations for transaction rows (amount, country ISO, category, bank name, transaction type).
- **Bearer Token Authentication**: Custom HTTP middleware enforcing Bearer token validation for protected routes while bypassing unauthenticated endpoints (`GET /health`, `POST /user`).
- **Automatic Token Activity Tracking**: Accessing endpoints with a token automatically updates `last_used_at` timestamps in PostgreSQL.
- **Lifecycle Integration**: Graceful start and shutdown of HTTP servers and database connection pools using `fx.Lifecycle`.

---

## API Reference

### Unauthenticated Endpoints

| Method | Endpoint | Description | Request Body / Params | Response |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/health` | Health check endpoint | None | `200 OK` (`OK(deployment check)`) |
| `POST` | `/user` | Registers a new user & returns an auth token | JSON: `{"name": "Alice"}` | `201 Created`: `{"user_auth_token": "<token-uuid>"}` |

### Authenticated Endpoints (Requires `Authorization: Bearer <token>`)

| Method | Endpoint | Description | Query Parameters / Body | Response |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/user` | Fetch user details by context token | None (uses context `user_uuid`) | `200 OK`: `User` JSON object |
| `POST` | `/transaction` | Create a transaction | JSON payload (`amount_e5`, `country_iso2`, `category`, `bank_name`, `txn_type`) | `201 Created`: `Transaction` JSON object |
| `GET` | `/transaction` | Fetch transaction by UUID | `?uuid=<txn-uuid>` | `200 OK`: `Transaction` JSON object |
| `GET` | `/transactions` | Fetch all transactions for a user | `?user_uuid=<user-uuid>` | `200 OK`: Array of `Transaction` JSON objects |
| `PUT` | `/transaction` | Update existing transaction | JSON payload with valid `uuid` | `200 OK` |
| `DELETE` | `/transaction` | Delete transaction by UUID | `?uuid=<txn-uuid>` | `200 OK` |

---

## Authentication & Middleware

Authentication is enforced globally via `AuthMiddleware` registered on Gorilla Mux router:

- **Bypass Logic**: `GET /health` and `POST /user` are open endpoints.
- **Token Lookup**: Extracts the token from the `Authorization: Bearer <token>` HTTP header.
- **Validation**: Verifies token existence in `user_tokens` table and checks if `expires_at` is in the future.
- **Context Injection**: Sets `"user_uuid"` into `r.Context()` for downstream handlers to consume.
- **Last Used Update**: Updates `last_used_at` timestamp in the background on successful token lookup.

---

## Database & Migrations

The service uses PostgreSQL for persistence. Schema migrations reside in `./migrations`:

### Tables

1. **`users`** (`000001_create_users_table.up.sql`)
   - `uuid` (UUID, Primary Key)
   - `name` (VARCHAR)
   - `created_at` (TIMESTAMPTZ)
   - `updated_at` (TIMESTAMPTZ)

2. **`transactionrows`** (`000002_create_txn_table.up.sql`)
   - `uuid` (UUID, Primary Key)
   - `amount_e5` (DOUBLE PRECISION)
   - `user_uuid` (VARCHAR, Indexed)
   - `country_iso2` (VARCHAR)
   - `category` (VARCHAR)
   - `bank_name` (VARCHAR)
   - `txn_type` (VARCHAR)
   - `created_at` (TIMESTAMPTZ)
   - `updated_at` (TIMESTAMPTZ)

3. **`user_tokens`** (`000003_create_token_table.up.sql`)
   - `user_id` (VARCHAR, Foreign Key to `users(uuid)`)
   - `token_uuid` (VARCHAR, Primary Key)
   - `prefix` (VARCHAR)
   - `name` (VARCHAR)
   - `scopes` (TEXT[])
   - `expires_at` (TIMESTAMPTZ)
   - `last_used_at` (TIMESTAMPTZ)
   - `created_at` (TIMESTAMPTZ)
   - `updated_at` (TIMESTAMPTZ)

---

## Dependency Injection Setup

The application uses **Uber FX** for structured dependency management:

- **`pkg.Module`**: Provides application-wide loggers (`*zap.Logger`).
- **`db.Module`**: Configures database options, connects PostgreSQL pool, and provides implementations for `UserRepository`, `TransactionRepository`, and `TokenRepository`.
- **`handlers.Module`**: Instantiates `UserServiceHandler` and `TransactionServiceHandler`.
- **`cmd/server/main.go`**: Assembles modules, initializes router and HTTP server, and hooks startup/shutdown callbacks into the FX lifecycle.

---

## Configuration

Configuration is automatically loaded from environment variables or a local `.env` file via `godotenv`.

| Key | Default Value | Description |
| :--- | :--- | :--- |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/penne_app?sslmode=disable` | PostgreSQL connection DSN string |

---

## Getting Started & Local Development

### Prerequisites

- **Go**: 1.25 or newer
- **PostgreSQL**: 14 or newer

### Setup & Run

1. Clone repository and navigate to `src/penne-service`:
   ```bash
   cd src/penne-service
   ```

2. Ensure PostgreSQL database `penne_app` is running and execute migrations using your preferred migration tool (e.g., `golang-migrate`):
   ```bash
   migrate -path ./migrations -database "$DATABASE_URL" up
   ```

3. Run the service locally:
   ```bash
   go run ./cmd/server
   ```

The HTTP server will start listening on port `:8080`.

---

## Testing & Code Coverage

Run the complete test suite across all packages with statement coverage reporting:

```bash
# Run tests inside penne-service directory
go test -v -coverprofile=coverage.out ./...

# View statement coverage summary
go test -cover ./...
```

### Coverage Breakdown

- `cmd/server/handlers`: **100.0%**
- `internal/db`: **99.3%**
- `cmd/server`: **95.7%** (100% of application logic, remaining lines are top-level `main()`)
- `internal/core`: **100%** (passed domain unit tests)

---

## Docker Support

Build and run using the multi-stage `Dockerfile`:

```bash
# Build Docker image from repo root
docker build -t penne-service -f src/penne-service/Dockerfile .

# Run container
docker run -p 8080:8080 -e DATABASE_URL="postgres://user:pass@host:5432/penne_app?sslmode=disable" penne-service
```
