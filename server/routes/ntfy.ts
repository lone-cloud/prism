import { createGroup, sendGroupMessage } from '@/modules/signal';
import { getGroupId, register } from '@/modules/store';
import { withAuth } from '@/utils/auth';
import { logError, logVerbose } from '@/utils/log';

const handleNtfyPublish = async (req: Request) => {
  try {
    const url = new URL(req.url);
    const topic = url.pathname.slice(1);

    if (!topic || topic.includes('/')) {
      return new Response('Invalid topic', { status: 400 });
    }

    const body = await req.text();
    if (!body) {
      return new Response('Message required', { status: 400 });
    }

    const title = req.headers.get('X-Title') || req.headers.get('Title') || req.headers.get('t');

    const topicKey = `ntfy-${topic}`;
    let groupId = getGroupId(topicKey);

    if (!groupId) {
      groupId = await createGroup(topic);
      register(topicKey, groupId, topic);
      logVerbose(`Created new group for ntfy topic: ${topic}`);
    }

    const message = title ? `**${title}**\n${body}` : body;

    await sendGroupMessage(groupId, message);

    logVerbose(`Sent ntfy message to topic ${topic}: ${title || body.substring(0, 50)}`);

    return new Response(
      JSON.stringify({
        id: Date.now().toString(),
        time: Math.floor(Date.now() / 1000),
        event: 'message',
        topic,
        message: body,
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
