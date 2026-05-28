import { Routes, Route } from 'react-router-dom';
import JobForm from './pages/jobForm';
import SessionStatus from './pages/sessionStatus';
import NotFound from './pages/notFound';
import ResultRouter from './pages/results/resultRouter';

function App() {
  return (
    <Routes>
      <Route path="/" element={<JobForm />} />
      <Route path="/sessions/:id" element={<SessionStatus />} />
      <Route path="/results/:type/:id" element={<ResultRouter />} />
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}

export default App;