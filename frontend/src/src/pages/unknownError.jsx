import { Link } from 'react-router-dom';

function UnknownError() {
  return (
    <div style={{ padding: '20px', textAlign: 'center' }}>
      <h1>Unknown Error Encountered.</h1>
      <Link to="/">Go to Homepage</Link>
    </div>
  );
};

export default UnknownError;