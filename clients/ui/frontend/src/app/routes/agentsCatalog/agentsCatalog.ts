export const agentsCatalogUrl = (): string => '/agents-catalog';

export const getAgentsCatalogDetailsRoute = (agentId: string): string =>
  `${agentsCatalogUrl()}/${encodeURIComponent(agentId)}/overview`;
