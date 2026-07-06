import type { McpFilterCategoryKey } from '~/app/pages/mcpCatalog/types/mcpCatalogFilterOptions';

export const MCP_CATALOG_TITLE = 'MCP Catalog';
export const MCP_CATALOG_DESCRIPTION =
  'Browse and deploy MCP servers provided by Red Hat partners and other providers.';

export const MCP_CATALOG_GALLERY = {
  CARDS_PER_ROW: 4,
  PAGE_SIZE: 10,
  SECTION_TITLE: 'MCP Servers',
} as const;

type GridSpan = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

const GRID_COLUMNS = 12;
const GRID_SPAN_VALUES: GridSpan[] = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12];

function toGridSpan(cols: number): GridSpan {
  const index = Math.min(Math.max(0, cols - 1), GRID_SPAN_VALUES.length - 1);
  return GRID_SPAN_VALUES[index];
}

export const MCP_CATALOG_GRID_SPAN: {
  sm: GridSpan;
  md: GridSpan;
  lg: GridSpan;
  xl2: GridSpan;
} = {
  sm: toGridSpan(GRID_COLUMNS),
  md: toGridSpan(GRID_COLUMNS / 2),
  lg: toGridSpan(GRID_COLUMNS / MCP_CATALOG_GALLERY.CARDS_PER_ROW),
  xl2: toGridSpan(GRID_COLUMNS / MCP_CATALOG_GALLERY.CARDS_PER_ROW),
};

export const MCP_FILTER_CATEGORY_NAMES: Record<McpFilterCategoryKey, string> = {
  deploymentMode: 'Deployment mode',
  supportedTransports: 'Supported transports',
  license: 'License',
  labels: 'Labels',
  securityIndicators: 'Security & Verification',
};

export const MCP_FILTER_KEYS: McpFilterCategoryKey[] = [
  'deploymentMode',
  'supportedTransports',
  'license',
  'labels',
  'securityIndicators',
];

export const BACKEND_TO_FRONTEND_FILTER_KEY: Record<string, McpFilterCategoryKey> = {
  transports: 'supportedTransports',
  tags: 'labels',
};

export const OTHER_MCP_SERVERS_DISPLAY_NAME = 'Other MCP servers';
