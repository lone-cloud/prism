import { createGroup, sendGroupMessage } from '@/modules/signal';
import { getMapping, register } from '@/modules/store';
import { sendUnifiedPushNotification } from '@/modules/unified-push';
import type { Notification } from '@/types/notifications';
import { logError, logVerbose } from '@/utils/log';

export const sendNotification = async (endpoint: string, notification: Notification) => {
  try {
    let mapping = getMapping(endpoint);

    if (!mapping) {
      const appName = endpoint.replace(/^(ntfy|proton)-/, '');
      register(endpoint, appName, 'signal');
      mapping = getMapping(endpoint);

      if (!mapping) {
        logError(`Failed to create mapping for endpoint: ${endpoint}`);
        return false;
      }
    }

    const { channel, upEndpoint, appName } = mapping;
    let { groupId } = mapping;

    if (channel === 'unifiedpush') {
      if (!upEndpoint) {
        logError(`UnifiedPush endpoint not configured for ${appName}`);
        return false;
      }

      return await sendUnifiedPushNotification(upEndpoint, notification);
    }

    if (!groupId) {
      groupId = await createGroup(appName);
      register(endpoint, appName, 'signal', { groupId });
    }

    await sendGroupMessage(groupId, notification);

    logVerbose(`Sent Signal notification to ${appName}: ${notification.message.substring(0, 50)}`);

    return true;
  } catch (error) {
    logError('Failed to send notification:', error);
    return false;
  }
};
