# users-api

A minimal Express user management API (in-memory).

## Quickstart
npm install
npm start

The API will be available at http://localhost:3000.

## Endpoints
* GET /healthz - Returns the status of the API.
* GET /users - Retrieves a list of all users.
* POST /users - Creates a new user.
* GET /users/:id - Retrieves a specific user by ID.

## Testing
npm test