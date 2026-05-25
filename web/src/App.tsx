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
  TagDetailPage,
  AlertRulesPage,
  NotificationsPage,
  AdminUsersPage,
  SyncSchedulesPage,
  DashboardPage,
} from './pages';
import {
  AuthProvider,
  PublicOnlyRoute,
  RequireAuth,
  RequireRole,
} from './auth';
import { ToastProvider } from './components/ToastProvider';
import './App.css';

const App = () => {
  return (
    <BrowserRouter>
      <AuthProvider>
        <ToastProvider>
          <Routes>
          <Route element={<PublicOnlyRoute />}>
            <Route path="/login" element={<LoginPage />} />
          </Route>

          <Route element={<RequireAuth />}>
            {/* Dashboard */}
            <Route path="/dashboard" element={<DashboardPage />} />

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
            <Route path="/tags/:id" element={<TagDetailPage />} />

            {/* Alert Rules */}
            <Route path="/alerts" element={<AlertRulesPage />} />

            {/* Notifications */}
            <Route path="/notifications" element={<NotificationsPage />} />

            {/* Admin */}
            <Route element={<RequireRole role="admin" />}>
              <Route path="/admin/users" element={<AdminUsersPage />} />
              <Route path="/admin/sync" element={<SyncSchedulesPage />} />
            </Route>
          </Route>

          {/* Redirect unknown routes to home */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </ToastProvider>
    </AuthProvider>
  </BrowserRouter>
  );
};

export default App;
