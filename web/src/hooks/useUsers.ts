import { useCallback, useEffect, useState } from 'react';
import { userService } from '../services';
import type { UserCreateRequest, UserInfo, UserUpdateRequest } from '../types';

export const useUsers = () => {
  const [users, setUsers] = useState<UserInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchUsers = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await userService.listUsers();
      setUsers(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch users');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const createUser = async (data: UserCreateRequest) => {
    await userService.createUser(data);
    await fetchUsers();
  };

  const updateUser = async (id: string, data: UserUpdateRequest) => {
    const updated = await userService.updateUser(id, data);
    setUsers((prev) => prev.map((u) => (u.id === id ? updated : u)));
    return updated;
  };

  const deleteUser = async (id: string) => {
    await userService.deleteUser(id);
    setUsers((prev) => prev.filter((u) => u.id !== id));
  };

  return {
    users,
    loading,
    error,
    refetch: fetchUsers,
    createUser,
    updateUser,
    deleteUser,
  };
};
