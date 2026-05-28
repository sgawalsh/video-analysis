const express = require('express');

function resultRoutes({ pool }) {
    const router = express.Router();

    router.get('/semanticSearchResult/:public_id', async (req, res) => {
        res.set('Cache-Control', 'no-store');
        const { public_id } = req.params;

        try {
            const result = await pool.query(
                `
                SELECT
                result,
                target_id
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

module.exports = resultRoutes;