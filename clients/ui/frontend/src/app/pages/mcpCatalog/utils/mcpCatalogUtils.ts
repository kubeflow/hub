import type { McpCatalogFiltersState } from '~/app/pages/mcpCatalog/types/mcpCatalogFilterOptions';
import { BACKEND_TO_FRONTEND_FILTER_KEY, MCP_FILTER_KEYS } from '~/app/pages/mcpCatalog/const';
import type {
  McpDeploymentMode,
  McpEndpoints,
  McpSecurityIndicator,
} from '~/app/mcpServerCatalogTypes';
import { stringFiltersToFilterQuery } from '~/app/shared/components/catalog';

export const isMcpRemoteDeploymentMode = (mode?: McpDeploymentMode): boolean => mode === 'remote';

export const getMcpServerPrimaryEndpoint = (
  endpoints?: McpEndpoints | null,
): string | undefined => {
  if (!endpoints) {
    return undefined;
  }
  const http = endpoints.http?.trim();
  if (http) {
    return http;
  }
  const sse = endpoints.sse?.trim();
  if (sse) {
    return sse;
  }
  return undefined;
};

const SECURITY_INDICATOR_LABELS: Record<keyof McpSecurityIndicator, string> = {
  verifiedSource: 'Verified source',
  secureEndpoint: 'Secure endpoint',
  sast: 'SAST',
  readOnlyTools: 'Read only tools',
};

const SECURITY_INDICATOR_KEYS: (keyof McpSecurityIndicator)[] = [
  'verifiedSource',
  'secureEndpoint',
  'sast',
  'readOnlyTools',
];

export const getSecurityIndicatorLabels = (
  securityIndicators?: McpSecurityIndicator | null,
): string[] => {
  if (!securityIndicators) {
    return [];
  }
  return SECURITY_INDICATOR_KEYS.filter((key) => Boolean(securityIndicators[key])).map(
    (key) => SECURITY_INDICATOR_LABELS[key],
  );
};

export const hasMcpFiltersApplied = (
  filters: McpCatalogFiltersState,
  searchQuery: string,
): boolean => {
  if (searchQuery && searchQuery.trim().length > 0) {
    return true;
  }
  for (const key of MCP_FILTER_KEYS) {
    const value = filters[key];
    if (Array.isArray(value) && value.length > 0) {
      return true;
    }
  }
  return false;
};

const FRONTEND_TO_BACKEND_FILTER_KEY: Record<string, string> = Object.fromEntries(
  Object.entries(BACKEND_TO_FRONTEND_FILTER_KEY).map(([backend, frontend]) => [frontend, backend]),
);

export function mcpFiltersToFilterQuery(filters: McpCatalogFiltersState): string {
  return stringFiltersToFilterQuery(filters, FRONTEND_TO_BACKEND_FILTER_KEY);
}
