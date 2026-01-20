import { API_KEY } from '@/constants/config';

const checkAuth = (req: Request) => {
  if (!API_KEY) {
    return new Response('Unauthorized', { status: 401 });
  }

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

export const withAuth =
  <T extends unknown[]>(handler: (req: Request, ...args: T) => Response | Promise<Response>) =>
  (req: Request, ...args: T) => {
    const auth = checkAuth(req);
    if (auth) return auth;
    return handler(req, ...args);
  };
