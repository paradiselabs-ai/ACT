const express = require('express');
const crypto = require('crypto');

const usersRouter = express.Router();
const usersMap = new Map();

usersRouter.get('/', async (req, res) => {
  const users = Array.from(usersMap.values());
  res.json({ users });
});

usersRouter.get('/:id', async (req, res) => {
  const user = usersMap.get(req.params.id);
  if (user) {
    res.json(user);
  } else {
    res.status(404).json({ error: 'user not found' });
  }
});

usersRouter.post('/', async (req, res) => {
  const { name, email } = req.body;
  if (!name || !email) {
    res.status(400).json({ error: 'name and email are required' });
  } else {
    const id = crypto.randomUUID();
    const user = { id, name, email, createdAt: new Date().toISOString() };
    usersMap.set(id, user);
    res.status(201).json(user);
  }
});

module.exports = usersRouter;