function SessionHeader({ counts }) {
  return (
      <div style={{ padding: 20 }}>
        <h2>Job Status</h2>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th>Total Jobs</th>
                <th>Succeeded</th>
                <th>Failed</th>
                <th>Running</th>
                <th>Pending</th>
              </tr>
            </thead>
            <tbody> 
              <tr>
                <td>{counts.total}</td>
                <td>{counts.succeeded}</td>
                <td>{counts.failed}</td>
                <td>{counts.running}</td>
                <td>{counts.pending}</td>
              </tr>
          </tbody>
        </table>
      </div>
  )
}
export default SessionHeader;