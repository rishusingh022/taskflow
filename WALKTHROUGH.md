# Code Walkthrough

A beginner-friendly, file-by-file explanation of the entire backend. If you've never seen a Go project before, start here.

---

## How to Read This Guide

We follow the path of an HTTP request through the codebase:

```
Browser sends request
    ↓
cmd/server/main.go         (starts the server)
    ↓
internal/router/router.go  (matches the URL to a handler)
    ↓
internal/middleware/        (checks the JWT token)
    ↓
internal/handler/           (parses the HTTP request, calls a service)
    ↓
internal/service/           (business logic — validation, authorization)
    ↓
internal/repository/        (writes/reads from the database)
    ↓
internal/model/             (data structures used everywhere)
    ↓
internal/database/          (connects to PostgreSQL, runs migrations)
    ↓
internal/config/            (reads environment variables)
```

---

## 1. `cmd/server/main.go` — The Starting Point

Every Go program starts with `main()`. This file does 5 things in order:

```
1. Load config       → reads DATABASE_URL, JWT_SECRET from environment
2. Connect to DB     → opens a PostgreSQL connection pool
3. Run migrations    → creates tables if they don't exist
4. Wire everything   → creates repos → services → handlers → router
5. Start the server  → listens on port 8080, handles graceful shutdown
```

**What's "wiring"?** It's manually connecting the pieces:
- The `UserRepo` needs a database connection → give it one
- The `AuthService` needs a `UserRepo` → give it one
- The `AuthHandler` needs an `AuthService` → give it one
- The router needs the handler → give it one

No magic, no framework. You can read the constructor calls top to bottom and understand exactly what depends on what.

**Graceful shutdown:** When you press Ctrl+C, the server doesn't just die. It:
1. Stops accepting new connections
2. Waits for in-progress requests to finish (up to 10 seconds)
3. Closes the database connection
4. Then exits

---

## 2. `internal/config/config.go` — Reading Environment Variables

This is the simplest file in the project. It defines a struct:

```go
type Config struct {
    Port        string  // which port to listen on (default: "8080")
    DatabaseURL string  // PostgreSQL connection string
    JWTSecret   string  // secret key for signing tokens
}
```

The `Load()` function reads each field from `os.Getenv()`. If `JWT_SECRET` is missing, it panics — because running without a secret would be a security problem.

**Why a struct instead of reading env vars everywhere?**
So you only read environment variables once, in one place. Every other file receives the config it needs through function parameters.

---

## 3. `internal/database/postgres.go` — Connecting to PostgreSQL

One function: `Connect(databaseURL) → *sqlx.DB`

It calls `sqlx.Connect("postgres", url)` and returns the connection pool. `sqlx` is a library that extends Go's standard `database/sql` with the ability to scan query results directly into structs.

This returns a **connection pool**, not a single connection. Go manages multiple connections automatically — you just call `db.QueryContext()` and it picks an available connection.

---

## 4. `internal/database/migrate.go` — Creating Tables

`RunMigrations(databaseURL)` runs the SQL files in the `migrations/` folder against the database.

The migration files follow a naming convention:
```
000001_create_users_table.up.sql      → creates the users table
000001_create_users_table.down.sql    → drops the users table (for rollbacks)
000002_create_projects_table.up.sql   → creates the projects table
...
```

The library (`golang-migrate`) tracks which migrations have already been applied in a special `schema_migrations` table. So running this multiple times is safe — it only applies new migrations.

**One quirk:** The migration file path differs between local development (`file://migrations`) and Docker containers (`file:///migrations`). The code tries the local path first, then falls back to the Docker path.

---

## 5. `internal/model/` — Data Structures

This is the "dictionary" of the project. Every data shape is defined here.

### `user.go`

```go
type User struct {
    ID        uuid.UUID `json:"id"       db:"id"`
    Name      string    `json:"name"     db:"name"`
    Email     string    `json:"email"    db:"email"`
    Password  string    `json:"-"        db:"password"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

Pay attention to the tags:
- **`db:"id"`** — tells sqlx "this field maps to the `id` column in the database"
- **`json:"id"`** — tells Go's JSON encoder "call this `id` in JSON output"
- **`json:"-"`** — the minus sign means "NEVER include this in JSON". The password hash is never sent to the client. Ever.

There are also **request structs** like `RegisterRequest` and `LoginRequest`. These represent what the client sends to us. Each one has a `Validate()` method that returns a map of field-level errors.

