const { describe, it, expect } = require('vitest');
const request = require('supertest');
const app = require('../src/server').default;

describe('Server', () => {
  it('GET /healthz returns 200 with status ok', async () => {
    const response = await request(app).get('/healthz');
    expect(response.status).toBe(200);
    expect(response.body).toEqual({ status: 'ok' });
  });

  it('GET /users initially returns empty array', async () => {
    const response = await request(app).get('/users');
    expect(response.status).toBe(200);
    expect(response.body).toEqual({ users: [] });
  });

  it('POST /users with valid body creates user and returns 201', async () => {
    const response = await request(app).post('/users').send({ name: 'John Doe', email: 'john@example.com' });
    expect(response.status).toBe(201);
    expect(response.body).toHaveProperty('id');
    expect(response.body).toEqual({ id: response.body.id, name: 'John Doe', email: 'john@example.com', createdAt: expect.any(String) });
  });

  it('POST /users with missing email returns 400', async () => {
    const response = await request(app).post('/users').send({ name: 'John Doe' });
    expect(response.status).toBe(400);
    expect(response.body).toEqual({ error: 'name and email are required' });
  });

  it('GET /users/:id with bad id returns 404', async () => {
    const response = await request(app).get('/users/invalid-id');
    expect(response.status).toBe(404);
    expect(response.body).toEqual({ error: 'user not found' });
  });
});