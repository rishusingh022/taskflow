# Backend Testing Guide

How to test the backend **by itself** — no frontend required. Just the API, a terminal, and `curl`.

---

## Prerequisites

Make sure the backend is running:

```bash
# Option A: Docker (recommended — includes PostgreSQL)
docker-compose up --build

# Option B: Local (requires local PostgreSQL)
cd backend
DATABASE_URL="postgres://user:pass@localhost:5432/taskflow?sslmode=disable" \
JWT_SECRET="test-secret-key" \
go run cmd/server/main.go
```

The API is available at `http://localhost:8080/api`.

Throughout this guide, we'll save values in shell variables (like `$TOKEN`) so you can copy-paste the commands in sequence.

---

## Happy Path — Full User Flow

### 1. Register a New User

```bash
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Raju Kumar",
    "email": "raju@example.com",
    "password": "password123"
  }' | jq .
```

**Expected response (201):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Raju Kumar",
    "email": "raju@example.com",
    "created_at": "2024-03-09T10:00:00Z"
  }
}
```

Save the token:
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Raju Kumar","email":"raju@example.com","password":"password123"}' \
  | jq -r '.token')

echo $TOKEN
```

### 2. Login

```bash
curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "raju@example.com",
    "password": "password123"
  }' | jq .
```

**Expected (200):** Same shape as register — returns `token` and `user`.

### 3. Create a Project

```bash
PROJECT_ID=$(curl -s -X POST http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"TaskFlow MVP","description":"First version of the app"}' \
  | jq -r '.id')

echo "Created project: $PROJECT_ID"
```

**Expected (201):**
```json
{
  "id": "...",
  "name": "TaskFlow MVP",
  "description": "First version of the app",
  "owner_id": "...",
  "created_at": "..."
}
```

### 4. List Your Projects

```bash
curl -s http://localhost:8080/api/projects \
  -H "Authorization: Bearer $TOKEN" | jq .
```

**Expected (200):**
```json
{
  "data": [{ "id": "...", "name": "TaskFlow MVP", ... }],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

### 5. Create Tasks

```bash
# Task 1: Todo
TASK1=$(curl -s -X POST http://localhost:8080/api/projects/$PROJECT_ID/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Set up database","description":"Configure PostgreSQL","priority":"high"}' \
  | jq -r '.id')

# Task 2: In Progress
TASK2=$(curl -s -X POST http://localhost:8080/api/projects/$PROJECT_ID/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Build API endpoints","status":"in_progress","priority":"high"}' \
  | jq -r '.id')

# Task 3: Low priority
curl -s -X POST http://localhost:8080/api/projects/$PROJECT_ID/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Write documentation","priority":"low"}' | jq .

echo "Task 1: $TASK1"
echo "Task 2: $TASK2"
```

### 6. List Tasks (with Filters)

```bash
# All tasks in the project
curl -s "http://localhost:8080/api/projects/$PROJECT_ID/tasks" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Filter by status
curl -s "http://localhost:8080/api/projects/$PROJECT_ID/tasks?status=todo" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Filter by priority
curl -s "http://localhost:8080/api/projects/$PROJECT_ID/tasks?priority=high" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Pagination
curl -s "http://localhost:8080/api/projects/$PROJECT_ID/tasks?page=1&limit=2" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 7. Update a Task

```bash
curl -s -X PATCH http://localhost:8080/api/tasks/$TASK1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"status":"done"}' | jq .
```

**Expected (200):** The updated task with `"status": "done"`.

### 8. Update a Project

```bash
curl -s -X PATCH http://localhost:8080/api/projects/$PROJECT_ID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"TaskFlow v2","description":"Improved version"}' | jq .
```

### 9. Get Project Stats

```bash
curl -s http://localhost:8080/api/projects/$PROJECT_ID/stats \
  -H "Authorization: Bearer $TOKEN" | jq .
```

**Expected (200):**
```json
{
  "total_tasks": 3,
  "by_status": { "todo": 1, "in_progress": 1, "done": 1 },
  "by_assignee": {}
}
```

### 10. Delete a Task

```bash
curl -s -X DELETE http://localhost:8080/api/tasks/$TASK2 \
  -H "Authorization: Bearer $TOKEN" -w "\nHTTP Status: %{http_code}\n"
```

**Expected:** HTTP 204 (No Content) — empty body.

### 11. Delete a Project (Cascades to Tasks)

```bash
curl -s -X DELETE http://localhost:8080/api/projects/$PROJECT_ID \
  -H "Authorization: Bearer $TOKEN" -w "\nHTTP Status: %{http_code}\n"
```

**Expected:** HTTP 204. All tasks in this project are also deleted.

---

## Unhappy Paths — Error Scenarios

### Authentication Errors

