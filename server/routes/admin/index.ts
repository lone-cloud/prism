import { ROUTES } from '@/constants/server';
import { withAuth } from '@/utils/auth';
import {
  handleEndpointsFragment,
  handleHealthFragment,
  handleLinkStatusCheck,
  handleQRImage,
  handleQRSection,
  handleSignalInfoFragment,
} from './fragments';

export const adminRoutes = {
  [ROUTES.ADMIN]: {
    GET: () => new Response(Bun.file('public/admin.html')),
  },

  [ROUTES.ADMIN_CSS]: Bun.file('public/admin.css'),

  [ROUTES.HEALTH_FRAGMENT]: {
    GET: withAuth(handleHealthFragment),
  },

  [ROUTES.SIGNAL_INFO_FRAGMENT]: {
    GET: withAuth(handleSignalInfoFragment),
  },

  [ROUTES.ENDPOINTS_FRAGMENT]: {
    GET: withAuth(handleEndpointsFragment),
  },

  [ROUTES.QR_SECTION]: {
    GET: withAuth(handleQRSection),
  },

  [ROUTES.QR_IMAGE]: {
    GET: withAuth(handleQRImage),
  },

  [ROUTES.STATUS_CHECK]: {
    GET: withAuth(handleLinkStatusCheck),
  },
};
