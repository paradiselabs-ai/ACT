const express = require('express');
const usersRouter = require('./routes/users');
const healthRouter = require('./routes/health');

const app = express();
app.use(express.json());

// Health check route
app.get('/healthz', (req, res) => {
  res.status(200).json({ status: 'ok' });
});

// Mount user routes
app.use('/users', usersRouter);

// Mount health routes
app.use('/healthz', healthRouter);

// Catch-all route for unknown routes
app.use((req, res) => {
  res.status(404).json({ error: 'not found' });
});

module.exports = app;
