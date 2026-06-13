const request = require('supertest');
const app = require('../src/server');

describe('Server tests', () => {
  it('should return 200 OK for health check', async () => {
    const res = await request(app).get('/healthz');
    expect(res.statusCode).toEqual(200);
  });
});