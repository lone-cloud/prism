import { Hono } from 'hono';
import { basicAuth } from 'hono/basic-auth';
import { ENDPOINT_PREFIX_UP } from '@/constants/config';
import { register } from '@/modules/store';
import { verifyApiKey } from '@/utils/auth';
import { logError, logVerbose } from '@/utils/log';

export const unifiedPush = new Hono();

unifiedPush.use(
  '*',
  basicAuth({
    verifyUser: (_, password) => verifyApiKey(password),
    realm: 'SUP UnifiedPush - Username: any, Password: API_KEY',
  }),
);

unifiedPush.post('/up/register', async (c) => {
  try {
    const body = await c.req.json<{
      appName: string;
      upEndpoint: string;
    }>();

    const { appName, upEndpoint } = body;

    if (!appName || !upEndpoint) {
      return c.json({ error: 'appName and upEndpoint are required' }, 400);
    }

    try {
      new URL(upEndpoint);
    } catch {
      return c.json({ error: 'Invalid upEndpoint URL' }, 400);
    }

    const endpoint = `${ENDPOINT_PREFIX_UP}${appName}`;

    register(endpoint, appName, 'unifiedpush', { upEndpoint });

    logVerbose(`Registered UP endpoint for ${appName}: ${upEndpoint}`);

    return c.json({
      endpoint,
      appName,
      channel: 'unifiedpush',
    });
  } catch (error) {
    logError('Failed to register UP endpoint:', error);

    return c.json({ error: 'Internal server error' }, 500);
  }
});
