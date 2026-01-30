export interface WebPushAction {
  id: string;
  endpoint: string;
  method: 'POST' | 'GET' | 'DELETE';
  data?: Record<string, unknown>;
}

export interface Notification {
  title?: string;
  message: string;
  actions?: WebPushAction[];
}

export type NotificationChannel = 'signal' | 'webhook';
