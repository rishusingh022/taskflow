# Backend Context — Quick Reference for Developers and AI Agents

> This file gives you the full picture of the backend in under 5 minutes.
> If you're an AI agent picking up this codebase, read this first.

---

## What Is This?

**TaskFlow** — A task management REST API built in Go 1.22. Users register, login (JWT), create projects, add tasks, filter/paginate, and view project stats.

## Tech Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| Language | Go 1.22 | Compiled, fast, strong stdlib |
| Router | chi v5 | stdlib-compatible (`http.Handler`), no lock-in |
| Database | PostgreSQL 16 | Relational data with proper FK constraints |
| SQL driver | sqlx | Named params + struct scanning, keeps SQL visible |
| Migrations | golang-migrate v4 | File-based SQL migrations, up/down support |
| Auth | golang-jwt/v5 + bcrypt | Industry standard, no homebrew crypto |
| Logging | log/slog (stdlib) | Structured JSON logging, zero dependencies |

## Project Structure at a Glance

```
backend/
├── cmd/server/main.go          ← THE entry point. Wires everything together.
├── internal/                   ← All application code (compiler-enforced private)
│   ├── config/                 ← Reads env vars (PORT, DATABASE_URL, JWT_SECRET)
│   ├── database/               ← DB connection pool + migration runner
│   ├── model/                  ← Domain types + request/response DTOs + validation + errors
│   ├── repository/             ← SQL queries (one file per entity)
│   ├── service/                ← Business logic + authorization rules
│   │   └── interfaces.go       ← Repository interfaces (for mocking in tests)
│   ├── handler/                ← HTTP handlers (parse request → call service → write response)
│   ├── middleware/             ← JWT auth + request logging
│   └── router/                 ← All route definitions in one place
├── migrations/                 ← Raw SQL files (000001-000004)
├── Dockerfile                  ← Multi-stage build (~15MB final image)
└── go.mod                      ← Module definition + dependencies
```

## Request Lifecycle (How a Request Flows)

```
Browser/Client
    │
    ▼
[chi Router]  ──── matches route, runs middleware chain
    │
    ▼
[Auth Middleware]  ──── extracts Bearer token, validates JWT, injects user_id into context
    │
    ▼
[Handler]  ──── parses request body/params, validates, calls service
    │
    ▼
[Service]  ──── business logic, authorization checks (e.g., "is this user the project owner?")
    │
    ▼
[Repository]  ──── executes SQL query via sqlx, returns domain type
    │
    ▼
[Handler]  ──── writes JSON response with status code
```

## Key Design Decisions

### 1. Four Layers (Not Three)

The reference Node.js projects merged business logic and database queries in `services/`. We split them:

- **Handler** — HTTP only (parse, respond)
- **Service** — Business rules (auth checks, validation, orchestration)
- **Repository** — Database only (SQL queries, transactions)
- **Model** — Types and validation

This separation allows testing services with mock repositories — no database needed.

### 2. Interfaces for Testability

```go
// service/interfaces.go
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}
```

Services accept interfaces. Tests inject mock implementations. Repository structs satisfy these interfaces with real SQL.

### 3. Constructor-Based Dependency Injection

```go
// main.go wiring
userRepo := repository.NewUserRepo(db)
authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
authHandler := handler.NewAuthHandler(authSvc)
```

No DI framework. No globals. Everything is explicit and traceable.

### 4. Domain Errors (Not HTTP Errors)

```go
// model/errors.go
var ErrNotFound = errors.New("not found")
var ErrForbidden = errors.New("forbidden")
```

Services return domain errors. Handlers map them to HTTP status codes:

```go
// handler/project_handler.go
func handleServiceError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, model.ErrNotFound):
        respondError(w, http.StatusNotFound, "not found")
    case errors.Is(err, model.ErrForbidden):
        respondError(w, http.StatusForbidden, "forbidden")
    ...
    }
}
```

This keeps services agnostic to HTTP.

## Authorization Rules

| Action | Who Can Do It |
|--------|--------------|
| Create project | Any authenticated user |
| Update/delete project | Project owner only |
| View project + tasks | Any authenticated user |
| Create task | Any authenticated user |
| Update task (status/fields) | Any authenticated user |
| Delete task | Project owner OR task creator |

## Database Schema (4 Migrations)

```
000001 — users table (id UUID PK, name, email UNIQUE, password, created_at)
000002 — projects table (id UUID PK, name, description, owner_id FK→users, created_at)
000003 — tasks table (id UUID PK, title, description, status ENUM, priority ENUM,
                      project_id FK→projects, assignee_id FK→users, created_by FK→users,
                      due_date, created_at, updated_at)
000004 — seed data (3 users, 3 projects, 10 tasks)
```

**Enum types** (PostgreSQL native):
- `task_status`: todo, in_progress, done
- `task_priority`: low, medium, high

## API Routes Summary

All routes under `/api`:

```
POST   /api/auth/register                    ← public
POST   /api/auth/login                       ← public
GET    /api/projects                         ← protected (paginated)
POST   /api/projects                         ← protected
GET    /api/projects/{id}                    ← protected (includes tasks)
PATCH  /api/projects/{id}                    ← protected (owner only)
DELETE /api/projects/{id}                    ← protected (owner only)
GET    /api/projects/{id}/stats              ← protected
GET    /api/projects/{id}/tasks              ← protected (filtered, paginated)
POST   /api/projects/{id}/tasks              ← protected
PATCH  /api/tasks/{id}                       ← protected
DELETE /api/tasks/{id}                       ← protected (owner or creator)
GET    /health                               ← public (no /api prefix)
```

## Test Files and What They Cover

```
config/config_test.go        ← env var loading (missing secret, custom port, custom DB)
middleware/auth_test.go      ← JWT validation (valid, no header, wrong format, expired, wrong secret)
handler/auth_handler_test.go ← register/login validation errors, invalid JSON
handler/response_test.go     ← JSON response helpers, pagination struct
model/model_test.go          ← all Validate() methods on all request types
service/auth_service_test.go ← register, login, duplicate email, wrong password (mock repo)
service/project_service_test.go ← CRUD, owner authorization, stats, not-found (mock repo)
service/task_service_test.go ← CRUD, due date parsing, delete auth, pagination (mock repo)
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP server port |
| `DATABASE_URL` | No | `postgres://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | **Yes** | — | HMAC signing key for JWT tokens |

## Common Tasks for Agents

### "Add a new entity" (e.g., Comments)
1. Create `internal/model/comment.go` — struct + request types + Validate()
2. Create `migrations/000005_create_comments.up.sql` + `.down.sql`
3. Create `internal/repository/comment_repo.go` — SQL queries
4. Add `CommentRepository` interface to `internal/service/interfaces.go`
5. Create `internal/service/comment_service.go` — business logic
6. Create `internal/handler/comment_handler.go` — HTTP handlers
7. Register routes in `internal/router/router.go`
8. Wire dependencies in `cmd/server/main.go`

### "Add a new field to an existing entity"
1. Create a new migration SQL file
2. Update the model struct in `internal/model/`
3. Update the repository SQL queries
4. Update handler/service if the field needs special handling

### "Add a new endpoint"
1. Add handler method to the relevant handler struct
2. Register the route in `internal/router/router.go`
3. Add service method if business logic is needed
