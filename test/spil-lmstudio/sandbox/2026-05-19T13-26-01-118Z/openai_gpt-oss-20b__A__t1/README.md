# users-api

A minimal in‑memory user management API built with Express. It stores users in a Map and exposes CRUD endpoints for future expansion.

## Quickstart

```bash
npm install
npm start
```
The server runs on `http://localhost:3000`.

## Endpoints
- **GET /healthz** – health check, returns `{ status: 'ok' }`.
- **GET /users** – list all users, returns `{ users: User[] }`.
- **GET /users/:id** – retrieve a single user, 404 if not found.
- **POST /users** – create a new user with `{ name, email }`.

## Testing
Run the test suite with:
```bash
npm test
```
