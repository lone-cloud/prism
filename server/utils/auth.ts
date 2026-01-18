import { API_KEY } from '../constants/config';

const checkAuth = (req: Request) => {
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

export const checkFormAuth = async (req: Request) => {
  const bearerAuth = checkAuth(req);
  if (!bearerAuth) return null;

  // Fall back to form password
  const body = await req.text();
  const params = new URLSearchParams(body);
  if (params.get('password') === API_KEY) {
    return null;
  }

  return new Response(null, { status: 403 });
};

export const withAuth =
  <T extends unknown[]>(handler: (req: Request, ...args: T) => Response | Promise<Response>) =>
  (req: Request, ...args: T) => {
    const auth = checkAuth(req);
    if (auth) return auth;
    return handler(req, ...args);
  };

export const withFormAuth =
  <T extends unknown[]>(handler: (req: Request, ...args: T) => Response | Promise<Response>) =>
  async (req: Request, ...args: T) => {
    const auth = await checkFormAuth(req);
    if (auth) return auth;
    return handler(req, ...args);
  };
