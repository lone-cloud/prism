export const PORT = Bun.env.PORT || 8080;
export const API_KEY = Bun.env.API_KEY;
export const VERBOSE = Bun.env.VERBOSE === 'true';
export const ALLOW_INSECURE_HTTP = Bun.env.ALLOW_INSECURE_HTTP === 'true';

export const DEVICE_NAME = 'SUP';

export const SUP_ENDPOINT_PREFIX = `[${DEVICE_NAME}:`;
export const LAUNCH_ENDPOINT_PREFIX = '[LAUNCH:';

export const BRIDGE_IMAP_USERNAME = Bun.env.BRIDGE_IMAP_USERNAME;
export const BRIDGE_IMAP_PASSWORD = Bun.env.BRIDGE_IMAP_PASSWORD;
export const PROTON_BRIDGE_HOST = Bun.env.PROTON_BRIDGE_HOST || 'protonmail-bridge';
export const PROTON_BRIDGE_PORT = Number.parseInt(Bun.env.PROTON_BRIDGE_PORT || '143', 10);
export const SUP_TOPIC = Bun.env.SUP_TOPIC || 'Proton Mail';
export const ENABLE_ANDROID_INTEGRATION = Bun.env.ENABLE_ANDROID_INTEGRATION === 'true';
