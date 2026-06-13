# users-api

This is a minimal user-management REST API built on top of Express. The codebase will be extended later with auth and a database; for now, users are stored in an in-memory Map.

## Quickstart

- `npm install`
- `npm start`

The API will be available at `http://localhost:3000`.

## Endpoints

- **GET /healthz**: Returns 200 with `{ status: 'ok' }`.
- **GET /users**: Returns 200 with `{ users: User[] }`. Empty array if no users created yet.
- **GET /users/:id**: Returns 200 with the User object, or 404 with `{ error: 'user not found' }`.
- **POST /users**: Accepts `{ name, email }`, validates both are non-empty strings, creates a User with generated id and ISO createdAt, returns 201 with the User.

## Testing

Run `npm test` to execute the tests.