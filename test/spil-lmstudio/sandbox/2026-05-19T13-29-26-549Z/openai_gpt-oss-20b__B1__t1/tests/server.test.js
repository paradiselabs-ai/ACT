const request = require('supertest');
const { describe, it, expect } = require('vitest');

const app = require('../src/server');

describe('Health endpoint', () => {
  it('GET /healthz returns status ok', async () => {
    const res = await request(app).get('/healthz');
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ status: 'ok' });
  });
});

describe('Users endpoint', () => {
  it('GET /users initially returns empty array', async () => {
    const res = await request(app).get('/users');
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ users: [] });
  });

  it('POST /users with valid body creates user', async () => {
    const payload = { name: 'Alice', email: 'alice@example.com' };
    const res = await request(app).post('/users').send(payload);
    expect(res.status).toBe(201);
    const user = res.body;
    expect(user).toHaveProperty('id');
    expect(user.name).toBe(payload.name);
    expect(user.email).toBe(payload.email);
    expect(user).toHaveProperty('createdAt');
  });

  it('POST /users missing email returns 400', async () => {
    const payload = { name: 'Bob' };
    const res = await request(app).post('/users').send(payload);
    expect(res.status).toBe(400);
    expect(res.body).toEqual({ error: 'name and email are required' });
  });

  it('GET /users/:id with unknown id returns 404', async () => {
    const res = await request(app).get('/users/nonexistent');
    expect(res.status).toBe(404);
    expect(res.body).toEqual({ error: 'user not found' });
  });
});
