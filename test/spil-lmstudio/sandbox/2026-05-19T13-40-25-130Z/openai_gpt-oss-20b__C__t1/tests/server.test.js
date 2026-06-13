const { describe, it, expect } = require('vitest');
const request = require('supertest');
const app = require('../src/server');

describe('API tests', () => {
  it('GET /healthz returns status ok', async () => {
    const res = await request(app).get('/healthz');
    expect(res.statusCode).toBe(200);
    expect(res.body).toEqual({ status: 'ok' });
  });

  it('GET /users initially empty', async () => {
    const res = await request(app).get('/users');
    expect(res.statusCode).toBe(200);
    expect(res.body).toEqual({ users: [] });
  });

  it('POST /users creates a user', async () => {
    const payload = { name: 'Alice', email: "alice@example.com" };
    const res = await request(app).post('/users').send(payload);
    expect(res.statusCode).toBe(201);
    expect(res.body).toHaveProperty('id');
    expect(res.body.name).toBe(payload.name);
    expect(res.body.email).toBe(payload.email);
    expect(res.body).toHaveProperty('createdAt');
  });

  it('POST /users missing email returns 400', async () => {
    const payload = { name: 'Bob' };
    const res = await request(app).post('/users').send(payload);
    expect(res.statusCode).toBe(400);
    expect(res.body).toEqual({ error: 'name and email are required' });
  });

  it('GET /users/:id unknown returns 404', async () => {
    const res = await request(app).get('/users/nonexistent');
    expect(res.statusCode).toBe(404);
    expect(res.body).toEqual({ error: 'user not found' });
  });
});
