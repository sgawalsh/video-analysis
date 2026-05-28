class EventHub {
  constructor() {
    this.sessions = new Map(); 
    // sessionId -> Set(res)
  }

  addClient(sessionId, res) {
    if (!this.sessions.has(sessionId)) {
      this.sessions.set(sessionId, new Set());
    }
    this.sessions.get(sessionId).add(res);
  }

  removeClient(sessionId, res) {
    const set = this.sessions.get(sessionId);
    if (!set) return;

    set.delete(res);
    if (set.size === 0) this.sessions.delete(sessionId);
  }

  broadcast(sessionId, event) {
    const clients = this.sessions.get(sessionId);
    if (!clients) return;

    const payload = `event: ${event.n_type}\ndata: ${JSON.stringify(event)}\n\n`;

    for (const res of clients) {
      res.write(payload);
    }
  }
}

module.exports = EventHub;