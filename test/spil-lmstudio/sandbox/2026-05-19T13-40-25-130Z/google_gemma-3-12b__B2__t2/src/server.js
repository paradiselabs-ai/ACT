const express = require('express');
const usersRouter = require('./routes/users');
const healthzRouter = require('./routes/healthz');

const app = express();
const port = process.env.PORT || 3000;

app.use('/users', usersRouter);
app.use('/healthz', healthzRouter);

app.listen(port, () => {
  console.log(`Server listening on port ${port}`);
});

module.exports = app;