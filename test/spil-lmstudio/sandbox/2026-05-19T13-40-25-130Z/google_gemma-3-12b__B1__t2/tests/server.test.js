const request = require('supertest');
const app = require('../src/server');

describe('Server tests', () => {
  it('should return 200 for /healthz', async () => {
    const res = await request(app).get('/healthz');
    expect(res.statusCode).toEqual(200);
    expect(res.body).toEqual({ status: 'ok' });
  });

  it('should return an empty array for /users', async () => {
    const res = await request(app).get('/users');
    expect(res.statusCode).toEqual(200);
    expect(res.body.users).toEqual([]);
  });

  it('should create a user and return 201 on POST /users', async () => {
    const res = await request(app).post('/users').send({ name: 'Test User', email: 'test@example.com' });
    expect(res.statusCode).toEqual(201);
    expect(res.body).toHaveProperty('id');
  });

  it('should return 400 on POST /users with missing email', async () => {
    const res = await request(app).post('/users').send({ name: 'Test User' });
    expect(res.statusCode).toEqual(400);
  });

  it('should return 404 on GET /users/:id with bad id', async () => {
    const res = await request(app).get('/users/invalid-id');
    expect(res.statusCode).toEqual(404);
  });
});