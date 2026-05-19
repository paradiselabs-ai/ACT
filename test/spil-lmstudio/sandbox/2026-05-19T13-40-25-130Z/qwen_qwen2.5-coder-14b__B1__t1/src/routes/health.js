const express = require('express');

const healthRouter = express.Router();

healthRouter.get('/', async (req, res) => {
  res.json({ status: 'ok' });
});

module.exports = healthRouter;