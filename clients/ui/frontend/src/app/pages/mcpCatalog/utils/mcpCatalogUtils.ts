import type { McpCatalogFiltersState } from '~/app/pages/mcpCatalog/types/mcpCatalogFilterOptions';
import { BACKEND_TO_FRONTEND_FILTER_KEY, MCP_FILTER_KEYS } from '~/app/pages/mcpCatalog/const';
import type {
  McpDeploymentMode,
  McpEndpoints,
  McpSecurityIndicator,
} from '~/app/mcpServerCatalogTypes';
import { hasFiltersApplied, stringFiltersToFilterQuery } from '~/app/shared/components/catalog';

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
): boolean => hasFiltersApplied(filters, MCP_FILTER_KEYS, searchQuery);

const FRONTEND_TO_BACKEND_FILTER_KEY: Record<string, string> = Object.fromEntries(
  Object.entries(BACKEND_TO_FRONTEND_FILTER_KEY).map(([backend, frontend]) => [frontend, backend]),
);

export function mcpFiltersToFilterQuery(filters: McpCatalogFiltersState): string {
  return stringFiltersToFilterQuery(filters, FRONTEND_TO_BACKEND_FILTER_KEY);
}

type McpServerJsonRemote = {
  type: string;
  url: string;
};

const DISPLAY_SERVER_JSON_KEYS = [
  '$schema',
  'name',
  'description',
  'title',
  'version',
  'websiteUrl',
] as const;

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

const getRemotesFromPackages = (packages: unknown): McpServerJsonRemote[] => {
  if (!Array.isArray(packages)) {
    return [];
  }

  return packages.flatMap((pkg) => {
    if (!isRecord(pkg)) {
      return [];
    }
    const { transport } = pkg;
    if (!isRecord(transport)) {
      return [];
    }
    const { type, url } = transport;
    if (typeof type !== 'string' || typeof url !== 'string' || !type || !url) {
      return [];
    }
    return [{ type, url }];
  });
};

const getExistingRemotes = (remotes: unknown): McpServerJsonRemote[] => {
  if (!Array.isArray(remotes)) {
    return [];
  }

  return remotes.flatMap((remote) => {
    if (!isRecord(remote)) {
      return [];
    }
    const { type, url } = remote;
    if (typeof type !== 'string' || typeof url !== 'string' || !type || !url) {
      return [];
    }
    return [{ type, url }];
  });
};

/** Maps opaque catalog serverJson into the compact remotes display shape. */
export const toDisplayServerJson = (
  serverJson: Record<string, unknown>,
): Record<string, unknown> => {
  const display: Record<string, unknown> = {};

  DISPLAY_SERVER_JSON_KEYS.forEach((key) => {
    if (key in serverJson) {
      display[key] = serverJson[key];
    }
  });

  const remotesFromField = getExistingRemotes(serverJson.remotes);
  const remotes =
    remotesFromField.length > 0 ? remotesFromField : getRemotesFromPackages(serverJson.packages);

  if (remotes.length > 0) {
    display.remotes = remotes;
  }

  return display;
};
