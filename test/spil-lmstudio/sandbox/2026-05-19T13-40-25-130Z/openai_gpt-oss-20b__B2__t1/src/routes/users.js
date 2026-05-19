const express = require('express');
const { randomUUID } = require('crypto');

const usersRouter = express.Router();
// In‑memory store
const usersMap = new Map();

function validateUser(body) {
  const { name, email } = body || {};
  if (!name?.trim() || !email?.trim()) {
    return false;
  }
  return true;
}

usersRouter.get('/', async (req, res) => {
  const users = Array.from(usersMap.values());
  res.json({ users });
});

usersRouter.get('/:id', async (req, res) => {
  const user = usersMap.get(req.params.id);
  if (!user) {
    return res.status(404).json({ error: 'user not found' });
  }
  res.json(user);
});

usersRouter.post('/', async (req, res) => {
  if (!validateUser(req.body)) {
    return res.status(400).json({ error: 'name and email are required' });
  }
  const id = randomUUID();
  const createdAt = new Date().toISOString();
  const user = { id, name: req.body.name.trim(), email: req.body.email.trim(), createdAt };
  usersMap.set(id, user);
  res.status(201).json(user);
});

module.exports = usersRouter;
