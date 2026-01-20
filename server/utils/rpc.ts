import { SIGNAL_CLI_SOCKET } from '@/constants/paths';

const MESSAGE_DELIMITER = '\n';
let rpcId = 1;

export const call = (method: string, params: Record<string, unknown>, account: string | null) =>
  new Promise((resolve, reject) => {
    let response = '';

    Bun.connect({
      unix: SIGNAL_CLI_SOCKET,
      socket: {
        data(socket, data) {
          response += new TextDecoder().decode(data);

          const isComplete = response.includes(MESSAGE_DELIMITER);
          if (isComplete) {
            socket.end();

            const parsed = JSON.parse(response.trim());

            if (parsed.error) {
              reject(new Error(`signal-cli RPC error: ${parsed.error.message}`));
            } else {
              resolve(parsed.result);
            }
          }
        },
        error(_socket, error) {
          reject(error);
        },
        connectError(_socket, error) {
          reject(error);
        },
        close() {
          const isComplete = response.includes(MESSAGE_DELIMITER);
          if (!isComplete) {
            reject(new Error('Connection closed before response received'));
          }
        },
        open(socket) {
          const request = JSON.stringify({
            jsonrpc: '2.0',
            id: rpcId++,
            method,
            params: account ? { account, ...params } : params,
          });

          socket.write(`${request}${MESSAGE_DELIMITER}`);
        },
      },
    });
  });
