interface EndpointMapping {
  endpoint: string;
  groupId: string;
  appName: string;
}

const mappings = new Map<string, EndpointMapping>();

export const register = (endpoint: string, groupId: string, appName: string) => {
  mappings.set(endpoint, { endpoint, groupId, appName });
};

export const getGroupId = (endpoint: string) => mappings.get(endpoint)?.groupId;

export const getAppName = (endpoint: string) => mappings.get(endpoint)?.appName;

export const getAllMappings = () => Array.from(mappings.values());

export const remove = (endpoint: string) => {
  mappings.delete(endpoint);
};
