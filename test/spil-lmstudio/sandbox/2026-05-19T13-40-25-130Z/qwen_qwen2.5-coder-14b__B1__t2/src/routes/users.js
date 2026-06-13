const express = require('express');
const crypto = require('crypto');

const usersRouter = express.Router();
const users = new Map();

usersRouter.get('/', async (req, res) => {
  const usersArray = Array.from(users.values());
  res.json({ users: usersArray });
});

usersRouter.get('/:id', async (req, res) => {
  const user = users.get(req.params.id);
  if (user) {
    res.json(user);
  } else {
    res.status(404).json({ error: 'user not found' });
  }
});

usersRouter.post('/', async (req, res) => {
  const { name, email } = req.body;

  if (!name || !email) {
    return res.status(400).json({ error: 'name and email are required' });
  }

  const id = crypto.randomUUID();
  const createdAt = new Date().toISOString();
  const newUser = { id, name, email, createdAt };

  users.set(id, newUser);
  res.status(201).json(newUser);
});

module.exports = usersRouter;