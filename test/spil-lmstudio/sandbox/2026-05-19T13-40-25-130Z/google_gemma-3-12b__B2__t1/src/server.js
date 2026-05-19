const express = require('express');
const usersRouter = require('./routes/users');
const healthRouter = require('./routes/health');

const app = express();
const port = process.env.PORT || 3000;

app.use(express.json());

app.use('/users', usersRouter);
app.use('/healthz', healthRouter);

app.listen(port, () => {
  console.log(`Server listening on port ${port}`);
});

module.exports = app;