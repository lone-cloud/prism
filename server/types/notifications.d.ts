export interface UnifiedPushAction {
  id: string;
  endpoint: string;
  method: 'POST' | 'GET' | 'DELETE';
  data?: Record<string, unknown>;
}

export interface Notification {
  title?: string;
  message: string;
  actions?: UnifiedPushAction[];
}

export type NotificationChannel = 'signal' | 'unifiedpush';