### `project.go`

Same pattern: a `Project` struct with tags, plus `CreateProjectRequest` and `UpdateProjectRequest` with `Validate()` methods.

### `task.go`

Same pattern, but also defines the enum types:

```go
type TaskStatus string
const (
    StatusTodo       TaskStatus = "todo"
    StatusInProgress TaskStatus = "in_progress"
    StatusDone       TaskStatus = "done"
)
```

Go doesn't have built-in enums. This `const` block + custom type achieves the same thing. The `Valid()` method checks if a string is one of the allowed values.

### `errors.go`

Five simple error variables:

```go
var ErrNotFound      = errors.New("not found")
var ErrForbidden     = errors.New("forbidden")
var ErrAlreadyExists = errors.New("already exists")
var ErrUnauthorized  = errors.New("unauthorized")
var ErrInvalidInput  = errors.New("invalid input")
```

These are the only errors the service layer can return. The handler layer maps each one to an HTTP status code. This keeps the service layer completely unaware of HTTP.

---

## 6. `internal/repository/` — Database Access

Each repository is a thin wrapper around SQL queries. No business logic here — just "save this" and "find that."

### `user_repo.go`

Three methods:
- **`Create(user)`** — `INSERT INTO users ... RETURNING created_at` (the DB fills in the timestamp)
- **`FindByEmail(email)`** — `SELECT * FROM users WHERE email = $1` (used during login)
- **`FindByID(id)`** — `SELECT * FROM users WHERE id = $1` (used to look up who made a request)

A key pattern: when `FindByEmail` finds no rows, it returns `(nil, nil)` — not an error. "Not found" is a valid outcome, not an exception. The service layer decides what to do with a nil result.

### `project_repo.go`

Standard CRUD plus two extras:
- **`FindByOwner(ownerID, page, limit)`** — paginated list of a user's projects. Returns `([]Project, totalCount, error)`.
- **`GetStats(projectID)`** — runs 3 queries to count tasks by status and assignee. Used for the project dashboard widget.
- **`Delete(projectID)`** — uses a **transaction** to delete tasks first, then the project. Either both succeed or neither does.

### `task_repo.go`

The most complex repo because of **dynamic filtering**:

```go
// User might filter by status, priority, or assignee — or any combination
if filter.Status != nil {
    conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
    args = append(args, *filter.Status)
    argIdx++
}
```

This builds the WHERE clause piece by piece, only including conditions that the user actually provided. The `$1`, `$2` placeholders prevent SQL injection — user input is NEVER concatenated into the query string.

---

## 7. `internal/service/` — Business Logic

This is where the "thinking" happens. Services validate input, check permissions, and orchestrate calls to repositories.

### `interfaces.go` — The Contract

Before looking at the services, look at this file. It defines **interfaces** — contracts saying "I need a thing that can do X, Y, and Z":

```go
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}
```

The real `UserRepo` from the repository package implements this interface. But in tests, we create a **mock** that also implements it — one that doesn't need a database at all. This is how we test business logic in isolation.

### `auth_service.go`

Two methods: `Register` and `Login`.

**Register flow:**
1. Check if email already taken → return `ErrAlreadyExists` (client sees 409)
2. Hash the password with bcrypt (cost 12 → ~250ms, intentionally slow to resist brute force)
3. Save the user to the database
4. Generate a JWT token valid for 24 hours
5. Return the token and user info

