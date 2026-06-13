const express = require('express');

// Create the Express application
const app = express();

// Middleware to parse JSON bodies
app.use(express.json());

// Import routers
const healthRouter = require('./routes/health');
const usersRouter = require('./routes/users');

// Mount routers at their base paths
app.use('/healthz', healthRouter);
app.use('/users', usersRouter);

// Catch‑all 404 handler for unknown routes
app.use((req, res) => {
  res.status(404).json({ error: 'not found' });
});

// Export the app for testing and start listening if this file is run directly
if (require.main === module) {
  const PORT = process.env.PORT || 3000;
  app.listen(PORT, () => {
    console.log(`Server listening on port ${PORT}`);
  });
}

module.exports = app;
