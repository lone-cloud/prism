import { networkInterfaces } from 'node:os';

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

export const getClientIP = (req: Request) => {
  // Cloudflare Tunnel
  const cfIP = req.headers.get('cf-connecting-ip');
  if (cfIP) return cfIP;

  // Standard reverse proxy headers
  const forwardedFor = req.headers.get('x-forwarded-for')?.split(',')[0]?.trim();
  if (forwardedFor) return forwardedFor;

  const realIP = req.headers.get('x-real-ip');
  if (realIP) return realIP;

  const remoteAddr = req.headers.get('remote-addr');
  if (remoteAddr) return remoteAddr;

  return 'unknown';
};

export const isLocalIP = (ip: string) => {
  if (ip === '::1' || ip === 'localhost') return true;

  const octets = ip.split('.').map(Number);
  if (octets.length !== 4 || octets.some((n) => Number.isNaN(n))) return false;

  const [a, b] = octets;
  return (
    a === 127 ||
    a === 10 ||
    (a === 192 && b === 168) ||
    (a === 172 && b !== undefined && b >= 16 && b <= 31)
  );
};
