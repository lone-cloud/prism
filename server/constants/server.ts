export const ROUTES = {
  HEALTH: '/health',
  LINK: '/link',
  LINK_QR: '/link/qr',
  LINK_STATUS: '/link/status',
  LINK_UNLINK: '/link/unlink',
  FAVICON: '/favicon.png',
  MATRIX_NOTIFY: '/_matrix/push/v1/notify',
  UP: '/up',
  UP_INSTANCE: '/up/:instance',
  ENDPOINTS: '/endpoints',
  NOTIFY_TOPIC: '/notify/:topic',
  TOPICS: '/topics',
} as const;

export const TEMPLATES = {
  LINKED: 'templates/linked.html',
  LINK: 'templates/link.html',
  SETUP: 'templates/setup.html',
} as const;