**Login flow:**
1. Find user by email → if not found, return `ErrUnauthorized`
2. Compare submitted password against stored hash → if wrong, return `ErrUnauthorized`  
   (same error for "no such user" and "wrong password" — so attackers can't tell which)
3. Generate JWT token
4. Return token and user info

**JWT token contents:**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "exp": 1709308800
}
```

The token is signed with HMAC-SHA256 using the `JWT_SECRET` env var. Anyone with the secret can create valid tokens — that's why it must stay secret.

### `project_service.go`

CRUD operations with ownership checks:

- **Create** — straightforward, sets `OwnerID = userID`
- **List** — calls `repo.FindByOwner(userID, page, limit)` — you only see YOUR projects
- **GetByID** — returns the project if it exists
- **Update** — checks `project.OwnerID == userID` → if not, returns `ErrForbidden`
- **Delete** — same ownership check, then deletes
- **GetStats** — calls the repo's stats query

The ownership check is 2 lines but super important:
```go
if project.OwnerID != userID {
    return nil, model.ErrForbidden
}
```
Without this, any logged-in user could modify any project.

### `task_service.go`

Similar to project service, but authorization is more nuanced:

- **Create** — must be the project owner to add tasks
- **Update** — project owner OR task creator can update
- **Delete** — project owner OR task creator can delete

There's a nil check on the project in the Delete method:
```go
if proj == nil {
    // Project was already deleted. Only the task creator can clean up orphaned tasks.
    if task.CreatedBy != userID {
        return model.ErrForbidden
    }
}
```
This handles the edge case where a project is deleted but tasks haven't been cascade-deleted yet (race condition).

---

## 8. `internal/handler/` — HTTP Layer

Handlers are the translation layer between HTTP and business logic. They don't contain business logic — they just:
1. Parse the HTTP request (JSON body, URL params, query strings)
2. Call a service method
3. Translate the result (or error) into an HTTP response

### `response.go` — Shared Response Helpers

Three functions used by every handler:

```go
respondJSON(w, statusCode, data)        // → 200 {"id":"...",...}
respondError(w, statusCode, message)    // → 400 {"error":"invalid input"}
respondValidationError(w, fieldErrors)  // → 400 {"error":"...","fields":{"name":"required"}}
```

These ensure every API response has the same shape. The client never has to guess the format.

### `auth_handler.go`

Two endpoints:
- **POST /api/auth/register** — parse JSON body → validate → call authService.Register → return token
- **POST /api/auth/login** — parse JSON body → validate → call authService.Login → return token

A key detail: the handler calls `handleServiceError(w, err)` which is a switch statement mapping domain errors to HTTP codes:
```
ErrAlreadyExists → 409 Conflict
ErrUnauthorized  → 401 Unauthorized
(anything else)  → 500 Internal Server Error
```

### `project_handler.go`

Five endpoints. The Create handler is typical:

```go
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req model.CreateProjectRequest
    if err := decodeJSON(r, &req); err != nil {
        respondError(w, 400, "invalid request body")
        return
    }
    if errs := req.Validate(); len(errs) > 0 {
        respondValidationError(w, errs)
        return
    }
    userID := middleware.UserIDFromContext(r.Context())
    proj, err := h.service.Create(r.Context(), req, userID)
    if err != nil {
        handleServiceError(w, err)
        return
    }
    respondJSON(w, http.StatusCreated, proj)
}
```

Notice the pattern: parse → validate → call service → handle error → respond. Every handler follows this exact pattern.

### `task_handler.go`

Similar to project handler, but also handles URL params for filtering:
```go
status := r.URL.Query().Get("status")       // ?status=todo
assignee := r.URL.Query().Get("assignee")   // ?assignee=<uuid>
```

These get packed into a `TaskFilter` struct and passed to the service.

---

## 9. `internal/middleware/` — Request Pipeline

### `auth.go` — JWT Authentication

This middleware runs BEFORE protected handlers. It:

1. Reads the `Authorization` header
2. Checks it starts with `Bearer `
3. Parses and validates the JWT token
4. Extracts `user_id` from the token claims
5. Stores the user ID in the request's context
6. Calls the next handler

If any step fails, it returns 401 with a JSON error — the handler never runs.

The user ID is stored using Go's `context.WithValue()`. Handlers retrieve it with:
```go
userID := middleware.UserIDFromContext(r.Context())
```

This is how handlers know "who" is making the request without parsing the token themselves.

### `logging.go` — Request Logging

Wraps every request with timing and logging:

```
2024-03-09 10:15:23 INFO request method=POST path=/api/projects status=201 duration=12ms
```

It uses a `wrappedWriter` trick: it wraps the standard `ResponseWriter` to intercept the status code (which is normally write-only).

---

## 10. `internal/router/router.go` — URL Routing

This file wires URLs to handler functions. It uses the `chi` router library.

The file is organized into two blocks:

1. **Public routes** (no auth required):
   - `POST /api/auth/register`
   - `POST /api/auth/login`

2. **Protected routes** (wrapped in `r.Use(authMw.Authenticate)`):
   - All project endpoints
   - All task endpoints

The `r.Use(authMw.Authenticate)` line means: "before running any handler in this group, run the auth middleware first." If the middleware rejects the request, the handler never runs.

CORS is configured here too — it allows the React frontend (running on a different port during development) to make requests.

---

## 11. SQL Migrations (`migrations/`)

Four migration pairs (up + down):

| # | What It Does |
|---|-------------|
| 1 | Creates `users` table + email index |
| 2 | Creates `projects` table + FK to users, owner index |
| 3 | Creates task_status and task_priority enum types |
| 4 | Creates `tasks` table + FKs to projects/users, 3 indexes |

**Up** migrations create things. **Down** migrations undo them (drop tables/types). You rarely run down migrations in production, but they're useful during development if you need to redo a migration.

---

## 12. Test Files

Tests live next to the code they test (Go convention):

| Test File | What It Tests | Needs Database? |
|-----------|---------------|-----------------|
| `config/config_test.go` | Env var loading, defaults | No |
| `middleware/auth_test.go` | JWT token parsing | No |
| `model/model_test.go` | Request validation (Validate methods) | No |
| `handler/response_test.go` | JSON response helpers | No |
| `handler/auth_handler_test.go` | Register/login HTTP handling | No (uses mock service) |
| `service/auth_service_test.go` | Register/login business logic | No (uses mock repo) |
| `service/project_service_test.go` | Project CRUD + ownership | No (uses mock repo) |
| `service/task_service_test.go` | Task CRUD + permissions | No (uses mock repo) |

**Key testing pattern:** Every test creates its own mock (a struct that implements the repository interface) and injects it into the service. No database, no network, no Docker. Tests run in milliseconds.

Example mock:
```go
type mockUserRepo struct {
    users map[string]*model.User
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
    return m.users[email], nil
}
```

---

## File Tree (Summary)

```
backend/
├── cmd/server/main.go            ← START HERE (entry point)
├── internal/
│   ├── config/config.go          ← reads env vars
│   ├── database/
│   │   ├── postgres.go           ← DB connection
│   │   └── migrate.go            ← runs SQL migrations
│   ├── model/
│   │   ├── user.go               ← User struct + request validation
│   │   ├── project.go            ← Project struct + validation
│   │   ├── task.go               ← Task struct + enums + validation
│   │   └── errors.go             ← domain errors (ErrNotFound, etc.)
│   ├── repository/
│   │   ├── user_repo.go          ← SQL for users
│   │   ├── project_repo.go       ← SQL for projects (with transactions)
│   │   └── task_repo.go          ← SQL for tasks (with dynamic filters)
│   ├── service/
│   │   ├── interfaces.go         ← contracts for testability
│   │   ├── auth_service.go       ← register/login logic
│   │   ├── project_service.go    ← project CRUD + ownership
│   │   └── task_service.go       ← task CRUD + permissions
│   ├── handler/
│   │   ├── response.go           ← JSON response helpers
│   │   ├── auth_handler.go       ← HTTP endpoints for auth
│   │   ├── project_handler.go    ← HTTP endpoints for projects
│   │   └── task_handler.go       ← HTTP endpoints for tasks
│   ├── middleware/
│   │   ├── auth.go               ← JWT verification
│   │   └── logging.go            ← request logging
│   └── router/router.go          ← URL → handler mapping
└── migrations/                   ← SQL files for DB setup
```

---

## How to Trace a Request

Let's trace what happens when a user creates a new project:

```
1. POST /api/projects with body {"name": "My Project"}

