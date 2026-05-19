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

  if (user) {
    res.json(user);
  } else {
    res.status(404).json({ error: 'user not found' });
  }
});

router.post('/', (req, res) => {
  const { name, email } = req.body;

  if (!name || !email) {
    res.status(400).json({ error: 'name and email are required' });
    return;
  }

  const id = crypto.randomUUID();
  const createdAt = new Date().toISOString();
  const newUser = { id, name, email, createdAt };
  users.set(id, newUser);

  res.status(201).json(newUser);
});

module.exports = router;
