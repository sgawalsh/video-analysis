const express = require('express');
const { jobFailures } = require('../metrics');
const YOUTUBE_URL_REGEX = /^(https?:\/\/)?(www\.)?(youtube\.com\/watch\?v=|youtu\.be\/)[A-Za-z0-9_-]+/;

function jobsRoutes({ pool }) {
    const router = express.Router();
    
    const VALID_JOB_TYPES = [
    'SEMANTIC_SEARCH',
    'TOPIC_DETECTION',
  ];

    router.post('/', async (req, res) => {
        const {
            type,
            video_url,
        } = req.body;

        // Validate type
        if (!VALID_JOB_TYPES.includes(type)) {
            return res.status(400).json({
                error: 'Invalid job type: ' + type,
            });
        }

        // Shared validation
        if (!video_url) {
            return res.status(400).json({error: 'Video url is required',});
        }

        if (!YOUTUBE_URL_REGEX.test(video_url)) {
            return res.status(400).json({error: 'Please enter a valid YouTube URL',});
        }

        // Semantic search validation
        if (type === 'SEMANTIC_SEARCH' && !req.body.search_term) {
            return res.status(400).json({error: 'Search term is required',});
        }

        const client = await pool.connect();

        try {
            await client.query('BEGIN');
            const result = await client.query(
                `
                INSERT INTO jobs (
                    type,
                    video_url,
                    payload
                )
                VALUES ($1, $2, $3)
                RETURNING
                public_id,
                type,
                status,
                created_at
                `,
                [
                    type,
                    video_url,
                    req.body,
                ]
            );

            await client.query('COMMIT');
            res.status(201).json(result.rows[0]);
        } catch (err) {
            await client.query('ROLLBACK');
            console.error(err);
            jobFailures.inc();
            res.status(500).json({ error: 'Database error' });
        } finally {
            client.release();
        }
    });

    router.get('/:public_id', async (req, res) => {
        res.set('Cache-Control', 'no-store');
        const { public_id } = req.params;

        try {
            const result = await pool.query(
                `
                SELECT
                public_id,
                type,
                status,
                video_url,
                payload,
                created_at,
                updated_at,
                started_at,
                attempts,
                last_error
                FROM jobs
                WHERE public_id = $1
                `,
                [public_id]
            );

            if (result.rowCount === 0) {
                return res.status(404).json({ error: 'Job not found' });
            }

            res.json(result.rows[0]);
        } catch (err) {
            console.error(err);
            res.status(500).json({ error: 'Database error' });
        }
    });

    return router;
}

module.exports = jobsRoutes;