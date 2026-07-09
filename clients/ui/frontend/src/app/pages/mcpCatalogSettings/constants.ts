export const MCP_FORM_LABELS = {
  NAME: 'Name',
  YAML_CONTENT: 'Upload a YAML file',
  CATALOG_YAML_FILE: 'Catalog YAML file',
  MCP_SERVERS: 'MCP servers',
  SERVER_VISIBILITY: 'Server visibility',
  INCLUDED_SERVERS: 'Included MCP servers',
  EXCLUDED_SERVERS: 'Excluded MCP servers',
  ENABLE_SOURCE: 'Enable source',
} as const;

export const MCP_BUTTON_LABELS = {
  ADD: 'Add',
  SAVE: 'Save',
  PREVIEW: 'Preview',
  CANCEL: 'Cancel',
} as const;

export const MCP_SOURCE_NAME_CHARACTER_LIMIT = 238;

export const MCP_VALIDATION_MESSAGES = {
  NAME_REQUIRED: 'Name is required',
  YAML_CONTENT_REQUIRED: 'YAML content is required',
} as const;

export const MCP_DESCRIPTION_TEXT = {
  ENABLE_SOURCE:
    'Enable users in your organization to view MCP servers from this source in the MCP catalog.',
  FILTER_INFO:
    'Optionally filter which MCP servers from your source appear in the MCP catalog. If no filters are set, all servers from the source will be visible.',
  INCLUDED_SERVERS:
    'Enter MCP server names to include from this source. Separate names with commas. Use an asterisk suffix for prefix matching (Example: Kubernetes*).',
  EXCLUDED_SERVERS:
    'Enter MCP server names to exclude from this source. Separate names with commas. Use wildcards for pattern matching (Example: *experimental*).',
} as const;

export const MCP_HELPER_TEXT = {
  YAML: 'Upload or paste a YAML string.',
} as const;

export const MCP_PLACEHOLDERS = {
  INCLUDED_SERVERS: 'Example: Kubernetes, GitHub, PostgreSQL',
  EXCLUDED_SERVERS: 'Example: *preview*',
} as const;

export const MCP_EXPECTED_YAML_FORMAT_LABEL = 'View expected file format';

export const MCP_EXPECTED_FORMAT_DRAWER_TITLE = 'Expected file format';

export const MCP_PAGE_TITLES = {
  MCP_CATALOG_PREVIEW: 'MCP catalog preview',
  PREVIEW_SERVERS: 'Preview servers',
} as const;

export const MCP_ERROR_MESSAGES = {
  PREVIEW_FAILED: 'Preview failed',
  SAVE_FAILED: 'Failed to save source',
  FILE_UPLOAD_FAILED: 'File upload failed',
  FILE_UPLOAD_FAILED_BODY:
    "The YAML file couldn't be uploaded. Check its syntax and structure, then try again.",
} as const;

export const MCP_EMPTY_STATE_TEXT = {
  NO_SERVERS_INCLUDED: 'No MCP servers included',
  NO_SERVERS_INCLUDED_BODY:
    'No MCP servers from this source are visible in the MCP catalog. To include servers, edit the server visibility settings of this source.',
  NO_SERVERS_EXCLUDED: 'No MCP servers excluded',
  NO_SERVERS_EXCLUDED_BODY: 'No MCP servers from this source are excluded by this filter.',
} as const;
