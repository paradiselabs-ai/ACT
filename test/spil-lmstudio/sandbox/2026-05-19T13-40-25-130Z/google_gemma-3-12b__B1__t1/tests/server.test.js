const request = require('supertest');
const app = require('../src/server');

describe('Server tests', () => {
  it('GET /healthz returns 200 and status ok', async () => {
    const res = await request(app).get('/healthz');
    expect(res.statusCode).toEqual(200);
    expect(res.body).toEqual({ status: 'ok' });
  });

  it('GET /users initially returns an empty array', async () => {
    const res = await request(app).get('/users');
    expect(res.statusCode).toEqual(200);
    expect(res.body).toEqual({ users: [] });
  });

  it('POST /users with valid body creates a user and returns 201', async () => {
    const res = await request(app)
      .post('/users')
      .send({ name: 'John Doe', email: 'john.doe@example.com' });
    expect(res.statusCode).toEqual(201);
    expect(res.body).toHaveProperty('id');
  });

  it('POST /users with missing email returns 400', async () => {
    const res = await request(app)
      .post('/users')
      .send({ name: 'John Doe' });
    expect(res.statusCode).toEqual(400);
    expect(res.body).toEqual({ error: 'name and email are required' });
  });

  it('GET /users/:id with bad id returns 404', async () => {
    const res = await request(app).get('/users/invalid-id');
    expect(res.statusCode).toEqual(404);
    expect(res.body).toEqual({ error: 'user not found' });
  });
});