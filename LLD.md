# Low-Level Design (LLD)

This document explains how every piece of the backend actually works at the code level.

---

## 1. Database Schema

### Tables

```sql
-- USERS: stores account info
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255) UNIQUE NOT NULL,
    password   VARCHAR(255) NOT NULL,   -- bcrypt hash, never plain text
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_users_email ON users(email);  -- fast login lookups


-- PROJECTS: a container for tasks
CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_projects_owner ON projects(owner_id);


-- Custom enum types (PostgreSQL feature — restricts values at DB level)
CREATE TYPE task_status   AS ENUM ('todo', 'in_progress', 'done');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high');

-- TASKS: individual work items within a project
CREATE TABLE tasks (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    status      task_status   NOT NULL DEFAULT 'todo',
    priority    task_priority NOT NULL DEFAULT 'medium',
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date    DATE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_tasks_project  ON tasks(project_id);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_id);
CREATE INDEX idx_tasks_status   ON tasks(status);
```

### Why These Indexes?

| Index | Used By | Query Pattern |
|-------|---------|---------------|
| `idx_users_email` | Login | `WHERE email = $1` — happens on every login |
| `idx_projects_owner` | List projects | `WHERE owner_id = $1` — happens on every page load |
| `idx_tasks_project` | List tasks | `WHERE project_id = $1` — every project detail page |
| `idx_tasks_assignee` | Filter tasks | `WHERE assignee_id = $1` — "my tasks" filter |
| `idx_tasks_status` | Filter tasks | `WHERE status = $1` — status filter pills on UI |

### FK Cascade Rules

```
Delete a user    → CASCADE deletes their projects → CASCADE deletes those tasks
Delete a project → CASCADE deletes its tasks
Delete an assignee → SET NULL on task.assignee_id (task stays, just unassigned)
Delete a task creator → SET NULL on task.created_by (task stays)
```

---

## 2. Go Structs (Model Layer)

Every table has a matching Go struct in `internal/model/`:

```go
// internal/model/user.go
type User struct {
    ID        uuid.UUID `json:"id"        db:"id"`
    Name      string    `json:"name"      db:"name"`
    Email     string    `json:"email"     db:"email"`
    Password  string    `json:"-"         db:"password"`   // json:"-" = never send to client
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

The **`db:"..."`** tags tell sqlx which database column maps to which Go field.
The **`json:"..."`** tags control what appears in API responses.
The **`json:"-"`** on Password means it's NEVER included in any JSON response.

### Request DTOs (Data Transfer Objects)

For every write operation, there's a separate request struct with a `Validate()` method:

```go
type RegisterRequest struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func (r *RegisterRequest) Validate() map[string]string {
    errs := make(map[string]string)
    if r.Name == ""                    { errs["name"]     = "is required" }
    if !emailRegex.MatchString(r.Email){ errs["email"]    = "must be a valid email" }
    if len(r.Password) < 6            { errs["password"]  = "must be at least 6 characters" }
    return errs   // empty map = no errors = valid
}
```

**Why `map[string]string` instead of a single error?**
So the client gets field-level feedback:
```json
{
  "error": "validation failed",
  "fields": {
    "email": "must be a valid email",
    "password": "must be at least 6 characters"
  }
}
```

### Domain Errors

```go
// internal/model/errors.go
var (
    ErrNotFound     = errors.New("not found")
    ErrForbidden    = errors.New("forbidden")
    ErrAlreadyExists = errors.New("already exists")
    ErrUnauthorized = errors.New("unauthorized")
    ErrInvalidInput = errors.New("invalid input")
)
```

These are **domain** errors, not HTTP errors. The service layer returns them. The handler layer maps them to HTTP status codes (404, 403, 409, 401, 400). This keeps the service layer completely HTTP-agnostic.

---

## 3. Repository Layer (SQL Queries)

Each repository is a struct wrapping `*sqlx.DB`:

```go
// internal/repository/user_repo.go
type UserRepo struct {
    db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo {
    return &UserRepo{db: db}
}
```

### Key SQL Patterns

**Creating a record (INSERT with RETURNING):**
```go
func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
    user.ID = uuid.New()  // generate UUID in Go, not DB
    query := `INSERT INTO users (id, name, email, password)
              VALUES ($1, $2, $3, $4) RETURNING created_at`
    return r.db.QueryRowContext(ctx, query,
        user.ID, user.Name, user.Email, user.Password,
    ).Scan(&user.CreatedAt)  // DB fills in created_at
}
```

**Finding a record (SELECT with nil handling):**
```go
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
    var user model.User
    err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE email = $1`, email)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil   // "not found" is not an error — it's a valid result
    }
    return &user, err
}
```

