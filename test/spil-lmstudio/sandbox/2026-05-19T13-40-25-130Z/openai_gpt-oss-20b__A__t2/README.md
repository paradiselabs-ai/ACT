# users-api

A minimal in‑memory user management API built with Express. It stores users in a simple JavaScript `Map` and exposes CRUD endpoints for demonstration purposes.

## Quickstart

```bash
npm install
npm start
```

The server will listen on `http://localhost:3000` by default.

## Endpoints
- **GET /healthz** – Health check, returns `{ status: "ok" }`.
- **GET /users** – List all users, returns `{ users: User[] }`.
- **GET /users/:id** – Retrieve a single user by ID, returns the `User` object.
- **POST /users** – Create a new user. Body must contain `{ name, email }`.

## Testing
Run the automated tests with:
```bash
npm test
```
