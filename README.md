# Task Scheduler (Go + SQLite)

A small web-based task scheduler with recurring tasks support.
The project exposes a JSON API and serves a minimal web UI from the `./web` directory.

## Features

- Create, update and delete tasks
- Mark tasks as done
  - Non-repeating tasks are deleted
  - Repeating tasks are moved to the next occurrence
- Recurring rules: daily (`d`), weekly (`w`), monthly (`m`), yearly (`y`)
- SQLite storage (no external services required)
- Simple password-based authentication with JWT
- Docker / docker-compose support

## Project structure

- `main.go` — application entry point (config + DB init + server start)
- `pkg/server` — HTTP server bootstrap (routes, static files, graceful shutdown)
- `pkg/api` — HTTP handlers (`/api/...`)
- `pkg/db` — database access (SQLite via `database/sql`)
- `web` — static UI (`/login.html`, `/index.html`, assets)

## Configuration

The application reads configuration from environment variables:

- `TODO_PASSWORD` — password for `/login.html` and JWT signing key (default: `12345`)
- `TODO_PORT` — HTTP port (default: `7540`)
- `TODO_DBFILE` — SQLite file path (default: `scheduler.db`)

You can create a `.env` file in the project root:

```env
TODO_PASSWORD=12345
TODO_PORT=7540
TODO_DBFILE=scheduler.db
```

> Note: printing secrets to logs is not recommended in production.

## Run locally

1. Install dependencies:

```bash
go mod tidy
```

2. Start the server:

```bash
go run main.go
```

3. Open the UI:

- `http://localhost:7540/login.html`

## API overview

### Public

- `POST /api/signin` — returns JWT token
- `GET /api/nextdate?now=YYYYMMDD&date=YYYYMMDD&repeat=<rule>` — returns next date as plain text

### Protected (requires token)

- `POST /api/task` — create task
- `GET /api/task?id=<id>` — get task
- `PUT /api/task` — update task
- `DELETE /api/task?id=<id>` — delete task
- `GET /api/tasks?search=<query>` — list tasks (optional search)
- `POST /api/task/done?id=<id>` — mark task as done

## Authentication

1. Request a token:

```bash
curl -s -X POST http://localhost:7540/api/signin \
  -H 'Content-Type: application/json' \
  -d '{"password":"12345"}'
```

2. Use the token:

- Cookie: `token=<JWT>`
- or Header: `Authorization: Bearer <JWT>`

## Tests

Tests are located in `./tests`.

Before running tests, update `tests/settings.go`:

- `Port` — server port
- `DBFile` — path to DB file
- `Token` — a valid JWT token (see Authentication section)

Run:

```bash
go test ./tests
```

## Docker

### Build & run

```bash
docker build -t todo-app .

docker run -it -p 7540:7540 \
  -v /path/to/local/scheduler.db:/app/scheduler.db \
  -e TODO_PASSWORD=12345 \
  -e TODO_PORT=7540 \
  todo-app
```

Open:

- `http://localhost:7540/login.html`

### docker-compose

```bash
docker-compose up --build
```

Then open:

- `http://localhost:7540/login.html`
