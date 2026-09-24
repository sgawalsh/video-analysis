const request = require('supertest');
const { createApp } = require('../app');

describe('API basic endpoints', () => {

  beforeAll(() => {
    const pool = {
      query: jest.fn(),
      connect: jest.fn(),
    };

    app = createApp({ pool, startListener: false });
  });

  test('GET /health returns health message', async () => {
    const res = await request(app).get('/health');

    expect(res.statusCode).toBe(200);
    expect(res.text).toBe('API is running');
  });

  test('GET /metrics returns Prometheus metrics', async () => {
    const res = await request(app).get('/metrics');

    expect(res.statusCode).toBe(200);
    expect(res.headers['content-type']).toContain('text/plain');
    expect(res.text).toContain('http_requests_total');
  });

  test('POST /sessions without description returns 400', async () => {
    const res = await request(app)
      .post('/sessions')
      .send({
        mode: "single",
        type: "SEMANTIC_SEARCH",
        searchTerm: "test query",
      }); // no description

    expect(res.statusCode).toBe(400);
    expect(res.body).toEqual({ error: 'Video url is required' });
  });
});