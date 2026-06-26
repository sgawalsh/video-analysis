async function runMigrations(pool, { enableCron = false } = {}) {

  // Create enum type for job type
  await pool.query(`
    DO $$
    BEGIN
      IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_type') THEN
        CREATE TYPE job_type AS ENUM (
          'CHANNEL_SEARCH',
          'KEYWORD_SEARCH',
          'SEMANTIC_SEARCH',
          'TOPIC_DETECTION_EMBED',
          'TOPIC_DETECTION_LLM',
          'VIDEO_SUMMARIZATION_TRANSCRIBE',
          'VIDEO_SUMMARIZATION_LLM'
        );
      END IF;
    END$$;
  `);

  // Create enum type for job status
  await pool.query(`
    DO $$
    BEGIN
      IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_status') THEN
        CREATE TYPE job_status AS ENUM (
          'PENDING',
          'RUNNING',
          'SUCCEEDED',
          'FAILED'
        );
      END IF;
    END$$;
  `);

  // Create session table
  await pool.query(`
    CREATE TABLE IF NOT EXISTS sessions (
      id SERIAL PRIMARY KEY,
      public_id UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
      type job_type NOT NULL,
      query TEXT,
      created_at TIMESTAMPTZ DEFAULT NOW()
    )
  `);

  // Create jobs table
  await pool.query(`
    CREATE TABLE IF NOT EXISTS jobs (
      id SERIAL PRIMARY KEY,
      session_id INT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
      session_public_id UUID NOT NULL,
      
      type job_type NOT NULL,
      status job_status NOT NULL DEFAULT 'PENDING',
      target_id TEXT NOT NULL,
      query TEXT,
      result JSONB NOT NULL DEFAULT '[]'::jsonb,

      created_at TIMESTAMPTZ DEFAULT NOW(),
      updated_at TIMESTAMPTZ DEFAULT NOW(),
      started_at TIMESTAMPTZ,

      start_date DATE,
      end_date DATE,

      attempts INT NOT NULL DEFAULT 0,
      last_error TEXT
    )
  `);

  await pool.query(`
    CREATE TABLE IF NOT EXISTS llm_job_info (
      id SERIAL PRIMARY KEY,
      job_id INT NOT NULL UNIQUE REFERENCES jobs(id) ON DELETE CASCADE,
      input JSONB
    )
  `);
  
  // Create index on status for efficient querying of pending jobs for KEDA
  await pool.query(`
    CREATE INDEX IF NOT EXISTS idx_jobs_status_pending
    ON jobs(status)
    WHERE status = 'PENDING';
  `);

  //create unique index to prevent duplicate jobs for same session and target
  await pool.query(`
    CREATE UNIQUE INDEX IF NOT EXISTS jobs_session_type_target_unique
    ON jobs (session_id, target_id);
  `);

   // Fires re-count triggers and completed notifications
  await pool.query(`
    CREATE OR REPLACE FUNCTION sse_notifications()
    RETURNS TRIGGER AS $$
    BEGIN

      PERFORM pg_notify(
        'session_events',
        json_build_object(
          'session_public_id', NEW.session_public_id,
          'n_type', 'status_changed'
        )::text
      );

      IF NEW.status = 'SUCCEEDED' THEN
        PERFORM pg_notify(
          'session_events',
          json_build_object(
            'session_public_id', NEW.session_public_id,
            'n_type', 'job_completed',
            'target_id', NEW.target_id,
            'result', NEW.result
          )::text
        );
      END IF;

      RETURN NEW;
    END;
    $$ LANGUAGE plpgsql;
  `);

  // Create trigger function for sse notifications
  await pool.query(`
    DO $$
    BEGIN
      IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'sse_notifications_trigger'
      ) THEN
        CREATE TRIGGER sse_notifications_trigger
        AFTER UPDATE ON jobs
        FOR EACH ROW
        WHEN (OLD.status IS DISTINCT FROM NEW.status AND NEW.type IS DISTINCT FROM 'CHANNEL_SEARCH')
        EXECUTE FUNCTION sse_notifications();
      END IF;
    END
    $$;
  `);

  //sets updated_at timestamp on update
  await pool.query(`
    CREATE OR REPLACE FUNCTION auto_updated_at()
    RETURNS TRIGGER AS $$
    BEGIN
      NEW.updated_at = NOW();
      RETURN NEW;
    END;
    $$ LANGUAGE plpgsql;
  `);

  // Create trigger function for automatic updated_at timestamp
  await pool.query(`
    DO $$
    BEGIN
      IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'jobs_set_updated_at'
      ) THEN
        CREATE TRIGGER jobs_set_updated_at
        BEFORE UPDATE ON jobs
        FOR EACH ROW
        EXECUTE FUNCTION auto_updated_at();
      END IF;
    END
    $$;
  `);
  
  // Create trigger function to notify when new pending jobs are available
  await pool.query(`
    CREATE OR REPLACE FUNCTION notify_jobs_available()
    RETURNS trigger AS $$
    BEGIN
      IF NEW.type IN ('CHANNEL_SEARCH', 'KEYWORD_SEARCH', 'SEMANTIC_SEARCH', 'TOPIC_DETECTION_EMBED', 'VIDEO_SUMMARIZATION_TRANSCRIBE') THEN
        PERFORM pg_notify('jobs_available', '');
      ELSIF NEW.type IN ('TOPIC_DETECTION_LLM', 'VIDEO_SUMMARIZATION_LLM') THEN
        PERFORM pg_notify('llm_jobs_available', '');
      END IF;
      RETURN NEW;
    END;
    $$ LANGUAGE plpgsql;
  `);
  
  // Create trigger to call the function after insert or status update
  await pool.query(`
    DO $$
    BEGIN
      IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'jobs_notify_trigger'
      ) THEN
        CREATE TRIGGER jobs_notify_trigger
        AFTER INSERT OR UPDATE OF status
        ON jobs
        FOR EACH ROW
        WHEN (NEW.status = 'PENDING')
        EXECUTE FUNCTION notify_jobs_available();
      END IF;
    END
    $$;
  `);


  if (enableCron) {
    // Ensure pg_cron extension exists
    await pool.query(`CREATE EXTENSION IF NOT EXISTS pg_cron;`);

    const jobName = 'cleanup_jobs';
    const schedule = process.env.cron_schedule || '0 * * * *';
    const retention = process.env.cron_job_retention || '1 day';
    const sqlCommand = `
      DELETE FROM jobs
      WHERE status = 'SUCCEEDED'
        AND created_at < NOW() - INTERVAL '${retention}';
    `;

    // Unschedule existing job (if any)
    await pool.query(`
      SELECT cron.unschedule(jobid)
      FROM cron.job
      WHERE jobname = $1;
    `, [jobName]);

    // Schedule new job
    await pool.query(`
      SELECT cron.schedule($1, $2, $3);
    `, [jobName, schedule, sqlCommand]);

    console.log(`Scheduled cron job '${jobName}' with schedule '${schedule}' to clean up succeeded jobs older than ${retention}.`);
  }
}

module.exports = { runMigrations };