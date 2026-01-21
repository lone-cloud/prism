import { timingSafeEqual } from 'node:crypto';
import { ALLOW_INSECURE_HTTP, API_KEY } from '@/constants/config';
import { getClientIP, isLocalIP } from '@/utils/ip';
import { logWarn } from '@/utils/log';

// Rate limiting: track failed auth attempts per IP
const failedAttempts = new Map<string, { count: number; resetAt: number }>();
const MAX_FAILED_ATTEMPTS = 5;
const LOCKOUT_DURATION = 15 * 60 * 1000; // 15 minutes

const isRateLimited = (ip: string) => {
  if (ALLOW_INSECURE_HTTP || isLocalIP(ip)) return false;

  const now = Date.now();
  const record = failedAttempts.get(ip);

  if (!record) return false;

  if (now > record.resetAt) {
    failedAttempts.delete(ip);
    return false;
  }

  return record.count >= MAX_FAILED_ATTEMPTS;
};

const recordFailedAttempt = (ip: string) => {
  if (ALLOW_INSECURE_HTTP || isLocalIP(ip)) return;

  const now = Date.now();
  const record = failedAttempts.get(ip);

  if (!record || now > record.resetAt) {
    failedAttempts.set(ip, { count: 1, resetAt: now + LOCKOUT_DURATION });
  } else {
    record.count++;
    if (record.count >= MAX_FAILED_ATTEMPTS) {
      logWarn(`IP ${ip} locked out after ${MAX_FAILED_ATTEMPTS} failed auth attempts`);
    }
  }
};

const checkAuth = (req: Request) => {
  if (!API_KEY) {
    return new Response('API_KEY environment variable not configured', { status: 401 });
  }

  const clientIP = getClientIP(req);

  if (isRateLimited(clientIP)) {
    return new Response('Too many failed attempts. Try again later.', { status: 429 });
  }

  if (!ALLOW_INSECURE_HTTP) {
    const proto = req.headers.get('x-forwarded-proto') || 'http';
    if (proto !== 'https') {
      return new Response('HTTPS required when API_KEY is configured', {
        status: 426,
        headers: { Upgrade: 'TLS/1.2, HTTP/1.1' },
      });
    }
  }

  const authHeader = req.headers.get('authorization');
  if (!authHeader || !authHeader.startsWith('Basic ')) {
    recordFailedAttempt(clientIP);
    return new Response(null, {
      status: 401,
      headers: { 'WWW-Authenticate': 'Basic realm="SUP Admin - Username: any, Password: API_KEY"' },
    });
  }

  const base64Credentials = authHeader.slice(6);
  const credentials = Buffer.from(base64Credentials, 'base64').toString('utf-8');
  const [, password] = credentials.split(':');
  const providedKey = password || '';

  if (providedKey.length !== API_KEY.length) {
    recordFailedAttempt(clientIP);
    return new Response(null, {
      status: 401,
      headers: { 'WWW-Authenticate': 'Basic realm="SUP Admin - Username: any, Password: API_KEY"' },
    });
  }

  const providedBuffer = Buffer.from(providedKey);
  const keyBuffer = Buffer.from(API_KEY);

  if (!timingSafeEqual(providedBuffer, keyBuffer)) {
    recordFailedAttempt(clientIP);
    return new Response(null, {
      status: 401,
      headers: { 'WWW-Authenticate': 'Basic realm="SUP Admin - Username: any, Password: API_KEY"' },
    });
  }

  failedAttempts.delete(clientIP);
  return null;
};

export const withAuth =
  <T extends unknown[]>(handler: (req: Request, ...args: T) => Response | Promise<Response>) =>
  (req: Request, ...args: T) => {
    const auth = checkAuth(req);
    if (auth) return auth;
    return handler(req, ...args);
  };
