import { useState } from 'react';
import { Plus, Shield, Trash2, UserCog, Users } from 'lucide-react';
import {
  Badge,
  Button,
  Card,
  CardContent,
  Input,
  Layout,
  Modal,
  Select,
} from '../components';
import { useUsers } from '../hooks';
import { useAuth } from '../auth';
import type { UserCreateRequest, UserInfo, UserRole } from '../types';

const ROLE_OPTIONS = [
  { value: 'user', label: '普通用户' },
  { value: 'admin', label: '管理员' },
];

const emptyCreateForm: UserCreateRequest = {
  username: '',
  password: '',
  email: '',
  role: 'user',
};

export const AdminUsersPage: React.FC = () => {
  const { users, loading, error, createUser, updateUser, deleteUser } =
    useUsers();
  const { user: currentUser } = useAuth();

  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] =
    useState<UserCreateRequest>(emptyCreateForm);
  const [createError, setCreateError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [roleSavingId, setRoleSavingId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const handleCreate = async () => {
    setCreateError(null);
    if (!createForm.username.trim() || !createForm.password) {
      setCreateError('用户名和密码必填');
      return;
    }
    if (!createForm.email.trim()) {
      setCreateError('邮箱必填');
      return;
    }
    if (createForm.password.length < 6) {
      setCreateError('密码至少 6 位');
      return;
    }
    setSubmitting(true);
    try {
      await createUser({
        username: createForm.username.trim(),
        password: createForm.password,
        email: createForm.email.trim(),
        role: createForm.role,
      });
      setCreateForm(emptyCreateForm);
      setShowCreate(false);
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : '创建用户失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleRoleChange = async (target: UserInfo, role: UserRole) => {
    if (role === target.role) return;
    setActionError(null);
    setRoleSavingId(target.id);
    try {
      await updateUser(target.id, { role });
    } catch (err) {
      setActionError(err instanceof Error ? err.message : '修改角色失败');
    } finally {
      setRoleSavingId(null);
    }
  };

  const handleDelete = async (target: UserInfo) => {
    if (currentUser?.id === target.id) {
      setActionError('无法删除当前登录账号');
      return;
    }
    if (!window.confirm(`确定要删除用户 ${target.username} 吗？此操作不可撤销。`))
      return;
    setActionError(null);
    setDeletingId(target.id);
    try {
      await deleteUser(target.id);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : '删除用户失败');
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <Layout>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">用户管理</h1>
          <p className="text-slate-500 mt-1">管理系统账号、角色和访问权限</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus size={18} className="mr-2" />
          创建用户
        </Button>
      </div>

      {actionError && (
        <Card className="mb-4 border-red-200 bg-red-50/50">
          <CardContent className="py-3 text-sm text-red-600">
            {actionError}
          </CardContent>
        </Card>
      )}

      {loading ? (
        <Card>
          <CardContent className="py-12 text-center text-slate-500">
            加载中...
          </CardContent>
        </Card>
      ) : error ? (
        <Card>
          <CardContent className="py-8 text-center text-red-500">
            {error}
          </CardContent>
        </Card>
      ) : users.length === 0 ? (
        <Card>
          <CardContent className="py-16 text-center">
            <div className="w-16 h-16 rounded-full bg-slate-100 flex items-center justify-center mx-auto mb-4">
              <Users size={32} className="text-slate-400" />
            </div>
            <p className="text-slate-500">暂无用户</p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead className="bg-slate-50/80 text-left text-xs uppercase tracking-wider text-slate-500">
                <tr>
                  <th className="px-6 py-3">用户名</th>
                  <th className="px-6 py-3">邮箱</th>
                  <th className="px-6 py-3">角色</th>
                  <th className="px-6 py-3">创建时间</th>
                  <th className="px-6 py-3 text-right">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {users.map((u) => {
                  const isSelf = currentUser?.id === u.id;
                  return (
                    <tr key={u.id} className="hover:bg-slate-50/60">
                      <td className="px-6 py-3 font-medium text-slate-900">
                        <div className="flex items-center gap-2">
                          {u.role === 'admin' ? (
                            <Shield size={16} className="text-indigo-500" />
                          ) : (
                            <UserCog size={16} className="text-slate-400" />
                          )}
                          {u.username}
                          {isSelf && (
                            <Badge variant="info" className="ml-1">
                              你
                            </Badge>
                          )}
                        </div>
                      </td>
                      <td className="px-6 py-3 text-slate-600">{u.email}</td>
                      <td className="px-6 py-3">
                        <Select
                          options={ROLE_OPTIONS}
                          value={u.role}
                          onChange={(e) =>
                            handleRoleChange(u, e.target.value as UserRole)
                          }
                          disabled={isSelf || roleSavingId === u.id}
                          className="w-32"
                        />
                      </td>
                      <td className="px-6 py-3 text-slate-500">
                        {new Date(u.created_at).toLocaleString('zh-CN')}
                      </td>
                      <td className="px-6 py-3 text-right">
                        <button
                          onClick={() => handleDelete(u)}
                          disabled={isSelf || deletingId === u.id}
                          className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-sm text-slate-500 hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40"
                          title={isSelf ? '不能删除自己' : '删除用户'}
                        >
                          <Trash2 size={16} />
                          删除
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <Modal
        isOpen={showCreate}
        onClose={() => {
          setShowCreate(false);
          setCreateError(null);
          setCreateForm(emptyCreateForm);
        }}
        title="创建用户"
      >
        <div className="space-y-4">
          <Input
            label="用户名"
            required
            value={createForm.username}
            onChange={(e) =>
              setCreateForm((p) => ({ ...p, username: e.target.value }))
            }
            placeholder="登录用户名"
          />
          <Input
            label="邮箱"
            type="email"
            required
            value={createForm.email}
            onChange={(e) =>
              setCreateForm((p) => ({ ...p, email: e.target.value }))
            }
            placeholder="user@example.com"
          />
          <Input
            label="初始密码"
            type="password"
            required
            value={createForm.password}
            onChange={(e) =>
              setCreateForm((p) => ({ ...p, password: e.target.value }))
            }
            helperText="至少 6 位，创建后用户可自行修改"
          />
          <Select
            label="角色"
            options={ROLE_OPTIONS}
            value={createForm.role}
            onChange={(e) =>
              setCreateForm((p) => ({
                ...p,
                role: e.target.value as UserRole,
              }))
            }
          />
          {createError && (
            <p className="text-sm text-red-600">{createError}</p>
          )}
          <div className="flex justify-end gap-2 pt-2">
            <Button
              variant="secondary"
              onClick={() => {
                setShowCreate(false);
                setCreateError(null);
                setCreateForm(emptyCreateForm);
              }}
              disabled={submitting}
            >
              取消
            </Button>
            <Button onClick={handleCreate} loading={submitting}>
              创建
            </Button>
          </div>
        </div>
      </Modal>
    </Layout>
  );
};
