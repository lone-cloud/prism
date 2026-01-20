import { fragmentRoutes } from './fragments';

export const adminRoutes = {
  '/': {
    GET: () => new Response(Bun.file('public/admin.html')),
  },

  '/admin.css': {
    GET: () => new Response(Bun.file('public/admin.css')),
  },

  ...fragmentRoutes,
};
