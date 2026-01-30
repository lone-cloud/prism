import { Database } from 'bun:sqlite';
import { SUP_DB } from '@/constants/paths';
import type { NotificationChannel } from '@/types/notifications';

type EndpointMapping = {
  endpoint: string;
  appName: string;
  channel: NotificationChannel;
  groupId: string | null;
  upEndpoint: string | null;
};

const db = new Database(SUP_DB);

db.run(`
  CREATE TABLE IF NOT EXISTS mappings (
    endpoint TEXT PRIMARY KEY,
    groupId TEXT,
    appName TEXT NOT NULL,
    channel TEXT NOT NULL DEFAULT 'signal',
    upEndpoint TEXT
  )
`);

export function register(
  endpoint: string,
  appName: string,
  channel: 'signal',
  options?: { groupId?: string },
): void;
export function register(
  endpoint: string,
  appName: string,
  channel: 'webhook',
  options: { upEndpoint: string },
): void;
export function register(
  endpoint: string,
  appName: string,
  channel: NotificationChannel,
  options: { groupId?: string; upEndpoint?: string } = {},
) {
  const { groupId = null, upEndpoint = null } = options;
  db.run(
    'INSERT OR IGNORE INTO mappings (endpoint, groupId, appName, channel, upEndpoint) VALUES (?, ?, ?, ?, ?)',
    [endpoint, groupId, appName, channel, upEndpoint],
  );
}

export const getMapping = (endpoint: string) => {
  const row = db
    .query(
      'SELECT endpoint, groupId, appName, channel, upEndpoint FROM mappings WHERE endpoint = ?',
    )
    .get(endpoint) as EndpointMapping | undefined;

  return row;
};

export const getAllMappings = () =>
  db
    .query('SELECT endpoint, groupId, appName, channel, upEndpoint FROM mappings')
    .all() as EndpointMapping[];

export const updateChannel = (endpoint: string, channel: NotificationChannel) =>
  db.run('UPDATE mappings SET channel = ? WHERE endpoint = ?', [channel, endpoint]);

export const removeEndpoint = (endpoint: string) =>
  db.run('DELETE FROM mappings WHERE endpoint = ?', [endpoint]);
