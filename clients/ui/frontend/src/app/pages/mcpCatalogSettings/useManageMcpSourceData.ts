import { GenericObjectState } from 'mod-arch-core';
import useGenericObjectState from 'mod-arch-core/dist/utilities/useGenericObjectState';
import { McpCatalogSourceType } from '~/app/mcpServerCatalogTypes';

export type ManageMcpSourceFormData = {
  name: string;
  id: string;
  sourceType: McpCatalogSourceType;
  yamlContent: string;
  includedServers: string;
  excludedServers: string;
  enabled: boolean;
  isDefault: boolean;
  yamlCatalogPath?: string;
};

const manageMcpSourceFormDataDefaults: ManageMcpSourceFormData = {
  name: '',
  id: '',
  sourceType: McpCatalogSourceType.YAML,
  yamlContent: '',
  includedServers: '',
  excludedServers: '',
  enabled: true,
  isDefault: false,
};

export const useManageMcpSourceData = (
  existingData?: Partial<ManageMcpSourceFormData>,
): GenericObjectState<ManageMcpSourceFormData> =>
  useGenericObjectState<ManageMcpSourceFormData>({
    ...manageMcpSourceFormDataDefaults,
    ...existingData,
  });
