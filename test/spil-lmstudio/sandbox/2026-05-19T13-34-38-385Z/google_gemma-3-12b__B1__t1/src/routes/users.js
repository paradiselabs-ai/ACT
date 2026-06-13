const express = require('express');
const router = express.Router();
const { randomUUID } = require('crypto');
const users = new Map();

// POST /users
router.post('/', (req, res) => {
  const { name, email } = req.body;

  if (!name || !email) {
    return res.status(400).json({ error: 'name and email are required' });
  }

  const id = randomUUID();
  const createdAt = new Date().toISOString();

  users.set(id, { id, name, email, createdAt });

  res.status(201).json({ id, name, email, createdAt });
});

// GET /users
router.get('/', (req, res) => {
  const userArray = Array.from(users.values());
  res.json({ users: userArray });
});

// GET /users/:id
router.get('/:id', (req, res) => {
  const id = req.params.id;
  const user = users.get(id);

  if (!user) {
    return res.status(404).json({ error: 'user not found' });
  }

  res.json(user);
});

module.exports = router;
