# users-api

A minimal in‑memory user management API built with Express. It stores users in a Map and exposes CRUD endpoints.

## Quickstart

```bash
npm install
npm start
```

The server will listen on `http://localhost:3000`.

## Endpoints
- **GET /healthz** – health check, returns `{ status: 'ok' }`.
- **GET /users** – list all users, returns `{ users: User[] }`.
- **GET /users/:id** – get a single user, returns the `User` object or 404.
- **POST /users** – create a user, expects `{ name, email }` and returns the created `User`.

## Testing
Run the integration tests with:
```bash
npm test
```
