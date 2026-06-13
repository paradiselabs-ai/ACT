const express = require('express');
const { randomUUID } = require('crypto');

const usersRouter = express.Router();
const usersMap = new Map();

// GET /users
usersRouter.get('/', async (req, res) => {
  const users = Array.from(usersMap.values());
  res.json({ users });
});

// GET /users/:id
usersRouter.get('/:id', async (req, res) => {
  const user = usersMap.get(req.params.id);
  if (!user) {
    res.status(404).json({ error: 'user not found' });
    return;
  }
  res.json(user);
});

// POST /users
usersRouter.post('/', async (req, res) => {
  const { name, email } = req.body || {};
  if (!name || !email || typeof name !== 'string' || typeof email !== 'string' || !name.trim() || !email.trim()) {
    res.status(400).json({ error: 'name and email are required' });
    return;
  }
  const id = randomUUID();
  const createdAt = new Date().toISOString();
  const user = { id, name, email, createdAt };
  usersMap.set(id, user);
  res.status(201).json(user);
});

module.exports = usersRouter;
