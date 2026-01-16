import { ROUTES } from '../constants/server';
import { createGroup, sendGroupMessage } from '../signal';
import { getAllMappings, getGroupId, register, remove } from '../store';
import { formatAsSignalMessage, parseUnifiedPushRequest } from '../unifiedpush';

export const handleMatrixNotify = async (req: Request) => {
  const message = await parseUnifiedPushRequest(req);
  const groupId = getGroupId(message.endpoint);

  if (!groupId) {
    return new Response('Endpoint not registered', { status: 404 });
  }

  const signalMessage = formatAsSignalMessage(message);
  await sendGroupMessage(groupId, signalMessage);

  return Response.json({ success: true });
};

export const handleRegister = async (req: Request, url: URL) => {
  const API_KEY = Bun.env.API_KEY;

  if (API_KEY && req.headers.get('authorization') !== `Bearer ${API_KEY}`) {
    return new Response(null, { status: 401 });
  }

  const endpointId = url.pathname.split('/')[2] ?? '';
  const { appName } = (await req.json()) as {
    appName: string;
    token?: string;
  };

  const groupId: string = getGroupId(endpointId) ?? (await createGroup(`SUP - ${appName}`));

  if (!getGroupId(endpointId)) {
    register(endpointId, groupId, appName);
  }

  const proto = req.headers.get('x-forwarded-proto') || 'http';
  const host = req.headers.get('host') || 'localhost:8080';
  const baseUrl = `${proto}://${host}`;
  const endpoint = `${baseUrl}${ROUTES.MATRIX_NOTIFY}/${endpointId}`;

  return Response.json({ endpoint, gateway: 'matrix' });
};

export const handleUnregister = async (url: URL) => {
  const endpointId = url.pathname.split('/')[2] ?? '';
  remove(endpointId);
  return new Response(null, { status: 204 });
};

export const handleDiscovery = () =>
  Response.json({
    unifiedpush: { version: 1 },
    gateway: 'matrix',
  });

export const handleEndpoints = () => Response.json(getAllMappings());
