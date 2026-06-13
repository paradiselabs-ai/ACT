const express = require('express');
const router = express.Router();

// In-memory user store (replace with database later)
const users = new Map();

// GET all users
router.get('/', (req, res) => {
  const userArray = Array.from(users.values());
  res.json(userArray);
});

// GET a specific user by ID
router.get('/:id', (req, res) => {
  const userId = req.params.id;
  const user = users.get(userId);

  if (user) {
    res.json(user);
  } else {
    res.status(404).send('User not found');
  }
});

// POST a new user
router.post('/', (req, res) => {
  const newUser = req.body;
  const userId = Math.random().toString(36).substring(2, 15);
  newUser.id = userId;
  users.set(userId, newUser);
  res.status(201).json(newUser);
});

module.exports = router;