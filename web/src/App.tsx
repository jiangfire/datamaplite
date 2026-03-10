import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import {
  SourcesPage,
  SourceDetailPage,
  SearchPage,
  ColumnDetailPage,
  TermsPage,
  LineagePage,
  DQRulesPage,
  DQResultsPage,
  TagsPage,
  AlertRulesPage,
  NotificationsPage,
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

        {/* Data Quality */}
        <Route path="/dq/rules" element={<DQRulesPage />} />
        <Route path="/dq/results" element={<DQResultsPage />} />

        {/* Tags */}
        <Route path="/tags" element={<TagsPage />} />

        {/* Alert Rules */}
        <Route path="/alerts" element={<AlertRulesPage />} />

        {/* Notifications */}
        <Route path="/notifications" element={<NotificationsPage />} />

        {/* Redirect unknown routes to home */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
};

export default App;
