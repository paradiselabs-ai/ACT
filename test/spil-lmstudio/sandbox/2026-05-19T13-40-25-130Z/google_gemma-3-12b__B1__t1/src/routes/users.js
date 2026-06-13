const express = require('express');
const router = express.Router();
const crypto = require('crypto');

const users = new Map();

router.get('/', (req, res) => {
  res.json({ users: Array.from(users.values()) });
});

router.get('/:id', (req, res) => {
  const id = req.params.id;
  const user = users.get(id);

  if (!user) {
    return res.status(404).json({ error: 'user not found' });
  }

  res.json(user);
});

router.post('/', (req, res) => {
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

module.exports = router;