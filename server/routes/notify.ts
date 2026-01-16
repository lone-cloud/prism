import { createGroup, sendGroupMessage } from '../signal';
import { addNotification, getAllMappings, getGroupId, getNotifications, register } from '../store';

interface NotificationMessage {
  topic: string;
  title?: string;
  message: string;
  priority?: 'min' | 'low' | 'default' | 'high' | 'urgent';
  tags?: string;
  click?: string;
}

const formatNotification = (notification: NotificationMessage) => {
  const parts: string[] = [];
  const priorityEmoji = {
    min: '🔕',
    low: '🔉',
    default: '🔔',
    high: '⚠️',
    urgent: '🚨',
  };

  const emoji = priorityEmoji[notification.priority || 'default'];
  const title = notification.title || notification.topic;

  parts.push(`${emoji} **${title}**`);
  parts.push(notification.message);

  if (notification.tags) {
    parts.push(`\n_Tags: ${notification.tags}_`);
  }

  if (notification.click) {
    parts.push(`\n🔗 ${notification.click}`);
  }

  return parts.join('\n');
};

export const handleNotify = async (req: Request, url: URL) => {
  const API_KEY = Bun.env.API_KEY;

  if (API_KEY && req.headers.get('authorization') !== `Bearer ${API_KEY}`) {
    return new Response(null, { status: 401 });
  }

  const topic = url.pathname.split('/')[2];
  if (!topic) {
    return new Response('Topic required', { status: 400 });
  }

  const body = await req.text();
  if (!body) {
    return new Response('Message required', { status: 400 });
  }

  const title = req.headers.get('x-title') || undefined;
  const priority = (req.headers.get('x-priority') || 'default') as NotificationMessage['priority'];
  const tags = req.headers.get('x-tags') || undefined;
  const click = req.headers.get('x-click') || undefined;

  const notification: NotificationMessage = {
    topic,
    title,
    message: body,
    priority,
    tags,
    click,
  };

  // Get or create a Signal group for this topic
  const topicKey = `notify-${topic}`;
  const groupId = getGroupId(topicKey) ?? (await createGroup(`SUP - ${topic}`));

  if (!getGroupId(topicKey)) {
    register(topicKey, groupId, topic);
  }

  // Format and send message
  const signalMessage = formatNotification(notification);
  await sendGroupMessage(groupId, signalMessage);

  // Store notification
  const priorityValue = { min: 1, low: 2, default: 3, high: 4, urgent: 5 }[priority || 'default'];
  addNotification({
    topic,
    title,
    message: body,
    priority: priorityValue,
    tags: tags?.split(',').map((t) => t.trim()),
    click,
  });

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

export const handleGetNotifications = (_req: Request, url: URL) => {
  const topic = url.searchParams.get('topic') || undefined;
  const endpoint = url.searchParams.get('endpoint') || undefined;

  const notifications = getNotifications({ topic, endpoint });

  return Response.json({ notifications });
};
