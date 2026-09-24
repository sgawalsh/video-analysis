jest.setTimeout(30000); // allow container startup time
jest.mock('../metrics', () => ({
  metricsMiddleware: jest.fn((req, res, next) => next()),
  metricsEndpoint: jest.fn((req, res) => {
    res.send('');
  }),
  jobFailures: {
    inc: jest.fn(),
  },
}));

const { PostgreSqlContainer } = require('@testcontainers/postgresql');
const { Pool } = require('pg');
const request = require('supertest');
const { createApp } = require('../app');
const { runMigrations } = require('../../migrations/migrate');

let container;
let pool;
let app;

beforeAll(async () => {
  // Start the container
  container = await new PostgreSqlContainer('postgres:18-bookworm')
    .withDatabase('testdb')
    .withUsername('testuser')
    .withPassword('testpass')
    .start();

  // Connect the pool
  pool = new Pool({ connectionString: container.getConnectionUri() });
  app = createApp({ pool, startListener: false });

  await runMigrations(pool);
});

beforeEach(async () => {
  // Clean the table before each test
  await pool.query('TRUNCATE TABLE jobs RESTART IDENTITY CASCADE');
});

afterAll(async () => {
  if (pool) await pool.end();
  if (container) await container.stop();
});

describe('POST /sessions integration', () => {
  test('creates a job successfully', async () => {
    const res = await request(app)
      .post('/sessions')
      .send({
        mode: "single",
        type: "SEMANTIC_SEARCH",
        videoURL: "https://www.youtube.com/watch?v=testURL",
        searchTerm: "test query",
      });

    expect(res.statusCode).toBe(201);
    expect(res.body).toHaveProperty('public_id');

    const dbRes = await pool.query('SELECT * FROM jobs WHERE session_public_id = $1', [res.body.public_id]);
    expect(dbRes.rowCount).toBe(1);
  });

  test('returns 400 if type missing', async () => {
    const res = await request(app).post('/sessions').send({type:"INVALID_TYPE"});
    expect(res.statusCode).toBe(400);
    expect(res.body).toEqual({ error: 'Invalid job type: INVALID_TYPE' });
  });

  test('returns 400 if query is missing for semantic search', async () => {
    const res = await request(app).post('/sessions').send({type:"SEMANTIC_SEARCH"});
    expect(res.statusCode).toBe(400);
    expect(res.body).toEqual({ error: 'Search term is required' });
  });

  test('returns 400 if query is missing for keyword search', async () => {
    const res = await request(app).post('/sessions').send({type:"KEYWORD_SEARCH"});
    expect(res.statusCode).toBe(400);
    expect(res.body).toEqual({ error: 'Search term is required' });
  });
});
