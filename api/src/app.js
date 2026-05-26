const express = require('express');
const jobsRoutes = require('./routes/jobs');
const resultRoutes = require('./routes/resultRoutes');
const {
  metricsMiddleware,
  metricsEndpoint,
} = require('./metrics');

function createApp({ pool }) {
  const app = express();
  app.use(express.json());
  
  app.use(metricsMiddleware);
  app.use('/jobs', jobsRoutes({ pool }));
  app.use('/results', resultRoutes({ pool }));
  app.get('/metrics', metricsEndpoint);

  app.get('/health', (_req, res) => {
    res.send('API is running');
  });

  return app;
}

module.exports = { createApp };