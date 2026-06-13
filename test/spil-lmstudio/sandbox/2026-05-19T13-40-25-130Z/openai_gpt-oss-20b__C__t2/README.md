# users-api

This is a simple in‑memory user management API built with Express. It stores users in a Map during runtime and is intended as an MVP that can later be extended with authentication and a real database.

## Quickstart

```bash
npm install
npm start
```

The server will be listening on `http://localhost:3000`.

## Endpoints

- **GET** `/healthz` – Health check, returns `{ status: "ok" }`.
- **GET** `/users` – List all users, returns `{ users: [] }.
- **POST** `/users` – Create a new user with `name` and `email`.
- **GET** `/users/:id` – Retrieve a user by ID.

## Testing

Run the test suite with:
```bash
npm test
```
