export const ROUTES = {
  HEALTH: '/health',
  LINK: '/link',
  LINK_QR: '/link/qr',
  LINK_STATUS: '/link/status',
  LINK_UNLINK: '/link/unlink',
  FAVICON: '/favicon.png',
  MATRIX_NOTIFY: '/_matrix/push/v1/notify',
  UP: '/up',
  UP_PREFIX: '/up/',
  ENDPOINTS: '/endpoints',
  NOTIFY_PREFIX: '/notify/',
  TOPICS: '/topics',
  NOTIFICATIONS: '/notifications',
} as const;

export const CONTENT_TYPE = {
  HTML: 'text/html',
  JSON: 'application/json',
  TEXT: 'text/plain',
} as const;

export const TEMPLATES = {
  LINKED: 'server/templates/linked.html',
  LINK: 'server/templates/link.html',
  SETUP: 'server/templates/setup.html',
} as const;
