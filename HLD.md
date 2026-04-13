# High-Level Design (HLD)

## What is TaskFlow?

TaskFlow is a task management system. Think of it like a simplified Jira or Trello. Users sign up, create projects, add tasks to those projects, and track what's done.

The backend is a REST API written in Go. It talks to a PostgreSQL database. A React frontend talks to this API.

## The Big Picture

```
┌─────────────────┐         ┌──────────────────┐         ┌──────────────┐
│  React Frontend │───HTTP──▶│   Go API Server  │───SQL──▶│  PostgreSQL  │
│  (browser)      │◀──JSON──│   (port 8080)    │◀───────│  (port 5432) │
└─────────────────┘         └──────────────────┘         └──────────────┘
```

That's it. Three boxes. One frontend, one backend, one database.

## Why a Monolith?

The reference projects we studied (`final-eval-auth` + `final-eval-be`) used microservices — two separate Node.js servers talking to each other over HTTP. That design had problems:

```
❌ Microservices approach (reference project):

Browser → Content API (port 3003) → Auth API (port 3001) → Database
                                  ↑
                        extra HTTP call every request
                        hardcoded "localhost:3001" breaks in Docker
```

```
✅ Monolith approach (our project):

Browser → Go API (port 8080) → Database
              ↑
    auth is just a function call (microseconds)
    one container, one port, one deployment
```

**Why is one server better here?**

| Problem with 2 servers | How 1 server solves it |
|---|---|
| Auth validation = HTTP call (adds ~10ms latency per request) | Auth validation = function call (adds ~0.001ms) |
| `http://localhost:3001` is hardcoded — breaks in Docker | No inter-service calls needed |
| Two Dockerfiles, two health checks, two log streams | One of everything |
| Shared types (User struct) duplicated in both projects | One `model/` package, used everywhere |
| If auth server goes down, content server is broken too | Single process — either it's all up or all down |

This doesn't mean microservices are bad. They're great when you have 50 engineers and need teams to deploy independently. For a 1-5 person project, a monolith is the right call.

## What the API Does

There are 3 main areas:

### 1. Authentication
- Register a new account (name, email, password)
- Login and get a JWT token
- Every other request needs that token in the `Authorization` header

### 2. Projects
- Create projects (you become the owner)
- List your projects (ones you own or have tasks in)
- Update/delete projects (owner only)
- View project statistics (how many tasks by status)

### 3. Tasks
- Create tasks within a project (any logged-in user)
- Filter tasks by status (todo, in_progress, done)
- Update task fields (any logged-in user)
- Delete tasks (project owner or the person who created the task)

## How Security Works

```
1. User registers:
   password → bcrypt hash (cost 12) → stored in database
   (original password is never stored)

2. User logs in:
   email + password → find user → bcrypt.Compare → generate JWT token
   (token expires in 24 hours)

3. Every protected request:
   Authorization: Bearer <token>
   → middleware extracts token
   → validates signature with server's secret key
   → extracts user_id
   → passes user_id to the handler via request context

4. Authorization checks happen in the service layer:
   "Is this user the project owner?" → if not, return 403 Forbidden
```

## Data Model (What's in the Database)

```
USERS                     PROJECTS                   TASKS
─────                     ────────                   ─────
id (UUID)                 id (UUID)                  id (UUID)
name                      name                       title
email (unique)            description                description
password (bcrypt hash)    owner_id → USERS.id        status (todo/in_progress/done)
created_at                created_at                 priority (low/medium/high)
                                                     project_id → PROJECTS.id
                                                     assignee_id → USERS.id
                                                     created_by → USERS.id
                                                     due_date
                                                     created_at
                                                     updated_at
```

**Relationships:**
- A user **owns** many projects (one-to-many)
- A project **has** many tasks (one-to-many)
- A task is **assigned to** one user (many-to-one, optional)
- A task is **created by** one user (many-to-one)

**What happens when things are deleted:**
- Delete a user → all their projects get deleted too (CASCADE)
- Delete a project → all its tasks get deleted too (CASCADE)
- Delete an assigned user → task's assignee becomes null (SET NULL)

## Technology Choices

| What | We Used | Why |
|---|---|---|
| Language | Go 1.22 | Fast, compiled, great stdlib, strong typing |
| HTTP Router | chi v5 | Compatible with Go's `net/http` (no lock-in) |
| Database | PostgreSQL 16 | Relational data, ACID transactions, mature |
| SQL Driver | sqlx | Keeps SQL visible — no ORM magic hiding queries |
| Migrations | golang-migrate | Plain SQL files with up/down support |
| Auth | JWT (HMAC-SHA256) | Stateless tokens — no session store needed |
| Hashing | bcrypt (cost 12) | Industry standard — much stronger than SHA-256 |
| Logging | slog (stdlib) | Structured JSON logs, zero external dependencies |
