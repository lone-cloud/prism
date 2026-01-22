import { Hono } from 'hono';
import { sendGroupMessage } from '@/modules/signal';
import { getGroupId, getOrCreateGroup, remove } from '@/modules/store';
import { formatAsSignalMessage, parseUnifiedPushRequest } from '@/modules/unified-push';

export const unifiedpush = new Hono();

unifiedpush.get('/up', (c) =>
  c.json({
    unifiedpush: { version: 1 },
    gateway: 'matrix',
  }),
);

unifiedpush.post('/_matrix/push/v1/notify', async (c) => {
  const message = await parseUnifiedPushRequest(c.req.raw);
  const groupId = getGroupId(message.endpoint);

  if (!groupId) {
    return c.text('Endpoint not registered', 404);
  }

  const signalMessage = formatAsSignalMessage(message);
  await sendGroupMessage(groupId, signalMessage);

  return c.json({ success: true });
});

unifiedpush.post('/up/:instance', async (c) => {
  const endpointId = c.req.param('instance');
  const { appName } = await c.req.json<{ appName: string; token?: string }>();

  await getOrCreateGroup(endpointId, appName);

  const proto = c.req.header('x-forwarded-proto') || 'http';
  const host = c.req.header('host') || 'localhost:8080';
  const baseUrl = `${proto}://${host}`;
  const endpoint = `${baseUrl}/_matrix/push/v1/notify/${endpointId}`;

  return c.json({ endpoint, gateway: 'matrix' });
});

unifiedpush.delete('/up/:instance', async (c) => {
  const endpointId = c.req.param('instance');
  remove(endpointId);
  return c.body(null, 204);
});
