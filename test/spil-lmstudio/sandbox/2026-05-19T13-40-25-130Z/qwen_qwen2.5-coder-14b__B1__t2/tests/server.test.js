const request = require('supertest');
const app = require('../src/server');

describe('Server', () => {
  it('should return 200 from /healthz', async () => {
    const response = await request(app).get('/healthz');
    expect(response.status).toBe(200);
    expect(response.body).toEqual({ status: 'ok' });
  });

  it('should return an empty array from /users', async () => {
    const response = await request(app).get('/users');
    expect(response.status).toBe(200);
    expect(response.body).toEqual({ users: [] });
  });

  it('should create a user and return 201', async () => {
    const response = await request(app)
      .post('/users')
      .send({ name: 'John Doe', email: 'john@example.com' });
    expect(response.status).toBe(201);
    expect(response.body).toHaveProperty('id');
    expect(response.body.name).toBe('John Doe');
    expect(response.body.email).toBe('john@example.com');
  });

  it('should return 400 for missing name or email', async () => {
    const response = await request(app)
      .post('/users')
      .send({ name: '' });
    expect(response.status).toBe(400);
    expect(response.body).toEqual({ error: 'name and email are required' });
  });

  it('should return 404 for non-existent user', async () => {
    const response = await request(app).get('/users/123');
    expect(response.status).toBe(404);
    expect(response.body).toEqual({ error: 'user not found' });
  });
});