2. router.go → matches POST /api/projects → runs auth middleware first

3. middleware/auth.go
   → reads "Authorization: Bearer eyJhbG..."
   → validates the JWT token
   → extracts user_id = "550e8400-..."
   → stores it in request context

4. handler/project_handler.go → Create()
   → decodeJSON(r, &req) → reads body into CreateProjectRequest{Name: "My Project"}
   → req.Validate() → name is not empty → no errors
   → middleware.UserIDFromContext(ctx) → gets "550e8400-..."
   → calls projectService.Create(ctx, req, userID)

5. service/project_service.go → Create()
   → builds a Project struct with Name, OwnerID
   → calls projectRepo.Create(ctx, &project)

6. repository/project_repo.go → Create()
   → generates UUID for the project
   → runs: INSERT INTO projects (id, name, description, owner_id) VALUES ($1, $2, $3, $4)
   → PostgreSQL creates the row

7. Back up the chain:
   → repo returns the project with generated timestamps
   → service returns it to the handler
   → handler calls respondJSON(w, 201, project)
   → client receives: {"id": "...", "name": "My Project", "owner_id": "550e8400-...", ...}
```

The request went through 6 layers and back. Each layer only knows about the one directly below it. The handler doesn't know SQL exists. The service doesn't know HTTP exists. The repo doesn't know about authentication. This separation is the entire point of the architecture.
