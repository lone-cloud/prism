import { checkSignalCli, hasValidAccount } from '../modules/signal';

export const handleHealth = async () => {
  const signalOk = await checkSignalCli();
  const linked = signalOk && (await hasValidAccount());

  return Response.json({
    status: 'ok',
    signal: signalOk ? 'connected' : 'disconnected',
    linked,
    mode: 'daemon',
  });
};
