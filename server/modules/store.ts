import { Database } from 'bun:sqlite';
import { SUP_DB } from '@/constants/paths';

interface EndpointMapping {
  endpoint: string;
  groupId: string;
  appName: string;
}

const db = new Database(SUP_DB);

db.run(`
  CREATE TABLE IF NOT EXISTS mappings (
    endpoint TEXT PRIMARY KEY,
    groupId TEXT NOT NULL,
    appName TEXT NOT NULL
  )
`);

export const register = (endpoint: string, groupId: string, appName: string) => {
  db.run('INSERT OR REPLACE INTO mappings (endpoint, groupId, appName) VALUES (?, ?, ?)', [
    endpoint,
    groupId,
    appName,
  ]);
};

export const getGroupId = (endpoint: string) => {
  const row = db.query('SELECT groupId FROM mappings WHERE endpoint = ?').get(endpoint) as
    | { groupId: string }
    | undefined;
  return row?.groupId;
};

export const getAppName = (endpoint: string) => {
  const row = db.query('SELECT appName FROM mappings WHERE endpoint = ?').get(endpoint) as
    | { appName: string }
    | undefined;
  return row?.appName;
};

export const getAllMappings = () =>
  db.query('SELECT endpoint, groupId, appName FROM mappings').all() as EndpointMapping[];

export const remove = (endpoint: string) => {
  db.run('DELETE FROM mappings WHERE endpoint = ?', [endpoint]);
};
