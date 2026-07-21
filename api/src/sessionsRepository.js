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

async function getSessionErrors(pool, public_id){
    const result = await pool.query(
        `
        SELECT target_id, last_error AS message
        FROM jobs
        WHERE session_public_id = $1 AND type != 'CHANNEL_SEARCH' AND status = 'FAILED'
        `,
        [public_id]
    );

    return result.rows;
}

async function getChannelSearchStatus(pool, public_id){
    const result = await pool.query(
        `
        SELECT status
        FROM jobs 
        WHERE type = 'CHANNEL_SEARCH' AND session_public_id = $1
        `,
        [public_id]
    );

    return result.rows[0].status;
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

async function getSessionResults(pool, public_id){
    const result = await pool.query(
        `
        SELECT target_id, result FROM jobs WHERE session_public_id = $1 AND status = 'SUCCEEDED' AND type != 'CHANNEL_SEARCH'
        `,
        [public_id]
    );
    return result.rows.reduce((acc, row) => {
        acc[row.target_id] = (row.result ?? []).map(item => ({
            ...item,
            target_id: row.target_id,
        }));

        return acc;
    }, {});
}

module.exports = {getSessionJobCounts, getSessionErrors, getChannelSearchStatus, getSessionType, getSessionResults};