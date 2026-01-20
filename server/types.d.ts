export interface JsonRpcRequest {
  jsonrpc: '2.0';
  id: number;
  method: string;
  params: Record<string, unknown>;
}

export interface JsonRpcResponse<T = unknown> {
  jsonrpc: '2.0';
  id: number;
  result?: T;
  error?: {
    code: number;
    message: string;
  };
}

export interface StartLinkResult {
  deviceLinkUri: string;
}

export interface FinishLinkResult {
  account: string;
}

export interface UpdateGroupResult {
  groupId: string;
}

export interface AccountInfo {
  number: string;
  name?: string;
  uuid?: string;
}

export type ListAccountsResult = AccountInfo[];
