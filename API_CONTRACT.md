# TaskFlow API Contract

Base URL: `http://localhost:8080/api`

All protected endpoints require the header:
```
Authorization: Bearer <token>
```

---

## Authentication

### POST /auth/register

Create a new account.

**Request:**
```json
{
  "name": "Raju Kumar",
  "email": "raju@example.com",
  "password": "password123"
}
```

**Response (201):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "Raju Kumar",
    "email": "raju@example.com",
    "created_at": "2024-01-15T10:00:00Z"
  }
}
```

**Errors:**
- `400` — Validation errors (missing fields, email format, short password)
- `409` — Email already registered

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Test User","email":"test@example.com","password":"password123"}'
```

---

### POST /auth/login

Authenticate and get a JWT token.

**Request:**
```json
{
  "email": "raju@example.com",
  "password": "password123"
}
```

**Response (200):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "Raju Kumar",
    "email": "raju@example.com",
    "created_at": "2024-01-15T10:00:00Z"
  }
}
```

**Errors:**
- `400` — Missing email or password
- `401` — Invalid credentials

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"raju@example.com","password":"password123"}'
```

---

## Projects

### GET /projects

List projects accessible to the authenticated user.

**Query Parameters:**
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| page  | int  | 1       | Page number |
| limit | int  | 20      | Items per page (max 100) |

**Response (200):**
```json
{
  "data": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "name": "Website Redesign",
      "description": "Complete overhaul of the company website",
      "owner_id": "550e8400-e29b-41d4-a716-446655440001",
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    }
  ],
  "page": 1,
  "limit": 20,
  "total": 3
}
```

```bash
TOKEN="your-jwt-token"
curl http://localhost:8080/api/projects \
  -H "Authorization: Bearer $TOKEN"
```

---

### POST /projects

Create a new project. The authenticated user becomes the owner.

**Request:**
```json
{
  "name": "New Project",
  "description": "Optional description"
}
```

**Response (201):**
```json
{
  "id": "...",
  "name": "New Project",
  "description": "Optional description",
  "owner_id": "550e8400-...",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

**Errors:**
- `400` — Name is required

```bash
curl -X POST http://localhost:8080/api/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"My Project","description":"A cool project"}'
```

---

### GET /projects/:id

Get a single project with its tasks.

**Response (200):**
```json
{
  "id": "660e8400-...",
  "name": "Website Redesign",
  "description": "Complete overhaul",
  "owner_id": "550e8400-...",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z",
  "tasks": [
    {
      "id": "770e8400-...",
      "title": "Create wireframes",
      "status": "in_progress",
      "priority": "high",
      ...
    }
  ]
}
```

**Errors:**
- `404` — Project not found

```bash
curl http://localhost:8080/api/projects/660e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer $TOKEN"
```

---

### PATCH /projects/:id

Update a project. Only the owner can update.

**Request:**
```json
{
  "name": "Updated Name",
  "description": "Updated description"
}
```

**Errors:**
- `403` — Not the project owner
- `404` — Project not found

```bash
curl -X PATCH http://localhost:8080/api/projects/660e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Name"}'
```

---

### DELETE /projects/:id

Delete a project and all its tasks. Only the owner can delete.

**Response (204):** No content

**Errors:**
- `403` — Not the project owner
- `404` — Project not found

```bash
curl -X DELETE http://localhost:8080/api/projects/660e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer $TOKEN"
```

---

### GET /projects/:id/stats

Get task statistics for a project.

**Response (200):**
```json
{
  "total": 10,
  "by_status": {
    "todo": 4,
    "in_progress": 3,
    "done": 3
  },
  "by_assignee": [
    {
      "user_id": "550e8400-...",
      "name": "Raju Kumar",
      "count": 4
    },
    {
      "user_id": "550e8400-...",
      "name": "Priya Sharma",
      "count": 3
    }
  ]
}
```

```bash
curl http://localhost:8080/api/projects/660e8400-e29b-41d4-a716-446655440001/stats \
  -H "Authorization: Bearer $TOKEN"
```

---

## Tasks

### GET /projects/:id/tasks

List tasks for a project with optional filtering.

**Query Parameters:**
| Param     | Type   | Description                        |
|-----------|--------|------------------------------------|
| status    | string | Filter by status: todo, in_progress, done |
| assignee  | string | Filter by assignee user ID         |
| page      | int    | Page number (default: 1)           |
| limit     | int    | Items per page (default: 20)       |

**Response (200):**
```json
{
  "data": [
    {
      "id": "770e8400-...",
      "title": "Create wireframes",
      "description": "Design wireframes for all main pages",
      "status": "in_progress",
      "priority": "high",
      "due_date": "2024-02-15T00:00:00Z",
      "project_id": "660e8400-...",
      "assignee_id": "550e8400-...",
      "created_by": "550e8400-...",
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-16T10:00:00Z"
    }
  ],
  "page": 1,
  "limit": 20,
  "total": 5
}
```

```bash
curl "http://localhost:8080/api/projects/660e8400-.../tasks?status=todo&page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

---

### POST /projects/:id/tasks

Create a task within a project.

**Request:**
```json
{
  "title": "Write unit tests",
  "description": "Cover all service layer functions",
  "status": "todo",
  "priority": "high",
  "due_date": "2024-03-01",
  "assignee_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

**Response (201):**
Returns the created task object.

**Errors:**
- `400` — Title is required, invalid status/priority
- `404` — Project not found

```bash
curl -X POST http://localhost:8080/api/projects/660e8400-.../tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"New task","priority":"medium","status":"todo"}'
```

---

### PATCH /tasks/:id

Update a task. Any authenticated user can update tasks.

**Request (partial updates supported):**
```json
{
  "status": "done",
  "priority": "low"
}
```

**Response (200):**
Returns the updated task object.

```bash
curl -X PATCH http://localhost:8080/api/tasks/770e8400-... \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"done"}'
```

---

### DELETE /tasks/:id

Delete a task. Allowed by: project owner OR the user who created the task.

**Response (204):** No content

**Errors:**
- `403` — Not authorized to delete
- `404` — Task not found

```bash
curl -X DELETE http://localhost:8080/api/tasks/770e8400-... \
  -H "Authorization: Bearer $TOKEN"
```

---

## Error Format

All errors follow a consistent format:

```json
{
  "error": "human-readable error message"
}
```

Validation errors return field-level details:

```json
{
  "error": "validation failed",
  "fields": {
    "email": "valid email is required",
    "password": "must be at least 6 characters"
  }
}
```

## Health Check

```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```
