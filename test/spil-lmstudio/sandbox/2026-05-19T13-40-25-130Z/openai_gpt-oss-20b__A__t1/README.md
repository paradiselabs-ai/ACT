# users-api

A minimal in‑memory user management API built with Express. It stores users in a simple JavaScript `Map` and exposes CRUD endpoints for future expansion.

## Quickstart

```bash
npm install
npm start
```

The server will listen on `http://localhost:3000` by default.

## Endpoints
- **GET /healthz** – Health check, returns `{ status: 'ok' }`.
- **GET /users** – List all users, returns `{ users: User[] }`.
- **GET /users/:id** – Retrieve a single user by ID, returns the `User` object.
- **POST /users** – Create a new user with `{ name, email }`.

## Testing
Run the integration tests with:
```bash
npm test
```
