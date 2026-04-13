# TaskFlow — Backend submission

**Role:** Backend engineer path — Go API + PostgreSQL + Docker + Postman collection. No React app (not required for this role per the brief).

---

## 1. Overview

TaskFlow is a small task-management product: users register and log in (JWT), own projects, and manage tasks (status, priority, assignee, due date). This repository implements the **REST API** and **database** only.

| Layer | Choice |
|--------|--------|
| Language | Go 1.23 (toolchain; CI can pin 1.22+ if needed) |
| HTTP | chi v5, `net/http` server |
| DB | PostgreSQL 16 (Alpine in Docker) |
| Access | sqlx + parameterized SQL |
| Migrations | golang-migrate (up + down per version) |
| Auth | JWT HS256, bcrypt cost **12** |
| Logging | `slog` JSON to stdout |

**Base URL:** `http://localhost:8080`
**API prefix:** `/api` (e.g. `POST /api/auth/login`)
**Health (no auth):** `GET /health` → `{"status":"ok"}`

---

## 2. Assignment brief (backend-relevant excerpts)

Below is the brief **trimmed to what this repo implements** (backend + infra + README rubric). Full-stack UI requirements are out of scope for this submission.

### Who builds what (reference)

| Role | Backend (Go) | Frontend | Docker + README |
|------|--------------|----------|-----------------|
| Full Stack | Required | Required | Required |
| **Backend** | **Required** | Not required — Postman/tests instead | **Required** |
| Frontend | Not required | Required | Required |

### Data model (implemented)

- **User:** `id` (UUID), `name`, `email` (unique), `password` (bcrypt hash), `created_at`
- **Project:** `id`, `name`, `description` (optional), `owner_id` → User, `created_at`
- **Task:** `id`, `title`, `description` (optional), `status` enum (`todo` \| `in_progress` \| `done`), `priority` enum (`low` \| `medium` \| `high`), `project_id`, `assignee_id` (nullable), `due_date` (optional), `created_at`, `updated_at`, **`created_by`** (who created the row — useful for audit; **delete** is enforced via project ownership, not creator)

Schema is **PostgreSQL** only; changes are **SQL migrations**, not ORM auto-migrate.

### Backend API (paths in this implementation use `/api`)

| Area | Spec path | This repo |
|------|-----------|-----------|
| Register | `POST /auth/register` | `POST /api/auth/register` |
| Login | `POST /auth/login` | `POST /api/auth/login` |
| Projects CRUD + list | `/projects…` | `/api/projects…` |
| Tasks | `/projects/:id/tasks`, `/tasks/:id` | `/api/...` |
| Non-auth | Bearer JWT | Same |

**Auth rules implemented**

- Passwords: **bcrypt**, cost **≥ 12**
- JWT: **24h** expiry; claims include **`user_id`**, **`email`**
- Protected routes: `Authorization: Bearer <token>`

**General API**

- JSON responses; validation → **400** `{ "error": "validation failed", "fields": { ... } }`
- Unauthenticated → **401** `{ "error": "unauthorized" }` (middleware); wrong login password → **401** `{ "error": "invalid email or password" }` (intentional message, still 401)
- Forbidden → **403** `{ "error": "forbidden" }`
- Not found → **404** `{ "error": "not found" }`
- Rate limited → **429** (from `go-chi/httprate`; no custom JSON body — client should back off and retry later)
- Structured logging (**slog**); **graceful shutdown** on SIGINT/SIGTERM

**Bonus (done)**

- Pagination: `?page=&limit=` on project list and task list (project list max **50** per page in service layer)
- `GET /api/projects/:id/stats` — counts by status and by assignee
- Unit tests (handlers/services/middleware/models) — **3+** test files covering auth and task logic (mocked repos; not full HTTP integration tests against Postgres)

### Infrastructure (this repo)

- `docker-compose.yml` at repo root: **Postgres + API** (backend-only; no React container — acceptable for **backend** role; full-stack would add a `frontend` service)
- `docker compose up --build` after `cp .env.example .env` — no extra migrate step
- Postgres credentials and API env vars configurable via **`.env`** (Compose substitutes `${VAR}`)
- **`Dockerfile`:** multi-stage build (Go build → minimal Alpine runtime)
- Migrations run **on API startup**; every migration has **up** and **down**
- Seed (`000004_seed_data`): **≥ 1 user** with known password, **≥ 1 project**, **≥ 3 tasks** with different statuses

### README rubric (this document)

| Section | Where |
|---------|--------|
| 1. Overview | §1 + stack table |
| 2. Architecture decisions | §3 |
| 3. Running locally | §4 |
| 4. Migrations | §5 |
| 5. Test credentials | §6 |
| 6. API reference | §7 + `taskflow.postman_collection.json` |
| 7. What you’d do with more time | §9 |

### Automatic disqualifiers (checked)

| Rule | Status |
|------|--------|
| Runs with `docker compose up` | Yes (API on **8080**, not 3000) |
| Real migrations | Yes (golang-migrate SQL files) |
| Passwords plaintext | No — bcrypt only |
| JWT secret only in source | No — required via **`JWT_SECRET`** env (compose/.env supply it; do not commit `.env`) |
| README present | Yes |

---

## 3. Architecture decisions & operations

### Code layout

- **`cmd/server`:** process entry, wiring, HTTP server lifecycle
- **`internal/handler`:** HTTP decode/encode, status codes
- **`internal/service`:** rules (ownership, filters, pagination caps)
- **`internal/repository`:** SQL only
- **`internal/middleware`:** JWT
- **`internal/router`:** routes under `/api`

