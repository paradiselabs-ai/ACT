const express = require('express');

const healthRouter = express.Router();

// Health check endpoint
healthRouter.get('/', async (req, res) => {
  res.status(200).json({ status: 'ok' });
});

module.exports = healthRouter;
