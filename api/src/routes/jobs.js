const express = require('express');
const { jobFailures } = require('../metrics');

function jobsRoutes({ pool }) {
    const router = express.Router();
    
    const VALID_JOB_TYPES = [
    'SEMANTIC_SEARCH',
    'TOPIC_DETECTION',
  ];

    router.post('/', async (req, res) => {
        const {
            mode,
            type,
            videoURL,
            channelName,
            startDate,
            endDate,
            searchTerm,
        } = req.body;

        // Validate type
        if (!VALID_JOB_TYPES.includes(type)) {
            return res.status(400).json({error: 'Invalid job type: ' + type,});
        }
        // Semantic search validation
        if (type === 'SEMANTIC_SEARCH' && !searchTerm) {
            return res.status(400).json({error: 'Search term is required',});
        }

        switch (mode){
            case 'single':
                singleVideoFlow(res, pool, type, videoURL, searchTerm);
                break;
            case 'channel':
                channelSearchFlow(res, pool, type, channelName, searchTerm, startDate, endDate);
                break;
            default:
                return res.status(400).json({error: 'Invalid mode',});
        }

    });

    router.get('/:public_id', async (req, res) => {
        res.set('Cache-Control', 'no-store');

        const { public_id } = req.params;

        try {
            const sessionType = await getSessionType(pool, public_id);

            const counts = await getSessionJobCounts(
                pool,
                public_id
            );

            res.json({
                type: sessionType,
                counts: counts,
                results: []
            });

        } catch (err) {
            if (err.message === 'Session not found') {
                return res.status(404).json({
                    error: 'Session not found'
                });
            }

            console.error(err);

            res.status(500).json({
                error: 'Database error'
            });
        }
    });

    return router;
}

async function getSessionJobCounts(pool, public_id){
    const result = await pool.query(
        `
        SELECT
            COUNT(*) FILTER (WHERE status = 'SUCCEEDED') AS succeeded,
            COUNT(*) FILTER (WHERE status = 'FAILED') AS failed,
            COUNT(*) FILTER (WHERE status = 'PENDING') AS pending,
            COUNT(*) FILTER (WHERE status = 'RUNNING') AS running,
            COUNT(*) AS total
        FROM jobs j
        JOIN sessions s ON s.id = j.session_id
        WHERE s.public_id = $1
        `,
        [public_id]
    );

    return result.rows[0];
}

async function getSessionType(pool, public_id){
    const result = await pool.query(
        `
        SELECT type FROM sessions WHERE public_id = $1
        `,
        [public_id]
    );
    if (result.rowCount === 0) {
        throw new Error("Session not found");
    }
    return result.rows[0].type;
}

async function singleVideoFlow(res, pool, type, videoURL, searchTerm) {
    if (!videoURL) {
        return res.status(400).json({error: 'Video url is required',});
    }
    const videoID = extractYoutubeVideoId(videoURL);
    if (!videoID) {
        return res.status(400).json({error: 'Please enter a valid YouTube URL',});
    }
    const client = await pool.connect();

    try {
        await client.query('BEGIN');

        const sessionResult = await client.query(
            `
            INSERT INTO sessions (type)
            VALUES ($1)
            RETURNING id, public_id
            `,
            [type]
        );

        const { id, public_id } = sessionResult.rows[0];

        await client.query(
            `
            INSERT INTO jobs (
                session_id,
                type,
                target_id,
                query
            )
            VALUES ($1, $2, $3, $4)
            `,
            [id, type, videoID, searchTerm]
        );

        await client.query('COMMIT');

        res.status(201).json({ public_id });

        console.log(`Created job for video ${videoID} with session ID ${id}`);

    } catch (err) {
        await client.query('ROLLBACK');
        console.error(err);
        jobFailures.inc();

        if (!res.headersSent) {
            res.status(500).json({ error: 'Database error' });
        }
    } finally {
        client.release();
    }
}

async function channelSearchFlow(res, pool, type, channelName, searchTerm, startDate, endDate) {
    if (!channelName) {
        return res.status(400).json({error: 'Channel name is required',});
    }
    if (!startDate || !endDate) {
        return res.status(400).json({error: 'Date fields are required',});
    }
    if (startDate > endDate){
        return res.status(400).json({error: 'Start date must be before end date',});
    }
    const client = await pool.connect();

    try {
        await client.query('BEGIN');

        const sessionResult = await client.query(
            `
            INSERT INTO sessions (type, query)
            VALUES ($1, $2)
            RETURNING id, public_id
            `,
            [type, searchTerm]
        );

        const { id: sessionID, public_id: publicID } = sessionResult.rows[0];

        await client.query(
            `
            INSERT INTO jobs (
                type,
                session_id,
                target_id,
                start_date,
                end_date
            )
            VALUES ($1, $2, $3 ,$4, $5)
            `,
            [type, sessionID, channelName, startDate, endDate]
        );

        await client.query('COMMIT');

        res.status(201).json({ publicID });

    } catch (err) {
        await client.query('ROLLBACK');
        console.error(err);
        jobFailures.inc();

        if (!res.headersSent) {
            res.status(500).json({ error: 'Database error' });
        }
    } finally {
        client.release();
    }
}

function extractYoutubeVideoId(input) {
    try {
        if (!/^https?:\/\//i.test(input)) {// add https protocol if not already present
            input = `https://${input}`;
        }
        
        const url = new URL(input);

        // youtube.com/watch?v=...
        if (
            url.hostname === 'youtube.com' ||
            url.hostname === 'www.youtube.com' ||
            url.hostname === 'm.youtube.com'
        ) {
            const id = url.searchParams.get('v');
            if (id) {
                return id;
            }
        }

        // youtu.be/...
        if (
            url.hostname === 'youtu.be' ||
            url.hostname === 'www.youtu.be'
        ) {
            return url.pathname.slice(1);
        }

        return null;
    } catch {
        return null;
    }
}

module.exports = jobsRoutes;