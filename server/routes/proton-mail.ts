import { Hono } from 'hono';
import { basicAuth } from 'hono/basic-auth';
import { markEmailAsRead } from '@/modules/proton-mail';
import { verifyApiKey } from '@/utils/auth';
import { logError, logVerbose } from '@/utils/log';

export const protonMail = new Hono();

protonMail.use(
  '*',
  basicAuth({
    verifyUser: (_, password, c) => verifyApiKey(password, c),
    realm: 'SUP Proton Mail - Username: any, Password: API_KEY',
  }),
);

protonMail.post('/api/proton-mail/mark-read', async (c) => {
  try {
    const body = await c.req.json();
    const { uid } = body;

    if (!uid || typeof uid !== 'number') {
      return c.json({ error: 'uid (number) is required' }, 400);
    }

    const result = await markEmailAsRead(uid);

    if (!result.success) {
      return c.json({ error: result.error }, 500);
    }

    logVerbose(`Marked email as read: UID ${uid}`);

    return c.json({ success: true });
  } catch (error) {
    logError('Failed to mark email as read:', error);
    return c.json({ error: 'Internal server error' }, 500);
  }
});
