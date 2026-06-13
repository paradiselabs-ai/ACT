const express = require('express');
const healthRouter = express.Router();

healthRouter.get('/', async (req, res) => {
  res.status(200).json({ status: 'ok' });
});

module.exports = healthRouter;
