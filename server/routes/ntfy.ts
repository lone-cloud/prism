import { Hono } from 'hono';
import { basicAuth } from 'hono/basic-auth';
import { sendNotification } from '@/modules/notifications';
import { verifyApiKey } from '@/utils/auth';
import { logError, logVerbose } from '@/utils/log';

export const ntfy = new Hono();

ntfy.use(
  '*',
  basicAuth({
    verifyUser: (_, password) => verifyApiKey(password),
    realm: 'PRISM ntfy - Username: any, Password: API_KEY',
  }),
);

ntfy.post('/:topic', async (c) => {
  try {
    const topic = decodeURIComponent(c.req.param('topic'));

    if (!topic || topic.includes('/')) {
      return c.text('Invalid topic', 400);
    }

    let message = await c.req.text();
    if (!message) {
      return c.text('Message required', 400);
    }

    const contentType = c.req.header('content-type') || '';
    let title: string | undefined =
      c.req.header('X-Title') || c.req.header('Title') || c.req.header('t') || undefined;

    if (contentType.includes('application/x-www-form-urlencoded')) {
      const params = new URLSearchParams(message);
      message = params.get('message') || message;
      title = title || params.get('title') || params.get('t') || undefined;
    }

    if (title === topic) title = undefined;

    await sendNotification(`ntfy-${topic}`, {
      title: title || undefined,
      message: message,
    });

    logVerbose(`Sent ntfy message to topic ${topic}: ${title || message.substring(0, 50)}`);

    return c.json({
      id: Date.now().toString(),
      time: Math.floor(Date.now() / 1000),
      event: 'message',
      topic,
      message,
    });
  } catch (error) {
    logError('Failed to handle ntfy publish:', error);
    return c.text('Internal server error', 500);
  }
});
