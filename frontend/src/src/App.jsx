import { Routes, Route } from 'react-router-dom';
import JobForm from './pages/jobForm';
import JobStatus from './pages/jobStatus';
import NotFound from './pages/notFound';
import ResultRouter from './pages/results/resultRouter';

function App() {
  return (
    <Routes>
      <Route path="/" element={<JobForm />} />
      <Route path="/jobs/:id" element={<JobStatus />} />
      <Route path="/results/:type/:id" element={<ResultRouter />} />
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}

export default App;