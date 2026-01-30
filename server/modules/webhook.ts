import type { Notification } from '@/types/notifications';
import { logError, logVerbose } from '@/utils/log';

export const sendWebhookNotification = async (endpoint: string, notification: Notification) => {
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
      logError(`Failed to send webhook notification: ${response.status} ${response.statusText}`);
      return false;
    }

    logVerbose(
      `Sent webhook notification to ${endpoint}: ${notification.message.substring(0, 50)}`,
    );

    return true;
  } catch (error) {
    logError('Failed to send webhook notification:', error);
    return false;
  }
};
