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
  it('GET /users initially empty', async () => {
    const res = await request(app).get('/users');
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ users: [] });
  });

  it('POST /users creates user', async () => {
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
    const res = await request(app).post('/users').send({ name: 'Bob' });
    expect(res.status).toBe(400);
    expect(res.body).toEqual({ error: 'name and email are required' });
  });

  it('GET /users/:id returns user or 404', async () => {
    // create a user first
    const payload = { name: 'Carol', email: 'carol@example.com' };
    const createRes = await request(app).post('/users').send(payload);
    const id = createRes.body.id;

    const getRes = await request(app).get(`/users/${id}`);
    expect(getRes.status).toBe(200);
    expect(getRes.body.id).toBe(id);

    const badRes = await request(app).get('/users/nonexistent');
    expect(badRes.status).toBe(404);
    expect(badRes.body).toEqual({ error: 'user not found' });
  });
});
