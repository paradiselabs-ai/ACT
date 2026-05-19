# users-api

A minimal Express user-management API that stores users in-memory.

## Quickstart

1. `npm install`
2. `npm start`
3. The server will be running on `http://localhost:3000`

## Endpoints

- **GET /healthz**: Returns the health status of the server.
- **GET /users**: Retrieves a list of all users.
- **POST /users**: Creates a new user with the provided name and email.
- **GET /users/:id**: Retrieves a specific user by ID.

## Testing

Run the tests using `npm test`.