**Dynamic filtering (building WHERE clauses):**
```go
func (r *TaskRepo) FindByProject(ctx context.Context, projectID uuid.UUID, filter model.TaskFilter) ([]model.Task, int, error) {
    conditions := []string{fmt.Sprintf("project_id = $%d", 1)}
    args := []interface{}{projectID}
    argIdx := 2

    if filter.Status != nil {
        conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
        args = append(args, *filter.Status)
        argIdx++
    }
    if filter.Assignee != nil {
        conditions = append(conditions, fmt.Sprintf("assignee_id = $%d", argIdx))
        args = append(args, *filter.Assignee)
        argIdx++
    }
    where := strings.Join(conditions, " AND ")
    // ... then use `where` in both COUNT and SELECT queries
}
```

**Transactional delete (delete tasks, then project, atomically):**
```go
func (r *ProjectRepo) Delete(ctx context.Context, id uuid.UUID) error {
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()  // rollback if anything fails

    tx.ExecContext(ctx, `DELETE FROM tasks WHERE project_id = $1`, id)
    tx.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
    return tx.Commit()  // only commit if both succeed
}
```

---

## 4. Service Layer (Business Logic)

Services are where the "thinking" happens — validation, authorization, orchestration.

### Interface-Based Design

```go
// internal/service/interfaces.go — the service defines what data access it NEEDS
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// internal/service/auth_service.go — depends on the INTERFACE, not the concrete repo
type AuthService struct {
    userRepo  UserRepository   // ← interface type
    jwtSecret string
}
```

**Why?** So tests can plug in a fake repo with no database:
```go
// in test file:
type mockUserRepo struct { users map[string]*model.User }
func (m *mockUserRepo) FindByEmail(ctx, email) (*model.User, error) {
    return m.users[email], nil
}
svc := NewAuthService(&mockUserRepo{...}, "secret")  // no DB needed
```

### Auth Service Logic

```
Register:
  1. Check if email already exists      → ErrAlreadyExists (409)
  2. Hash password with bcrypt(cost=12)  → ~250ms (intentionally slow for security)
  3. Create user in database
  4. Generate JWT token (24h expiry)
  5. Return { token, user }

Login:
  1. Find user by email                  → nil means ErrUnauthorized (401)
  2. Compare password with bcrypt hash   → mismatch means ErrUnauthorized (401)
  3. Generate JWT token
  4. Return { token, user }
```

### Authorization Logic (Who Can Do What)

```go
// ProjectService.Update — only the owner can update
func (s *ProjectService) Update(ctx, projectID, userID, req) (*Project, error) {
    proj := s.projectRepo.FindByID(ctx, projectID)
    if proj == nil          { return model.ErrNotFound }     // 404
    if proj.OwnerID != userID { return model.ErrForbidden }  // 403
    // ... apply updates
}

// TaskService.Delete — owner OR creator can delete
func (s *TaskService) Delete(ctx, taskID, userID) error {
    task := s.taskRepo.FindByID(ctx, taskID)
    proj := s.projectRepo.FindByID(ctx, task.ProjectID)
    if proj == nil {
        // project was deleted — only creator can clean up
        if task.CreatedBy != userID { return model.ErrForbidden }
    } else if proj.OwnerID != userID && task.CreatedBy != userID {
        return model.ErrForbidden  // 403
    }
}
```

---

## 5. Handler Layer (HTTP Glue)

Every handler follows the exact same 4-step pattern:

```go
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
    // STEP 1: Parse input
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

    // STEP 2: Call service
    proj, err := h.service.Create(r.Context(), req, userID)

    // STEP 3: Handle errors
    if err != nil {
        handleServiceError(w, err)
        return
    }

    // STEP 4: Send response
    respondJSON(w, http.StatusCreated, proj)
}
```

**Error mapping (centralized in one function):**
```go
func handleServiceError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, model.ErrNotFound):      respondError(w, 404, "not found")
    case errors.Is(err, model.ErrForbidden):     respondError(w, 403, "forbidden")
    case errors.Is(err, model.ErrAlreadyExists): respondError(w, 409, "already exists")
    case errors.Is(err, model.ErrUnauthorized):  respondError(w, 401, "unauthorized")
    case errors.Is(err, model.ErrInvalidInput):  respondError(w, 400, "invalid input")
    default:                                      respondError(w, 500, "internal error")
    }
}
```

---

## 6. Middleware

### JWT Authentication Middleware

