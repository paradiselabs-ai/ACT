const express = require('express');
const usersRouter = require('./routes/users');
const healthRouter = require('./routes/health');

const app = express();
app.use(express.json());

// Mount routers
app.use('/users', usersRouter);
app.use('/healthz', healthRouter);

// Catch-all 404 handler for unknown routes
app.use((req, res) => {
  res.status(404).json({ error: 'not found' });
});

// Start server if this file is run directly
if (require.main === module) {
  const PORT = process.env.PORT || 3000;
  app.listen(PORT, () => {
    console.log(`Server listening on port ${PORT}`);
  });
}

module.exports = app;
