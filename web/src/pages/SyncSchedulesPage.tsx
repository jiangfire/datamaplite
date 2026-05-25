import { useState } from 'react';
import { Layout } from '../components/Layout';
import { useSyncSchedules } from '../hooks/useSyncSchedules';
import { useSources } from '../hooks/useSources';
import type { SyncScheduleCreate, SyncScheduleUpdate } from '../services/syncScheduleService';

export function SyncSchedulesPage() {
  const { schedules, loading, createSchedule, updateSchedule, deleteSchedule } = useSyncSchedules();
  const { sources } = useSources();
  const [showModal, setShowModal] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState<Partial<SyncScheduleCreate>>({
    name: '',
    source_id: '',
    cron_expression: '',
    description: '',
    is_active: true,
  });

  const openCreate = () => {
    setEditingId(null);
    setFormData({
      name: '',
      source_id: sources[0]?.id || '',
      cron_expression: '0 2 * * *',
      description: '',
      is_active: true,
    });
    setShowModal(true);
  };

  const openEdit = (schedule: typeof schedules[0]) => {
    setEditingId(schedule.id);
    setFormData({
      name: schedule.name,
      source_id: schedule.source_id,
      cron_expression: schedule.cron_expression,
      description: schedule.description || '',
      is_active: schedule.is_active,
    });
    setShowModal(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.name || !formData.source_id || !formData.cron_expression) return;

    const data: SyncScheduleCreate = {
      name: formData.name,
      source_id: formData.source_id,
      cron_expression: formData.cron_expression,
      description: formData.description,
      is_active: formData.is_active ?? true,
    };

    if (editingId) {
      const update: SyncScheduleUpdate = {};
      if (formData.name !== undefined) update.name = formData.name;
      if (formData.cron_expression !== undefined) update.cron_expression = formData.cron_expression;
      if (formData.description !== undefined) update.description = formData.description;
      if (formData.is_active !== undefined) update.is_active = formData.is_active;
      await updateSchedule(editingId, update);
    } else {
      await createSchedule(data);
    }
    setShowModal(false);
  };

  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'success': return 'text-green-600';
      case 'failed': return 'text-red-600';
      case 'running': return 'text-blue-600';
      default: return 'text-gray-400';
    }
  };

  const getSourceName = (sourceId: string) => {
    return sources.find(s => s.id === sourceId)?.name || sourceId;
  };

  return (
    <Layout>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-2xl font-bold">Sync Schedules</h1>
          <button
            onClick={openCreate}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            + New Schedule
          </button>
        </div>

        {loading ? (
          <div className="text-gray-500">Loading...</div>
        ) : schedules.length === 0 ? (
          <div className="text-gray-500 text-center py-12">No sync schedules configured</div>
        ) : (
          <div className="bg-white rounded-lg shadow overflow-hidden">
            <table className="min-w-full">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Source</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Cron</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Last Run</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Next Run</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {schedules.map((schedule) => (
                  <tr key={schedule.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <div className="font-medium">{schedule.name}</div>
                      {schedule.description && (
                        <div className="text-xs text-gray-500">{schedule.description}</div>
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm">{getSourceName(schedule.source_id)}</td>
                    <td className="px-4 py-3">
                      <code className="bg-gray-100 px-2 py-1 rounded text-xs">{schedule.cron_expression}</code>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                        schedule.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {schedule.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm">
                      {schedule.last_run_at ? (
                        <span className={getStatusColor(schedule.last_run_status)}>
                          {new Date(schedule.last_run_at).toLocaleString()} ({schedule.last_run_status})
                        </span>
                      ) : (
                        <span className="text-gray-400">Never</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      {schedule.next_run_at ? (
                        new Date(schedule.next_run_at).toLocaleString()
                      ) : (
                        <span className="text-gray-400">-</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        onClick={() => openEdit(schedule)}
                        className="text-blue-600 hover:text-blue-800 text-sm mr-3"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => {
                          if (confirm('Delete this schedule?')) {
                            deleteSchedule(schedule.id);
                          }
                        }}
                        className="text-red-600 hover:text-red-800 text-sm"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {showModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg p-6 w-full max-w-md">
              <h2 className="text-xl font-bold mb-4">{editingId ? 'Edit Schedule' : 'New Schedule'}</h2>
              <form onSubmit={handleSubmit}>
                <div className="mb-4">
                  <label className="block text-sm font-medium mb-1">Name *</label>
                  <input
                    type="text"
                    value={formData.name || ''}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    className="w-full border rounded-lg px-3 py-2"
                    required
                  />
                </div>
                <div className="mb-4">
                  <label className="block text-sm font-medium mb-1">Data Source *</label>
                  <select
                    value={formData.source_id || ''}
                    onChange={(e) => setFormData({ ...formData, source_id: e.target.value })}
                    className="w-full border rounded-lg px-3 py-2"
                    required
                  >
                    <option value="">Select a source...</option>
                    {sources.map((source) => (
                      <option key={source.id} value={source.id}>{source.name}</option>
                    ))}
                  </select>
                </div>
                <div className="mb-4">
                  <label className="block text-sm font-medium mb-1">Cron Expression *</label>
                  <input
                    type="text"
                    value={formData.cron_expression || ''}
                    onChange={(e) => setFormData({ ...formData, cron_expression: e.target.value })}
                    className="w-full border rounded-lg px-3 py-2 font-mono text-sm"
                    placeholder="0 2 * * *"
                    required
                  />
                  <p className="text-xs text-gray-500 mt-1">Format: sec min hour day month weekday (e.g. 0 2 * * *)</p>
                </div>
                <div className="mb-4">
                  <label className="block text-sm font-medium mb-1">Description</label>
                  <textarea
                    value={formData.description || ''}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    className="w-full border rounded-lg px-3 py-2"
                    rows={2}
                  />
                </div>
                <div className="mb-4">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={formData.is_active}
                      onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
                      className="mr-2"
                    />
                    Active
                  </label>
                </div>
                <div className="flex justify-end gap-3">
                  <button
                    type="button"
                    onClick={() => setShowModal(false)}
                    className="px-4 py-2 border rounded-lg hover:bg-gray-50"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                  >
                    {editingId ? 'Update' : 'Create'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}
