const express = require('express');
const crypto = require('crypto');

const usersRouter = express.Router();
// In‑memory store
const usersMap = new Map();

usersRouter.get('/', async (req, res) => {
  const users = Array.from(usersMap.values());
  res.status(200).json({ users });
});

usersRouter.get('/:id', async (req, res) => {
  const user = usersMap.get(req.params.id);
  if (!user) {
    return res.status(404).json({ error: 'user not found' });
  }
  res.status(200).json(user);
});

usersRouter.post('/', async (req, res) => {
  const { name, email } = req.body || {};
  if (!name || !email || typeof name !== 'string' || typeof email !== 'string' || !name.trim() || !email.trim()) {
    return res.status(400).json({ error: 'name and email are required' });
  }
  const id = crypto.randomUUID();
  const createdAt = new Date().toISOString();
  const user = { id, name, email, createdAt };
  usersMap.set(id, user);
  res.status(201).json(user);
});

module.exports = usersRouter;
