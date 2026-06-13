const request = require('supertest');
const app = require('../src/server');

describe('Server tests', () => {
  it('GET /healthz returns 200 and status ok', async () => {
    const res = await request(app).get('/healthz');
    expect(res.statusCode).toEqual(200);
    expect(res.body).toEqual({ status: 'ok' });
  });
});