### Access control (production-oriented)

Every protected read or write checks **who the JWT user is**, not only whether they are logged in.

**Project access** for `GET /projects/:id`, `GET /projects/:id/stats`, `GET /projects/:id/tasks`, and task mutations inside that project: the user must be either the **project owner** or **assigned to at least one task** in that project (see `UserHasProjectAccess` in the project repository).

| Action | Who may do it |
|--------|----------------|
| List projects | Owner or assignee (projects appear if you own them or have a task assigned there) |
| Create project | Any authenticated user (they become owner) |
| Get / stats / list tasks for a project | Owner or assignee on a task in that project |
| Update / delete project | Owner only |
| Create task in a project | **Owner only** |
| Update task | Owner **or** current assignee of that task |
| Delete task | **Owner of the project only** (being the creator or assignee is not enough) |

**Note:** There is no separate “admin” role in this codebase; “roles” are implied by ownership and assignment.

### Rate limiting

Per-IP limits (via [httprate](https://github.com/go-chi/httprate)):

- **`POST /api/auth/register`** and **`POST /api/auth/login`:** 30 requests per minute per IP — bcrypt is slow on purpose; this reduces credential stuffing and accidental overload.
- **All other `/api/*` routes (with Bearer JWT):** 600 requests per minute per IP — enough for normal use, still bounded.
- **`GET /health`:** not throttled (suitable for load balancers and probes).

Tune these in `internal/router/router.go` if you deploy behind a trusted proxy (you may want `LimitByRealIP` and correct `X-Forwarded-For` handling).

---

## 4. Running locally

Assumes **Docker** + **Docker Compose** only.

```bash
git clone https://github.com/<your-username>/taskflow.git
cd taskflow
cp .env.example .env
docker compose up --build -d
```

- **Health:** http://localhost:8080/health
- **API:** http://localhost:8080/api

**If seeded login fails** (stale DB volume from an old seed hash):

```bash
docker compose down -v
docker compose up --build -d
```

Seed uses `ON CONFLICT (email) DO UPDATE` for passwords so **re-applying** migration 4 on a fresh DB always stores the correct bcrypt for `password123`.

---

## 5. Migrations

Applied **automatically** when the API container starts (`internal/database/migrate.go`). If migrations fail, the process **exits** (no half-ready API).

---

## 6. Test credentials (seed)

```
Email:    test@example.com
Password: password123
```

Also seeded (same password): `priya@example.com`, `amit@example.com`.

**Seeded project id (stable):** `a1b2c3d4-e5f6-7890-abcd-ef1234567890` (Website Redesign)

**Bcrypt:** Seed SQL stores one **verified** bcrypt string (cost 12) for `password123` and applies it to all three demo users so reviewers always have a known login. **Production:** real users are created via `Register`, which calls `bcrypt.GenerateFromPassword` — each password gets its **own** salt and hash (never copy-paste hashes for real accounts).

---

## 7. API reference & Postman

Import **`taskflow.postman_collection.json`** into Postman.

| Collection area | Purpose |
|-----------------|--------|
| **Smoke — reviewer / CI** | `GET /health` → login `test@example.com` → `GET /api/projects` (assertions on 200 + token) |
| **Flow A** | Seeded user deep dive (projects, tasks, filters, stats, patch task) |
| **Flow B** | Register with **unique email** (`postman+{timestamp}@example.com`) → full CRUD (safe to re-run) |
| **Error cases** | 401/403/404/400 samples; **E8a → E8b** Priya cannot PATCH Raju’s project; **E8c** Priya cannot read a project where she has no tasks |

Collection variables: `tf_host`, `tf_token`, `tf_project_id`, `tf_task_id`, `tf_reg_email`, `tf_token_priya`.

**List response shape** (pagination): `{ "data": [...], "total", "page", "limit", "total_pages" }` — differs from frontend-only mock appendix (`projects` / `tasks` keys); this collection matches the **real** API.

---

## 8. Requirements checklist (backend)

- [x] PostgreSQL + SQL migrations (up/down), no ORM auto-migrate
- [x] `POST /api/auth/register`, `POST /api/auth/login`, JWT 24h, `user_id` + `email` claims
- [x] bcrypt cost ≥ 12
- [x] Projects: list (owner or assignee), CRUD, owner-only patch/delete
- [x] Tasks: list + filters `status`, `assignee`; create (owner only); patch (owner or assignee); delete (project owner only)
- [x] 400 validation shape; 401 vs 403 vs 404 semantics
- [x] `slog` logging; graceful shutdown
- [x] Bonus: pagination, stats endpoint, tests
- [x] Docker Compose + multi-stage Dockerfile + `.env.example`
- [x] Seed: user + project + 3+ tasks, multiple statuses
- [x] Postman collection for reviewers

**Gaps / not claimed**

- [ ] Integration tests hitting real Postgres (e.g. testcontainers)
- [ ] Distributed rate limits (Redis) when running multiple API replicas
- [ ] Fine-grained project membership beyond “has an assigned task”

---

## 9. What I’d do with more time

- Integration tests + CI job on `docker compose`
- proof-of-work on register in public-facing deployments
- OpenAPI spec checked into repo
- Explicit project membership table (invite / role) instead of inferring access from assignee only
- Separate read replicas / caching only if product justified

---

## 10. Tests (local)

```bash
go test ./... -count=1
```

Uses mocked repositories — no DB required.
