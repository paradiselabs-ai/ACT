# users-api
This is a simple user management service API implemented as an in-memory MVP using Node.js and Express.

## Quickstart
1. Run `npm install`
2. Start the server with `npm start`
3. Access the API at `http://localhost:3000`

## Endpoints
- **GET /healthz**: Returns the health status of the API.
- **GET /users**: Retrieves a list of all users.
- **POST /users**: Creates a new user with the provided name and email.
- **GET /users/:id**: Retrieves a specific user by ID.

## Testing
Run the tests with `npm test`