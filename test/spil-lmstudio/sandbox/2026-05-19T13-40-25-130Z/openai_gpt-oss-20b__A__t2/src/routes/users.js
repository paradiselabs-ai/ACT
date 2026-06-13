const express = require('express');
const { randomUUID } = require('crypto');

// In‑memory storage scoped to this module
const usersMap = new Map();
const router = express.Router();

// GET /users – list all users
router.get('/', async (req, res) => {
  const users = Array.from(usersMap.values());
  res.json({ users });
});

// GET /users/:id – get a single user
router.get('/:id', async (req, res) => {
  const user = usersMap.get(req.params.id);
  if (!user) return res.status(404).json({ error: 'user not found' });
  res.json(user);
});

// POST /users – create a new user
router.post('/', async (req, res) => {
  const { name, email } = req.body;
  if (!name || !email) {
    return res.status(400).json({ error: 'name and email are required' });
  }
  const id = randomUUID();
  const createdAt = new Date().toISOString();
  const user = { id, name, email, createdAt };
  usersMap.set(id, user);
  res.status(201).json(user);
});

module.exports = router;
