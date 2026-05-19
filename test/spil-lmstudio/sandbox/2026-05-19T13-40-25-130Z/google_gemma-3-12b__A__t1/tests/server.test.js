const request = require('supertest');
const app = require('../src/server');

describe('Server tests', () => {
  it('GET /healthz returns 200 with status ok', async () => {
    const res = await request(app).get('/healthz');
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ status: 'ok' });
  });

  it('GET /users initially returns an empty array', async () => {
    const res = await request(app).get('/users');
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ users: [] });
  });

  it('POST /users with valid body creates user and returns 201', async () => {
    const res = await request(app)
      .post('/users')
      .send({ name: 'Test User', email: 'test@example.com' });
    expect(res.status).toBe(201);
    expect(res.body).toEqual(expect.objectContaining({ name: 'Test User', email: 'test@example.com' }));
  });

  it('POST /users with missing email returns 400', async () => {
    const res = await request(app)
      .post('/users')
      .send({ name: 'Test User' });
    expect(res.status).toBe(400);
    expect(res.body).toEqual({ error: 'name and email are required' });
  });

  it('GET /users/:id with bad id returns 404', async () => {
    const res = await request(app).get('/users/invalid-id');
    expect(res.status).toBe(404);
    expect(res.body).toEqual({ error: 'user not found' });
  });
});