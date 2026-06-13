const express = require('express');
const usersRouter = require('./routes/users');
const healthRouter = require('./routes/health');

const app = express();
app.use(express.json());
app.use('/users', usersRouter);
app.use('/healthz', healthRouter);

// catch‑all 404
app.use((req, res) => {
  res.status(404).json({ error: 'not found' });
});

if (require.main === module) {
  const PORT = process.env.PORT || 3000;
  app.listen(PORT, () => {
    console.log(`Server listening on port ${PORT}`);
  });
}

module.exports = app;
