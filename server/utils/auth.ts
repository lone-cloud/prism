import { API_KEY } from '../constants/config';

export const checkAuth = (req: Request) => {
  if (!API_KEY) return null;

  const proto = req.headers.get('x-forwarded-proto') || 'http';
  const host = req.headers.get('host') || '';
  const isLocalhost = host.startsWith('localhost') || host.startsWith('127.0.0.1');

  if (proto !== 'https' && !isLocalhost) {
    return new Response('HTTPS required when API_KEY is configured', { status: 403 });
  }

  if (req.headers.get('authorization') !== `Bearer ${API_KEY}`) {
    return new Response(null, { status: 401 });
  }

  return null;
};
