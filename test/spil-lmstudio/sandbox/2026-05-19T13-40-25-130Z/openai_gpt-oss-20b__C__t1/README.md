# users-api

A lightweight in‑memory user management service built with Express. It stores users in a Map during runtime and is intended as an MVP that can later be extended with authentication and a real database.

## Quickstart

```bash
npm install
npm start
```
The server will run on `http://localhost:3000`.

## Endpoints
- **GET /healthz** – liveness check, returns `{ status: 'ok' }`.
- **GET /users** – list all users, returns `{ users: User[] }`.
- **GET /users/:id** – retrieve a single user by ID, returns the `User` object.
- **POST /users** – create a new user with `{ name, email }`, returns the created `User`.

## Testing
Run the automated tests with:
```bash
npm test
```
