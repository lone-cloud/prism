import { ROUTES } from '@/constants/server';
import { createGroup, sendGroupMessage } from '@/modules/signal';
import { getGroupId, register, remove } from '@/modules/store';
import { formatAsSignalMessage, parseUnifiedPushRequest } from '@/modules/unifiedpush';

const handleMatrixNotify = async (req: Request) => {
  const message = await parseUnifiedPushRequest(req);
  const groupId = getGroupId(message.endpoint);

  if (!groupId) {
    return new Response('Endpoint not registered', { status: 404 });
  }

  const signalMessage = formatAsSignalMessage(message);
  await sendGroupMessage(groupId, signalMessage);

  return Response.json({ success: true });
};

const handleRegister = async (req: Request) => {
  const url = new URL(req.url);
  const endpointId = url.pathname.split('/')[2] ?? '';
  const { appName } = (await req.json()) as {
    appName: string;
    token?: string;
  };

  const groupId: string = getGroupId(endpointId) ?? (await createGroup(appName));

  if (!getGroupId(endpointId)) {
    register(endpointId, groupId, appName);
  }

  const proto = req.headers.get('x-forwarded-proto') || 'http';
  const host = req.headers.get('host') || 'localhost:8080';
  const baseUrl = `${proto}://${host}`;
  const endpoint = `${baseUrl}${ROUTES.MATRIX_NOTIFY}/${endpointId}`;

  return Response.json({ endpoint, gateway: 'matrix' });
};

const handleUnregister = async (req: Request) => {
  const url = new URL(req.url);
  const endpointId = url.pathname.split('/')[2] ?? '';
  remove(endpointId);
  return new Response(null, { status: 204 });
};

const handleDiscovery = () =>
  Response.json({
    unifiedpush: { version: 1 },
    gateway: 'matrix',
  });

export const unifiedPushRoutes = {
  [ROUTES.UP]: {
    GET: handleDiscovery,
  },

  [ROUTES.MATRIX_NOTIFY]: {
    POST: handleMatrixNotify,
  },

  [ROUTES.UP_INSTANCE]: {
    POST: handleRegister,
    DELETE: handleUnregister,
  },
};