```bash
# No token at all
curl -s http://localhost:8080/api/projects | jq .
# → 401 {"error": "authorization header required"}

# Invalid token
curl -s http://localhost:8080/api/projects \
  -H "Authorization: Bearer invalid-token" | jq .
# → 401 {"error": "invalid token"}

# Malformed header (missing "Bearer" prefix)
curl -s http://localhost:8080/api/projects \
  -H "Authorization: just-a-token" | jq .
# → 401 {"error": "invalid authorization header format"}
```

### Registration Errors

```bash
# Missing required fields
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{}' | jq .
# → 400 {"error":"validation failed","fields":{"email":"...","name":"...","password":"..."}}

# Password too short
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","email":"test@test.com","password":"123"}' | jq .
# → 400 {"error":"validation failed","fields":{"password":"must be at least 6 characters"}}

# Invalid email format
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","email":"not-an-email","password":"password123"}' | jq .
# → 400 {"error":"validation failed","fields":{"email":"must be a valid email"}}

# Duplicate email (register twice)
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","email":"duplicate@test.com","password":"password123"}' > /dev/null

curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Test2","email":"duplicate@test.com","password":"password456"}' | jq .
# → 409 {"error": "already exists"}
```

### Login Errors

```bash
# Wrong password
curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"raju@example.com","password":"wrongpassword"}' | jq .
# → 401 {"error": "unauthorized"}

# Non-existent email
curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"nobody@example.com","password":"password123"}' | jq .
# → 401 {"error": "unauthorized"}
# (same error — intentionally doesn't reveal whether the email exists)
```

### Authorization Errors (Forbidden)

```bash
# Register a second user
TOKEN2=$(curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Other User","email":"other@example.com","password":"password123"}' \
  | jq -r '.token')

# Create a project as User 1
PROJECT_ID=$(curl -s -X POST http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Private Project"}' \
  | jq -r '.id')

# Try to update it as User 2
curl -s -X PATCH http://localhost:8080/api/projects/$PROJECT_ID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN2" \
  -d '{"name":"Hacked!"}' | jq .
# → 403 {"error": "forbidden"}

# Try to delete it as User 2
curl -s -X DELETE http://localhost:8080/api/projects/$PROJECT_ID \
  -H "Authorization: Bearer $TOKEN2" -w "\nHTTP Status: %{http_code}\n"
# → 403 {"error": "forbidden"}
```

### Not Found Errors

```bash
# Non-existent project
curl -s http://localhost:8080/api/projects/00000000-0000-0000-0000-000000000000 \
  -H "Authorization: Bearer $TOKEN" | jq .
# → 404 {"error": "not found"}

# Non-existent task
curl -s -X DELETE http://localhost:8080/api/tasks/00000000-0000-0000-0000-000000000000 \
  -H "Authorization: Bearer $TOKEN" -w "\nHTTP Status: %{http_code}\n"
# → 404 {"error": "not found"}
```

### Validation Errors (Projects & Tasks)

```bash
# Create project with empty name
curl -s -X POST http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":""}' | jq .
# → 400 {"error":"validation failed","fields":{"name":"is required"}}

# Create task with invalid status
curl -s -X POST http://localhost:8080/api/projects/$PROJECT_ID/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Test","status":"invalid_status"}' | jq .
# → 400 {"error":"validation failed","fields":{"status":"must be todo, in_progress, or done"}}

# Create task with invalid priority
curl -s -X POST http://localhost:8080/api/projects/$PROJECT_ID/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Test","priority":"urgent"}' | jq .
# → 400 {"error":"validation failed","fields":{"priority":"must be low, medium, or high"}}

# Create task with empty title
curl -s -X POST http://localhost:8080/api/projects/$PROJECT_ID/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":""}' | jq .
# → 400 {"error":"validation failed","fields":{"title":"is required"}}

# Malformed JSON body
curl -s -X POST http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d 'not json' | jq .
# → 400 {"error": "invalid request body"}
```

---

## Running Unit Tests (No Database Required)

```bash
cd backend
go test ./... -v
```

This runs all tests across all packages. No Docker, no database needed — all tests use mock repositories.

To run tests for a specific package:

```bash
go test ./internal/service/... -v     # service layer tests only
go test ./internal/handler/... -v     # handler tests only
go test ./internal/middleware/... -v   # middleware tests only
go test ./internal/model/... -v       # validation tests only
```

To see test coverage:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out       # opens in browser
go tool cover -func=coverage.out       # prints per-function coverage
```

---

## Error Response Cheat Sheet

| HTTP Code | Meaning | When |
|-----------|---------|------|
| 200 | OK | Successful read or update |
| 201 | Created | Successful create (register, new project, new task) |
| 204 | No Content | Successful delete |
| 400 | Bad Request | Invalid JSON, failed validation |
| 401 | Unauthorized | Missing/invalid/expired token, wrong password |
| 403 | Forbidden | Valid token but not the owner |
| 404 | Not Found | ID doesn't exist in database |
| 409 | Conflict | Email already registered |
| 500 | Internal Error | Unexpected server error |
