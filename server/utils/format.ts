export const formatUptime = (seconds: number): string => {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);

  const parts = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0) parts.push(`${minutes}m`);
  if (secs > 0 || parts.length === 0) parts.push(`${secs}s`);

  return parts.join(' ');
};

export const formatPhoneNumber = (phone: string): string => {
  if (!phone) return phone;

  // US/Canada: +1 (234) 567-8901
  if (phone.startsWith('+1') && phone.length === 12) {
    return `+1 (${phone.slice(2, 5)}) ${phone.slice(5, 8)}-${phone.slice(8)}`;
  }

  // International: +XX XXXX XXXX
  if (phone.startsWith('+')) {
    const match = phone.match(/^\+(\d{1,3})(\d{3,4})(\d+)$/);
    if (match) {
      const [, code, first, rest] = match;
      const parts = rest?.match(/.{1,4}/g) || [];
      return `+${code} ${first} ${parts.join(' ')}`;
    }
  }

  return phone;
};
