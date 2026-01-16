import { CONTENT_TYPE, ROUTES, TEMPLATES } from '../constants/server';
import { finishLink, generateLinkQR, hasValidAccount, initSignal, unlinkDevice } from '../signal';

export const handleLink = async () => {
  const linked = await hasValidAccount();
  const template = linked ? TEMPLATES.LINKED : TEMPLATES.LINK;
  let html = await Bun.file(template).text();

  if (linked && Bun.env.API_KEY) {
    const passwordField =
      '<input type="password" name="password" placeholder="Enter API_KEY" required style="padding: 8px; margin-right: 10px; border: 1px solid #ccc; border-radius: 4px;" />';
    html = html.replace('{{PASSWORD_FIELD}}', passwordField);
  } else if (linked) {
    html = html.replace('{{PASSWORD_FIELD}}', '');
  }

  return new Response(html, {
    headers: { 'content-type': CONTENT_TYPE.HTML },
  });
};

export const handleLinkQR = async () => {
  const qrDataUrl = await generateLinkQR();
  return new Response(qrDataUrl, {
    headers: { 'content-type': CONTENT_TYPE.TEXT },
  });
};

export const handleLinkStatus = async () => {
  let linked = await hasValidAccount();

  if (!linked) {
    try {
      await finishLink();
      await initSignal({});
      linked = true;
    } catch {
      // Not ready yet or failed
    }
  }

  return Response.json({ linked });
};

export const handleUnlink = async (req: Request, daemon: ReturnType<typeof Bun.spawn> | null) => {
  const API_KEY = Bun.env.API_KEY;

  if (API_KEY) {
    const formData = await req.formData();
    const password = formData.get('password');

    if (password !== API_KEY) {
      return new Response(null, { status: 403 });
    }
  }

  await unlinkDevice();

  if (daemon) {
    daemon.kill();
  }

  await new Promise((resolve) => setTimeout(resolve, 500));

  return new Response('', {
    status: 303,
    headers: { Location: ROUTES.LINK },
  });
};
