# users-api

A minimal user-management REST API built with Express. This is an in-memory MVP.

## Quickstart

1. `npm install`
2. `npm start`
3. Access the API at `http://localhost:3000`

## Endpoints

- **GET /healthz**: Returns the health status of the API.
- **GET /users**: Retrieves a list of all users.
- **POST /users**: Creates a new user with the provided name and email.
- **GET /users/:id**: Retrieves a specific user by ID.

## Testing

Run the tests using `npm test`.