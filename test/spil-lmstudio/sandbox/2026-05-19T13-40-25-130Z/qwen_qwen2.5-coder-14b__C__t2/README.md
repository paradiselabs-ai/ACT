# users-api

This is a simple user management REST API using Express.js. It's an in-memory MVP that will be extended with authentication and a real database later.

## Quickstart

1. `npm install`
2. `npm start`
3. Access the API at `http://localhost:3000`

## Endpoints

- **GET /users**: Returns a list of all users.
- **GET /users/:id**: Retrieves a user by ID.
- **POST /users**: Creates a new user with the provided name and email.
- **GET /healthz**: Returns the health status of the API.

## Testing

Run `npm test` to execute the tests using vitest and supertest.