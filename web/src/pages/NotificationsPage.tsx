import { useState } from 'react';
import { Bell, Check, CheckCheck, Clock, Database, Plus, Minus, Edit3, Type } from 'lucide-react';
import { Layout, Button, Card, CardContent } from '../components';
import { useNotifications } from '../hooks';

const getChangeTypeIcon = (changeType: string) => {
  switch (changeType) {
    case 'add_object':
    case 'add_column':
      return <Plus size={16} className="text-green-500" />;
    case 'drop_object':
    case 'drop_column':
      return <Minus size={16} className="text-red-500" />;
    case 'alter_column':
    case 'change_type':
      return <Edit3 size={16} className="text-yellow-500" />;
    default:
      return <Type size={16} className="text-slate-400" />;
  }
};

const getChangeTypeLabel = (changeType: string) => {
  const labels: Record<string, string> = {
    add_object: '新增对象',
    drop_object: '删除对象',
    add_column: '新增字段',
    drop_column: '删除字段',
    alter_column: '修改字段',
    change_type: '类型变更',
  };
  return labels[changeType] || changeType;
};

export const NotificationsPage: React.FC = () => {
  const [unreadOnly, setUnreadOnly] = useState(false);
  const { notifications, stats, loading, error, markAsRead, markAllAsRead } =
    useNotifications(unreadOnly);

  const handleMarkAsRead = async (id: string) => {
    await markAsRead([id]);
  };

  const formatTime = (time: string) => {
    const date = new Date(time);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return '刚刚';
    if (minutes < 60) return `${minutes}分钟前`;
    if (hours < 24) return `${hours}小时前`;
    if (days < 7) return `${days}天前`;
    return date.toLocaleDateString('zh-CN');
  };

  return (
    <Layout>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">通知中心</h1>
          <p className="text-slate-500 mt-1">
            {stats && (
              <span>
                共 {stats.total_count} 条通知，
                <span className="text-indigo-600 font-medium"> {stats.unread_count} </span>
                条未读
              </span>
            )}
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant={unreadOnly ? 'primary' : 'secondary'}
            onClick={() => setUnreadOnly(!unreadOnly)}
          >
            <Bell size={18} className="mr-2" />
            {unreadOnly ? '显示全部' : '仅未读'}
          </Button>
          {stats && stats.unread_count > 0 && (
            <Button variant="secondary" onClick={markAllAsRead}>
              <CheckCheck size={18} className="mr-2" />
              全部已读
            </Button>
          )}
        </div>
      </div>

      {loading ? (
        <div className="text-center py-12">加载中...</div>
      ) : error ? (
        <Card>
          <CardContent className="p-8 text-center text-red-500">{error}</CardContent>
        </Card>
      ) : notifications.length === 0 ? (
        <Card>
          <CardContent className="py-16 text-center">
            <div className="w-16 h-16 rounded-full bg-slate-100 flex items-center justify-center mx-auto mb-4">
              <Bell size={32} className="text-slate-400" />
            </div>
            <p className="text-slate-500">暂无通知</p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {notifications.map((notification) => (
            <Card
              key={notification.id}
              className={notification.is_read ? 'opacity-70' : 'border-l-4 border-l-indigo-500'}
            >
              <CardContent className="p-4">
                <div className="flex items-start gap-4">
                  <div className="flex-shrink-0 mt-1">
                    {getChangeTypeIcon(notification.change_type)}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <h3 className="font-medium text-slate-900">{notification.title}</h3>
                        <p className="text-sm text-slate-600 mt-1">{notification.message}</p>
                      </div>
                      <span className="text-xs text-slate-400 flex-shrink-0">
                        {formatTime(notification.created_at)}
                      </span>
                    </div>

                    <div className="flex items-center gap-2 mt-3 text-xs">
                      <span className="flex items-center gap-1 px-2 py-1 bg-slate-100 text-slate-600 rounded">
                        <Database size={12} />
                        {notification.source_name}
                      </span>
                      <span className="px-2 py-1 bg-indigo-50 text-indigo-700 rounded">
                        {getChangeTypeLabel(notification.change_type)}
                      </span>
                      {notification.webhook_sent ? (
                        <span className="flex items-center gap-1 px-2 py-1 bg-green-50 text-green-700 rounded">
                          <Check size={12} />
                          Webhook已发送
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 px-2 py-1 bg-slate-50 text-slate-500 rounded">
                          <Clock size={12} />
                          Webhook未发送
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="flex flex-col gap-1">
                    {!notification.is_read && (
                      <button
                        onClick={() => handleMarkAsRead(notification.id)}
                        className="p-2 text-slate-400 hover:text-indigo-600 hover:bg-indigo-50 rounded-lg"
                        title="标记为已读"
                      >
                        <Check size={16} />
                      </button>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </Layout>
  );
};
