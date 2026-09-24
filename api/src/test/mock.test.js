const request = require('supertest');

jest.mock('../metrics', () => ({
  metricsMiddleware: jest.fn((req, res, next) => next()),
  metricsEndpoint: jest.fn((req, res) => {
    res.send('');
  }),
  jobFailures: {
    inc: jest.fn(),
  },
}));

jest.mock('pg', () => {
  const mClient = {
    query: jest.fn(),
    release: jest.fn(),
  };
  const mPool = {
    connect: jest.fn(() => Promise.resolve(mClient)),
    query: jest.fn(),
  };
  return { Pool: jest.fn(() => mPool) };
});

const { createApp } = require('../app');
const { Pool } = require('pg');

describe('Job API DB integration (mocked)', () => {
  let pool;
  let client;

  beforeEach(() => {
    pool = new Pool();
    client = {
      query: jest.fn(),
      release: jest.fn(),
    };
    pool.connect.mockResolvedValue(client);
    app = createApp({ pool, startListener: false });
  });

  test('POST /sessions with description inserts job and returns 201', async () => {
    const fakeSession = {
    id: 1,
    public_id: '12345678-1234-1234-1234-123456789abc',
  };

    // Simulate the DB returning the inserted job
    client.query.mockImplementation((sql, params) => {
      if (sql.startsWith('BEGIN') || sql.startsWith('COMMIT')) {
        return Promise.resolve();
      }
      else if (sql.includes('INSERT INTO sessions')) {
        return Promise.resolve({ rows: [fakeSession] });
      }
      else if (sql.includes('INSERT INTO jobs')) {
        return Promise.resolve({ rows: [] });
      }
    });

    const res = await request(app)
      .post('/sessions')
      .send({
        mode: "single",
        type: "SEMANTIC_SEARCH",
        videoURL: "https://www.youtube.com/watch?v=testURL",
        searchTerm: "test query",
      });

    expect(res.statusCode).toBe(201);
    expect(res.body).toEqual({
      public_id: fakeSession.public_id,
    });

    // Check that BEGIN, INSERT, COMMIT were called
    expect(client.query).toHaveBeenCalledWith('BEGIN');
    // expect(client.query).toHaveBeenCalledWith(
    //   'INSERT INTO jobs (description) VALUES($1) RETURNING public_id, description',
    //   ['Test job']
    // );
    expect(client.query).toHaveBeenCalledWith(expect.stringContaining('INSERT INTO sessions'), expect.any(Array));
    expect(client.query).toHaveBeenCalledWith(expect.stringContaining('INSERT INTO jobs'), expect.any(Array));
    expect(client.query).toHaveBeenCalledWith('COMMIT');
    expect(client.release).toHaveBeenCalled();
  });

  test('POST /sessions DB failure triggers rollback and 500', async () => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    const error = new Error('DB error');

    client.query.mockImplementation((sql) => {
      if (sql.startsWith('BEGIN')) return Promise.resolve();
      if (sql.includes('INSERT INTO sessions')) throw error;
      if (sql.startsWith('ROLLBACK')) return Promise.resolve();
    });

    const res = await request(app)
      .post('/sessions')
      .send({
        mode: "single",
        type: "SEMANTIC_SEARCH",
        videoURL: "https://www.youtube.com/watch?v=testURL",
        searchTerm: "fail job",
      });

    expect(res.statusCode).toBe(500);
    expect(res.body).toEqual({ error: 'Database error' });
    expect(client.query).toHaveBeenCalledWith('ROLLBACK');
    expect(client.release).toHaveBeenCalled();
    console.error.mockRestore();
  });
});
