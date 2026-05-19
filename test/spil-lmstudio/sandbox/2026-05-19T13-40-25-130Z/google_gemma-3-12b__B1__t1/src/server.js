const express = require('express');
const usersRouter = require('./routes/users');
const healthRouter = require('./routes/health');

const app = express();
app.use(express.json());

app.use('/healthz', healthRouter);
app.use('/users', usersRouter);

// Catch-all route for unknown routes
app.use((req, res) => {
  res.status(404).json({ error: 'not found' });
});

module.exports = app;

if (require.main === module) {
  app.listen(3000, () => {
    console.log('Server listening on port 3000');
  });
}