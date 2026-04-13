# TaskFlow — Backend API

Go REST API for the TaskFlow: JWT auth, PostgreSQL, SQL migrations (golang-migrate), projects and tasks with role-appropriate access rules.

**Role:** Backend-only submission. See [API_CONTRACT.md](API_CONTRACT.md) for request/response examples.

---

## 1. Overview

- **What it does:** Register/login, list/create/update/delete projects (owner rules), list/create/update/delete tasks (filters, pagination), optional per-project stats.
- **Stack:** Go 1.22, chi, sqlx, PostgreSQL 16, golang-migrate, JWT (HS256), bcrypt (cost 12), structured logging with `slog`.
- **Base path:** All API routes are under `/api` (e.g. `POST /api/auth/login`). Health: `GET /health`.

---

## 2. Architecture decisions

- **Layers:** `handler` → `service` → `repository` keeps HTTP thin, business rules testable with mocked repos.
- **SQL:** Hand-written queries with parameter binding (no string-concatenated user input in SQL). Dynamic `WHERE` clauses in task list use fixed fragment templates and bound args.
- **IDs:** UUIDs in DB and API.
- **List responses:** Paginated JSON uses a `data` envelope plus `total`, `page`, `limit`, `total_pages` (documented in [API_CONTRACT.md](API_CONTRACT.md)); this differs from the frontend mock’s `projects` / `tasks` top-level keys—clients should use this contract.
- **Intentional tradeoffs:**
  - `GET /api/projects/{id}`, `GET .../stats`, and task listing do not re-check “membership” beyond “project exists”; any authenticated user with a UUID could read that project. Tightening this would add `user_id` checks against owner/assignee (or a membership table).
  - `PATCH /api/tasks/{id}` allows any authenticated user to update a task (handler documents this). Delete remains restricted to project owner or task creator.
  - Migration failures are logged and the process exits so the API does not serve traffic against a partially migrated database.

---

## 3. Running locally

Requires **Docker** and **Docker Compose** only.

```bash
git clone https://github.com/<your-username>/taskflow.git
cd taskflow
cp .env.example .env
# Optional: edit .env for JWT_SECRET and database settings
docker compose up --build
```

- **API:** http://localhost:8080
- **Health:** http://localhost:8080/health

There is no SPA in this repo; use curl, [BACKEND_TESTING.md](BACKEND_TESTING.md), or a REST client against port **8080**.

---

## 4. Running migrations

Migrations run **automatically** when the API container starts (`internal/database/migrate.go` via `cmd/server/main.go`). No separate migrate command is required for Docker.

To run the binary locally against your own Postgres, ensure `DATABASE_URL` and `JWT_SECRET` are set; migrations still run on startup.

---

## 5. Test credentials (seed data)

Seed is applied by migration `000004_seed_data` (see `migrations/`).

```
Email:    test@example.com
Password: password123
```

Additional seeded users (same password): `priya@example.com`, `amit@example.com`.

---

## 6. API reference

- **[API_CONTRACT.md](API_CONTRACT.md)** — endpoints, statuses, pagination shape, curl examples.
- **[BACKEND_TESTING.md](BACKEND_TESTING.md)** — happy-path and error-path checks.

---

## 7. What I’d do with more time

- Row-level authorization on project/task reads and task updates aligned with product rules.
- Integration tests against a real Postgres (testcontainers or CI service).
- `docker compose` profile or second service if a frontend is added later; env-only secrets for production (no defaults in compose for `JWT_SECRET`).
- OpenAPI export generated from handlers or a single spec file.

---

## Related docs

| File | Purpose |
|------|---------|
| [PROBLEM_STATEMENT.md](PROBLEM_STATEMENT.md) | Original take-home brief (reference) |
| [HLD.md](HLD.md) / [LLD.md](LLD.md) | Design notes |
| [WALKTHROUGH.md](WALKTHROUGH.md) | Code walkthrough |

---

## Tests

```bash
go test ./... -count=1
```

Unit tests use mocked repositories; no database required.
