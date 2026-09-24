const express = require('express');
const sessionRoutes = require('./routes/sessionRoutes');
const createPgListener = require('./pgListener');
const EventHub = require('./eventHub');

const {
  metricsMiddleware,
  metricsEndpoint,
} = require('./metrics');

function createApp({ pool, startListener = true }) {
  const app = express();
  app.use(express.json());
  
  app.use(metricsMiddleware);
  app.get('/metrics', metricsEndpoint);

  app.get('/health', (_req, res) => {
    res.send('API is running');
  });

  const hub = new EventHub();

  app.use('/sessions', sessionRoutes({ pool, hub }));
  if (startListener) {
    createPgListener({ pool, hub })
      .then(() => console.log('PG listener started'))
      .catch(console.error);
  }

  return app;
}

module.exports = { createApp };