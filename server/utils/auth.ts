import { timingSafeEqual } from 'node:crypto';
import { networkInterfaces } from 'node:os';
import type { Context } from 'hono';
import { getConnInfo } from 'hono/bun';
import { ALLOW_INSECURE_HTTP, API_KEY } from '@/constants/config';

export const isLocalIP = (addr: string | undefined) => {
  if (!addr || addr === '::1' || addr === 'localhost') return true;

  const octets = addr.split('.').map(Number);
  if (octets.length !== 4 || octets.some((n) => Number.isNaN(n))) return false;

  const [a, b] = octets;

  return (
    a === 127 ||
    a === 10 ||
    (a === 192 && b === 168) ||
    (a === 172 && b !== undefined && b >= 16 && b <= 31)
  );
};

export const verifyApiKey = (password: string, c?: Context) => {
  if (!API_KEY || password.length !== API_KEY.length) return false;

  if (c && !ALLOW_INSECURE_HTTP) {
    const addr = getConnInfo(c).remote.address;
    const proto = c.req.header('x-forwarded-proto') || 'http';

    if (proto !== 'https' && !isLocalIP(addr)) {
      return false;
    }
  }

  const providedBuffer = Buffer.from(password);
  const keyBuffer = Buffer.from(API_KEY);

  return timingSafeEqual(providedBuffer, keyBuffer);
};

export const getLanIP = () => {
  const nets = networkInterfaces();

  for (const name of Object.keys(nets)) {
    const interfaces = nets[name];

    if (!interfaces) continue;

    for (const iface of interfaces) {
      if (iface.family === 'IPv4' && !iface.internal) {
        return iface.address;
      }
    }
  }

  return null;
};
