const express = require('express');
const usersRouter = require('./routes/users');
const healthRouter = require('./routes/health');

const app = express();
app.use(express.json());

app.use('/users', usersRouter);
app.use('/healthz', healthRouter);

// catch-all 404 handler
app.use((req, res) => {
  res.status(404).json({ error: 'not found' });
});

if (require.main === module) {
  app.listen(3000, () => {
    console.log('Server is running on http://localhost:3000');
  });
}

module.exports = app;