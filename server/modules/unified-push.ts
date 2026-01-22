import { SUP_ENDPOINT_PREFIX } from '@/constants/config';

interface UnifiedPushMessage {
  endpoint: string;
  title?: string;
  body?: string;
  data?: Record<string, unknown>;
}

export const parseUnifiedPushRequest = async (req: Request) => {
  const url = new URL(req.url);
  const endpointId = url.pathname.split('/').pop() ?? '';

  let title: string | undefined;
  let body: string | undefined;
  let data: Record<string, unknown> | undefined;

  const contentType = req.headers.get('content-type') || '';

  if (contentType.includes('application/json')) {
    const json = (await req.json()) as Record<string, unknown>;
    title = json.title as string | undefined;
    body = json.body as string | undefined;
    data = (json.data as Record<string, unknown>) || json;
  } else {
    const text = await req.text();
    body = text;
  }

  return {
    endpoint: endpointId,
    title,
    body,
    data,
  };
};

export const formatAsSignalMessage = (msg: UnifiedPushMessage) => {
  const parts: string[] = [`${SUP_ENDPOINT_PREFIX}${msg.endpoint}]`];

  if (msg.title) {
    parts.push(`**${msg.title}**`);
  }

  if (msg.body) {
    parts.push(msg.body);
  }

  if (msg.data && Object.keys(msg.data).length > 0) {
    parts.push(JSON.stringify(msg.data, null, 2));
  }

  return parts.join('\n');
};
