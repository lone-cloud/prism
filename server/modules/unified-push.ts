import type { Notification } from '@/types/notifications';
import { logError, logVerbose } from '@/utils/log';

export const sendUnifiedPushNotification = async (endpoint: string, notification: Notification) => {
  try {
    const payload = {
      message: notification.message,
      ...(notification.title && { title: notification.title }),
      ...(notification.actions && { actions: notification.actions }),
    };

    const response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      logError(`Failed to send UP notification: ${response.status} ${response.statusText}`);
      return false;
    }

    logVerbose(`Sent UP notification to ${endpoint}: ${notification.message.substring(0, 50)}`);

    return true;
  } catch (error) {
    logError('Failed to send UP notification:', error);
    return false;
  }
};
