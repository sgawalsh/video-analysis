const {getSessionJobCounts} = require('./sessionsRepository');
async function createPgListener({ pool, hub }) {
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

  while (true) {
    let client, eventPayload;

    try {
      client = await pool.connect();
      await client.query('LISTEN session_events');

      console.log("PG listener connected");

      await new Promise((resolve, reject) => {
        const handleNotification = async (msg) => {
          try {
            const rawEvent = JSON.parse(msg.payload);
            console.log("Received PG event:", rawEvent);

            if (rawEvent.n_type === 'status_changed'){

              // Fetch the derived counts 
              const counts = await getSessionJobCounts(pool, rawEvent.session_public_id);
              console.log("Derived counts for session", rawEvent.session_public_id, counts);
              
              eventPayload = {
                ...rawEvent,
                counts,
                emittedAt: Date.now() // To handle race conditions
              };
            } else if (['job_completed', 'job_failed', 'channel_search_succeeded', 'channel_search_failed'].includes(rawEvent.n_type)) {
              eventPayload = rawEvent
            }else{
              console.log("Unknown event type:", rawEvent.n_type);
              return;
            }

            hub.broadcast(eventPayload.session_public_id, eventPayload);
          } catch (err) {
            console.error("Error processing PG event payload:", err);
          }
        };

        client.on('notification', handleNotification);
        client.on('error', reject);
        
        client.once('end', () => {
          client.off('notification', handleNotification);
          resolve();
        });
      });
    } catch (err) {
      console.error("PG listener crashed:", err);
    } finally {
      if (client) {
        try {
          await client.query('UNLISTEN session_events').catch(() => {});
          client.release();
        } catch (e) {
          console.error("Error releasing client", e);
        }
      }
    }

    console.log("PG listener restarting in 2s...");
    await sleep(2000);
  }
}

module.exports = createPgListener;
