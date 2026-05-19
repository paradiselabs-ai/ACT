const express = require('express');
const { randomUUID } = require('crypto');

const usersRouter = express.Router();
// In-memory storage: Map of id -> user object
const usersMap = new Map();

// Helper to validate request body
function validateUserBody(body) {
  const { name, email } = body || {};
  if (!name || !email) {
    return false;
  }
  if (typeof name !== 'string' || typeof email !== 'string') {
    return false;
  }
  if (name.trim() === '' || email.trim() === '') {
    return false;
  }
  return true;
}

// GET /users - list all users
usersRouter.get('/', async (req, res) => {
  const users = Array.from(usersMap.values());
  res.status(200).json({ users });
});

// GET /users/:id - get specific user
usersRouter.get('/:id', async (req, res) => {
  const { id } = req.params;
  if (!usersMap.has(id)) {
    return res.status(404).json({ error: 'user not found' });
  }
  const user = usersMap.get(id);
  res.status(200).json(user);
});

// POST /users - create user
usersRouter.post('/', async (req, res) => {
  if (!validateUserBody(req.body)) {
    return res.status(400).json({ error: 'name and email are required' });
  }
  const { name, email } = req.body;
  const id = randomUUID();
  const createdAt = new Date().toISOString();
  const user = { id, name, email, createdAt };
  usersMap.set(id, user);
  res.status(201).json(user);
});

module.exports = usersRouter;
