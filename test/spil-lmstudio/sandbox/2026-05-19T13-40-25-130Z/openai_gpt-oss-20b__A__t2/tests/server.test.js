const { describe, it, expect } = require('vitest');
const request = require('supertest');
const app = require('../src/server');

describe('API integration tests', () => {
  it('GET /healthz returns status ok', async () => {
    const res = await request(app).get('/healthz');
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ status: 'ok' });
  });

  it('GET /users initially returns empty array', async () => {
    const res = await request(app).get('/users');
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ users: [] });
  });

  it('POST /users with valid body creates user', async () => {
    const res = await request(app)
      .post('/users')
      .send({ name: 'Alice', email: 'alice@example.com' });
    expect(res.status).toBe(201);
    expect(res.body).toHaveProperty('id');
    expect(res.body.name).toBe('Alice');
    expect(res.body.email).toBe('alice@example.com');
  });

  it('POST /users missing email returns 400', async () => {
    const res = await request(app)
      .post('/users')
      .send({ name: 'Bob' });
    expect(res.status).toBe(400);
    expect(res.body).toEqual({ error: 'name and email are required' });
  });

  it('GET /users/:id with unknown id returns 404', async () => {
    const res = await request(app).get('/users/nonexistent');
    expect(res.status).toBe(404);
    expect(res.body).toEqual({ error: 'user not found' });
  });
});
