import { sendGroupMessage } from '@/modules/signal';
import { getOrCreateGroup } from '@/modules/store';
import { withAuth } from '@/utils/auth';
import { logError, logVerbose } from '@/utils/log';

const handleNtfyPublish = async (req: Request) => {
  try {
    const url = new URL(req.url);
    const topic = decodeURIComponent(url.pathname.slice(1));

    if (!topic || topic.includes('/')) {
      return new Response('Invalid topic', { status: 400 });
    }

    let message = await req.text();
    if (!message) {
      return new Response('Message required', { status: 400 });
    }

    const contentType = req.headers.get('content-type') || '';
    let title: string | undefined =
      req.headers.get('X-Title') || req.headers.get('Title') || req.headers.get('t') || undefined;
    let androidPackage: string | undefined =
      req.headers.get('X-Package') ||
      req.headers.get('Package') ||
      req.headers.get('p') ||
      undefined;

    if (contentType.includes('application/x-www-form-urlencoded')) {
      const params = new URLSearchParams(message);
      message = params.get('message') || message;
      title = title || params.get('title') || params.get('t') || undefined;
      androidPackage = androidPackage || params.get('package') || params.get('p') || undefined;
    }

    if (title === topic) title = undefined;

    const groupId = await getOrCreateGroup(`ntfy-${topic}`, topic);

    await sendGroupMessage(groupId, message, {
      androidPackage: androidPackage || undefined,
      title: title || undefined,
    });

    logVerbose(`Sent ntfy message to topic ${topic}: ${title || message.substring(0, 50)}`);

    return new Response(
      JSON.stringify({
        id: Date.now().toString(),
        time: Math.floor(Date.now() / 1000),
        event: 'message',
        topic,
        message,
      }),
      {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      },
    );
  } catch (error) {
    logError('Failed to handle ntfy publish:', error);
    return new Response('Internal server error', { status: 500 });
  }
};

export const ntfyRoutes = {
  '/:topic': {
    POST: withAuth(handleNtfyPublish),
  },
};
