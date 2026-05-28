function SessionHeader({ counts }) {
  return (
      <div style={{ padding: 20 }}>
        <h2>Job Status</h2>
        <p>Total Jobs: {counts.total}</p>
        <p>Succeeded: {counts.succeeded}</p>
        <p>Failed: {counts.failed}</p>
        <p>Pending: {counts.pending}</p>
        <p>Running: {counts.running}</p>
    </div>
  )
}
export default SessionHeader;