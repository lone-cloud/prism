import { createGroup, sendGroupMessage } from '../modules/signal';
import { getAllMappings, getGroupId, register } from '../modules/store';

interface NotificationMessage {
  topic: string;
  title?: string;
  message: string;
}

const formatNotification = (notification: NotificationMessage) => {
  const parts: string[] = [];
  const title = notification.title || notification.topic;

  parts.push(`**${title}**`);
  parts.push(notification.message);

  return parts.join('\n');
};

export const handleNotify = async (req: Request, url: URL) => {
  const topic = url.pathname.split('/')[2];
  if (!topic) {
    return new Response('Topic required', { status: 400 });
  }

  const body = await req.text();
  if (!body) {
    return new Response('Message required', { status: 400 });
  }

  const title = req.headers.get('x-title') || undefined;

  const notification: NotificationMessage = {
    topic,
    title,
    message: body,
  };

  const topicKey = `notify-${topic}`;
  const groupId = getGroupId(topicKey) ?? (await createGroup(topic));

  if (!getGroupId(topicKey)) {
    register(topicKey, groupId, topic);
  }

  const signalMessage = formatNotification(notification);
  await sendGroupMessage(groupId, signalMessage);

  return Response.json({ success: true, topic, groupId });
};

export const handleTopics = () => {
  const allMappings = getAllMappings();
  const topics = allMappings
    .filter((m) => m.endpoint.startsWith('notify-'))
    .map((m) => ({
      topic: m.appName,
      groupId: m.groupId,
    }));

  return Response.json({ topics });
};
