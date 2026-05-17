import { Navigate, Outlet, useLocation } from 'react-router-dom';
import type { UserRole } from '../types';
import { useAuth } from './useAuth';

interface RequireRoleProps {
  role: UserRole | UserRole[];
}

export const RequireRole: React.FC<RequireRoleProps> = ({ role }) => {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 to-indigo-50/30 flex items-center justify-center px-4">
        <div className="text-center">
          <div className="w-10 h-10 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto" />
          <p className="text-slate-500 mt-4">正在验证权限...</p>
        </div>
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  const allowed = Array.isArray(role) ? role : [role];
  if (!allowed.includes(user.role)) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
};
