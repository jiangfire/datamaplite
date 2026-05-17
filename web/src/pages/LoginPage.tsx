import { useState } from 'react';
import { Navigate, useLocation, useNavigate } from 'react-router-dom';
import { Database, Lock, User } from 'lucide-react';
import { Button, Card, CardContent, Input } from '../components';
import { useAuth } from '../auth';

export const LoginPage: React.FC = () => {
  const { isAuthenticated, login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const from =
    (location.state as { from?: { pathname?: string; search?: string } } | null)
      ?.from;
  const nextParam = new URLSearchParams(location.search).get('next');
  const redirectTo = from?.pathname
    ? `${from.pathname}${from.search || ''}`
    : nextParam || '/';

  if (isAuthenticated) {
    return <Navigate to={redirectTo} replace />;
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!username.trim() || !password) {
      setError('请输入用户名和密码');
      return;
    }

    setSubmitting(true);
    setError(null);
    try {
      await login({
        username: username.trim(),
        password,
      });
      navigate(redirectTo, { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(99,102,241,0.18),_transparent_35%),linear-gradient(135deg,_#f8fafc,_#eef2ff_40%,_#f8fafc)] px-4 py-12">
      <div className="mx-auto max-w-5xl grid gap-8 lg:grid-cols-[1.2fr,0.8fr] items-center">
        <div className="hidden lg:block">
          <div className="inline-flex items-center gap-3 rounded-full bg-white/80 px-4 py-2 text-sm text-slate-600 shadow-sm border border-white/70">
            <Database size={16} className="text-indigo-600" />
            DataMap-Lite
          </div>
          <h1 className="mt-6 text-5xl font-black tracking-tight text-slate-900">
            登录后进入
            <span className="block text-indigo-600">元数据治理台</span>
          </h1>
          <p className="mt-6 max-w-xl text-lg leading-8 text-slate-600">
            当前前端已切到受保护路由模式。登录后可继续使用数据源、术语、数据质量、告警与通知等能力。
          </p>
        </div>

        <Card className="border-white/80 bg-white/90 shadow-2xl shadow-indigo-100/70 backdrop-blur">
          <CardContent className="p-8">
            <div className="flex items-center gap-3 mb-8">
              <div className="w-12 h-12 rounded-2xl bg-gradient-to-br from-indigo-600 to-violet-600 text-white flex items-center justify-center shadow-lg shadow-indigo-500/30">
                <Lock size={22} />
              </div>
              <div>
                <h2 className="text-2xl font-bold text-slate-900">登录</h2>
                <p className="text-sm text-slate-500">
                  使用系统账号访问受保护页面
                </p>
              </div>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <Input
                label="用户名"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                autoComplete="username"
                placeholder="请输入用户名"
                required
              />
              <Input
                label="密码"
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                autoComplete="current-password"
                placeholder="请输入密码"
                required
              />

              {error && (
                <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                  {error}
                </div>
              )}

              <Button type="submit" className="w-full" loading={submitting}>
                <User size={16} className="mr-2" />
                登录
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};