```go
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Get "Authorization: Bearer <token>" header
        // 2. Split to get the token part
        // 3. jwt.Parse with HMAC validation
        // 4. Extract user_id from claims
        // 5. Put user_id into request context
        ctx := context.WithValue(r.Context(), userIDKey, userID)
        next.ServeHTTP(w, r.WithContext(ctx))
        // 6. If any step fails → 401 with JSON error
    })
}

// Handlers retrieve the user_id like this:
func UserIDFromContext(ctx context.Context) uuid.UUID {
    return ctx.Value(userIDKey).(uuid.UUID)
}
```

### Request Logging Middleware

```go
func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        wrapped := &wrappedWriter{ResponseWriter: w, statusCode: 200}
        next.ServeHTTP(wrapped, r)
        slog.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "status", wrapped.statusCode,
            "duration", time.Since(start),
        )
    })
}
```

---

## 7. Router (Route Definitions)

All routes in one file, grouped by auth requirement:

```go
r.Route("/api", func(r chi.Router) {
    // PUBLIC — no token needed
    r.Post("/auth/register", authHandler.Register)
    r.Post("/auth/login", authHandler.Login)

    // PROTECTED — token required (middleware runs first)
    r.Group(func(r chi.Router) {
        r.Use(authMw.Authenticate)    // ← this middleware runs before every handler below

        r.Get("/projects", projectHandler.List)
        r.Post("/projects", projectHandler.Create)
        r.Get("/projects/{id}", projectHandler.GetByID)
        r.Patch("/projects/{id}", projectHandler.Update)
        r.Delete("/projects/{id}", projectHandler.Delete)
        r.Get("/projects/{id}/stats", projectHandler.GetStats)

        r.Get("/projects/{id}/tasks", taskHandler.ListByProject)
        r.Post("/projects/{id}/tasks", taskHandler.Create)

        r.Patch("/tasks/{id}", taskHandler.Update)
        r.Delete("/tasks/{id}", taskHandler.Delete)
    })
})
```

---

## 8. Dependency Wiring (main.go)

```go
func main() {
    cfg := config.Load()                        // read env vars
    db  := database.Connect(cfg.DatabaseURL)    // open postgres connection pool
    database.RunMigrations(cfg.DatabaseURL)      // apply SQL migrations

    // Layer 1: Repositories (talk to database)
    userRepo    := repository.NewUserRepo(db)
    projectRepo := repository.NewProjectRepo(db)
    taskRepo    := repository.NewTaskRepo(db)

    // Layer 2: Services (talk to repositories via interfaces)
    authSvc    := service.NewAuthService(userRepo, cfg.JWTSecret)
    projectSvc := service.NewProjectService(projectRepo, taskRepo)
    taskSvc    := service.NewTaskService(taskRepo, projectRepo)

    // Layer 3: Handlers (talk to services)
    authHandler    := handler.NewAuthHandler(authSvc)
    projectHandler := handler.NewProjectHandler(projectSvc)
    taskHandler    := handler.NewTaskHandler(taskSvc)

    // Middleware
    authMw := middleware.NewAuthMiddleware(cfg.JWTSecret)

    // Wire everything into the router
    r := router.New(authHandler, projectHandler, taskHandler, authMw)

    // Start HTTP server with graceful shutdown
    srv := &http.Server{Addr: ":8080", Handler: r}
    go srv.ListenAndServe()
    // ... wait for SIGTERM, then srv.Shutdown()
}
```

No DI framework. No globals. Every dependency is visible. You can trace any request from router → handler → service → repository → SQL by following the constructors.

---

## 9. Pagination

```go
// Handler parses query params:
page, _ := strconv.Atoi(r.URL.Query().Get("page"))    // ?page=2
limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))  // ?limit=10

// Service clamps values:
if page < 1    { page = 1 }
if limit < 1 || limit > 100 { limit = 20 }

// Repository does the math:
// offset = (page - 1) * limit
// SELECT ... LIMIT $1 OFFSET $2
// Also: SELECT COUNT(*) for total

// Handler wraps response:
respondJSON(w, 200, PaginatedResponse{
    Data:  projects,
    Total: total,
    Page:  page,
    Limit: limit,
})
```

Client gets:
```json
{
  "data": [...],
  "total": 42,
  "page": 2,
  "limit": 10
}
```

---

## 10. Testing Architecture

```
Test Type       What It Tests                  Needs DB?   Location
─────────       ─────────────                  ─────────   ────────
Model tests     Validate() on request structs  No          internal/model/model_test.go
Middleware      JWT parsing + context           No          internal/middleware/auth_test.go
Handler tests   request parsing + responses     No          internal/handler/*_test.go
Service tests   business logic + auth rules     No (mocks)  internal/service/*_test.go
Config tests    env var loading                 No          internal/config/config_test.go
```

All tests are in the **same directory** as the code they test — this is Go convention. There is no separate `tests/` folder.
