async function getSessionJobCounts(pool, public_id){
    const result = await pool.query(
        `
        SELECT
            COUNT(*) FILTER (WHERE status = 'SUCCEEDED') AS succeeded,
            COUNT(*) FILTER (WHERE status = 'FAILED') AS failed,
            COUNT(*) FILTER (WHERE status = 'PENDING') AS pending,
            COUNT(*) FILTER (WHERE status = 'RUNNING') AS running,
            COUNT(*) AS total
        FROM jobs
        WHERE session_public_id = $1 AND type != 'CHANNEL_SEARCH'
        `,
        [public_id]
    );

    return result.rows[0];
}

module.exports = getSessionJobCounts;