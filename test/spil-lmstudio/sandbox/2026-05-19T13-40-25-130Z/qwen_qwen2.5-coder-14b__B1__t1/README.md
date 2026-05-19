# users-api

A minimal user-management REST API built on Express. This is an in-memory MVP.

## Quickstart

1. `npm install`
2. `npm start`
3. The server will be running on `http://localhost:3000`.

## Endpoints

- **GET /healthz**: Returns the health status of the server.
- **GET /users**: Retrieves a list of all users.
- **POST /users**: Creates a new user with the provided name and email.
- **GET /users/:id**: Retrieves a specific user by ID.

## Testing

To run the tests, execute `npm test`.