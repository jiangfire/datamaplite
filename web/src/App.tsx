import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import {
  LoginPage,
  SourcesPage,
  SourceDetailPage,
  SearchPage,
  ColumnDetailPage,
  TermsPage,
  TermDetailPage,
  LineagePage,
  DQRulesPage,
  DQResultsPage,
  TagsPage,
  AlertRulesPage,
  NotificationsPage,
} from './pages';
import { AuthProvider, PublicOnlyRoute, RequireAuth } from './auth';
import './App.css';

const App = () => {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route element={<PublicOnlyRoute />}>
            <Route path="/login" element={<LoginPage />} />
          </Route>

          <Route element={<RequireAuth />}>
            {/* Data Sources */}
            <Route path="/" element={<SourcesPage />} />
            <Route path="/sources/:id" element={<SourceDetailPage />} />

            {/* Search */}
            <Route path="/search" element={<SearchPage />} />

            {/* Column Detail */}
            <Route path="/columns/:id" element={<ColumnDetailPage />} />

            {/* Business Terms */}
            <Route path="/terms" element={<TermsPage />} />
            <Route path="/terms/:id" element={<TermDetailPage />} />

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
          </Route>

          {/* Redirect unknown routes to home */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
};

export default App;
