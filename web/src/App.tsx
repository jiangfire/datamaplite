import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import {
  SourcesPage,
  SourceDetailPage,
  SearchPage,
  ColumnDetailPage,
  TermsPage,
  LineagePage,
} from './pages';
import './App.css';

const App = () => {
  return (
    <BrowserRouter>
      <Routes>
        {/* Data Sources */}
        <Route path="/" element={<SourcesPage />} />
        <Route path="/sources/:id" element={<SourceDetailPage />} />

        {/* Search */}
        <Route path="/search" element={<SearchPage />} />

        {/* Column Detail */}
        <Route path="/columns/:id" element={<ColumnDetailPage />} />

        {/* Business Terms */}
        <Route path="/terms" element={<TermsPage />} />

        {/* Lineage Analysis */}
        <Route path="/lineage" element={<LineagePage />} />

        {/* Redirect unknown routes to home */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
};

export default App;
