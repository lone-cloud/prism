interface EndpointMapping {
  endpoint: string;
  groupId: string;
  appName: string;
}

interface Notification {
  id: string;
  topic: string;
  endpoint?: string;
  time: number;
  title?: string;
  message: string;
  priority?: number;
  tags?: string[];
  click?: string;
}

const mappings = new Map<string, EndpointMapping>();
const notifications: Notification[] = [];
const MAX_NOTIFICATIONS = 1000;
const RETENTION_DAYS = 7;

export const register = (endpoint: string, groupId: string, appName: string) => {
  mappings.set(endpoint, { endpoint, groupId, appName });
};

export const getGroupId = (endpoint: string) => mappings.get(endpoint)?.groupId;

export const getAppName = (endpoint: string) => mappings.get(endpoint)?.appName;

export const getAllMappings = () => Array.from(mappings.values());

export const remove = (endpoint: string) => {
  mappings.delete(endpoint);
};

export const addNotification = (notification: Omit<Notification, 'id' | 'time'>) => {
  const id = crypto.randomUUID();
  const time = Date.now();

  notifications.unshift({
    id,
    time,
    ...notification,
  });

  if (notifications.length > MAX_NOTIFICATIONS) {
    notifications.splice(MAX_NOTIFICATIONS);
  }

  const cutoff = Date.now() - RETENTION_DAYS * 24 * 60 * 60 * 1000;
  const firstOldIndex = notifications.findIndex((n) => n.time < cutoff);
  if (firstOldIndex > 0) {
    notifications.splice(firstOldIndex);
  }

  return id;
};

export const getNotifications = (filter?: { topic?: string; endpoint?: string }) => {
  let result = notifications;

  if (filter?.topic) {
    result = result.filter((n) => n.topic === filter.topic);
  }

  if (filter?.endpoint) {
    result = result.filter((n) => n.endpoint === filter.endpoint);
  }

  return result;
};
