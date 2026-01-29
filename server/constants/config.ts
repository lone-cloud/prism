export const PORT = Bun.env.PORT || 8080;
export const API_KEY = Bun.env.API_KEY;
export const VERBOSE_LOGGING = Bun.env.VERBOSE_LOGGING === 'true';
export const ALLOW_INSECURE_HTTP = Bun.env.ALLOW_INSECURE_HTTP === 'true';
export const RATE_LIMIT = Number.parseInt(Bun.env.RATE_LIMIT || '100', 10);

export const DEVICE_NAME = Bun.env.DEVICE_NAME || 'SUP';

export const SUP_ENDPOINT_PREFIX = `[${DEVICE_NAME}:`;

export const PROTON_IMAP_USERNAME = Bun.env.PROTON_IMAP_USERNAME;
export const PROTON_IMAP_PASSWORD = Bun.env.PROTON_IMAP_PASSWORD;
export const PROTON_BRIDGE_HOST = Bun.env.PROTON_BRIDGE_HOST || 'protonmail-bridge';
export const PROTON_BRIDGE_PORT = Number.parseInt(Bun.env.PROTON_BRIDGE_PORT || '143', 10);
export const PROTON_SUP_TOPIC = Bun.env.PROTON_SUP_TOPIC || 'Proton Mail';

export const IMAP_INBOX = 'INBOX';
export const IMAP_SEEN_FLAG = '\\Seen';
export const IMAP_RECONNECT_BASE_DELAY = 10000;
export const IMAP_MAX_RECONNECT_DELAY = 300000;

export const ENDPOINT_PREFIX_PROTON = 'proton-';
export const ENDPOINT_PREFIX_NTFY = 'ntfy-';
export const ENDPOINT_PREFIX_UP = 'up-';

export const ACTION_MARK_READ = 'mark-read';
