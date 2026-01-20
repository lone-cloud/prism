import { timingSafeEqual } from 'node:crypto';
import { API_KEY } from '@/constants/config';
import { logWarn } from '@/utils/log';

// Rate limiing: track failed auth attempts per IP
const failedAttempts = new Map<string, { count: number; resetAt: number }>();
const MAX_FAILED_ATTEMPTS = 5;
const LOCKOUT_DURATION = 15 * 60 * 1000; // 15 minutes

// Cloudflare Tunnel sends real IP in CF-Connecting-IP
const getClientIP = (req: Request) =>
  req.headers.get('cf-connecting-ip') ||
  req.headers.get('x-forwarded-for')?.split(',')[0]?.trim() ||
  req.headers.get('x-real-ip') ||
  'unknown';

const isRateLimited = (ip: string) => {
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
    return new Response('Unauthorized', { status: 401 });
  }

  const clientIP = getClientIP(req);

  if (isRateLimited(clientIP)) {
    return new Response('Too many failed attempts. Try again later.', { status: 429 });
  }

  const proto = req.headers.get('x-forwarded-proto') || 'http';
  const host = req.headers.get('host') || '';
  const isLocalhost = host.startsWith('localhost') || host.startsWith('127.0.0.1');

  if (proto !== 'https' && !isLocalhost) {
    return new Response('HTTPS required when API_KEY is configured', { status: 403 });
  }

  const authHeader = req.headers.get('authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    recordFailedAttempt(clientIP);
    return new Response(null, { status: 401 });
  }

  const providedKey = authHeader.slice(7);

  if (providedKey.length !== API_KEY.length) {
    recordFailedAttempt(clientIP);
    return new Response(null, { status: 401 });
  }

  const providedBuffer = Buffer.from(providedKey);
  const keyBuffer = Buffer.from(API_KEY);

  if (!timingSafeEqual(providedBuffer, keyBuffer)) {
    recordFailedAttempt(clientIP);
    return new Response(null, { status: 401 });
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
