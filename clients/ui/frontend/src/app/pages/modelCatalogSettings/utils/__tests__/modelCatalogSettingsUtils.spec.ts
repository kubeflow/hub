import { mockHuggingFaceCatalogSourceConfig, mockYamlCatalogSourceConfig } from '~/__mocks__';
import { CatalogSourceType } from '~/app/modelCatalogTypes';
import {
  catalogSourceConfigToFormData,
  getPayloadForConfig,
  transformFormDataToConfig,
} from '~/app/pages/modelCatalogSettings/utils/modelCatalogSettingsUtils';
import { ManageSourceFormData } from '~/app/pages/modelCatalogSettings/useManageSourceData';

const catalogSourceDefaultConfigYAMLMock = mockYamlCatalogSourceConfig({});
const catalogSourceConfigYAMLMock = mockYamlCatalogSourceConfig({ isDefault: false });
const catalogSourceConfigHFMock = mockHuggingFaceCatalogSourceConfig({});

const yamlFormData: ManageSourceFormData = {
  accessToken: '',
  allowedModels: '',
  enabled: true,
  excludedModels: '',
  id: 'sample_source_1',
  isDefault: false,
  name: 'Source 1',
  organization: '',
  sourceType: CatalogSourceType.YAML,
  yamlContent: 'models:\n  - name: model1',
};
const yamlDefaultFormData: ManageSourceFormData = {
  accessToken: '',
  allowedModels: '',
  enabled: true,
  excludedModels: '',
  id: 'sample_source_1',
  isDefault: true,
  name: 'Source 1',
  organization: '',
  sourceType: CatalogSourceType.YAML,
  yamlContent: '',
};
const hfFormData: ManageSourceFormData = {
  accessToken: 'apikey',
  allowedModels: '',
  enabled: true,
  excludedModels: '',
  id: 'source_2',
  isDefault: false,
  name: 'Huggingface source 2',
  organization: 'org1',
  sourceType: CatalogSourceType.HUGGING_FACE,
  yamlContent: '',
};

describe('catalogSourceConfigToFormData', () => {
  it('should convert the data from catalogSourceConfig to formData', () => {
    expect(catalogSourceConfigToFormData(catalogSourceDefaultConfigYAMLMock)).toEqual(
      yamlDefaultFormData,
    );
    expect(catalogSourceConfigToFormData(catalogSourceConfigYAMLMock)).toEqual({
      accessToken: '',
      allowedModels: '',
      enabled: true,
      excludedModels: '',
      id: 'sample_source_1',
      isDefault: false,
      name: 'Source 1',
      organization: '',
      sourceType: CatalogSourceType.YAML,
      yamlContent: '',
    });
    expect(catalogSourceConfigToFormData(catalogSourceConfigHFMock)).toEqual(hfFormData);
  });
});

describe('transformFormDataToConfig', () => {
  it('should transform YAML form data to full config', () => {
    expect(transformFormDataToConfig(yamlFormData)).toEqual({
      id: 'sample_source_1',
      name: 'Source 1',
      enabled: true,
      isDefault: false,
      type: CatalogSourceType.YAML,
      yaml: 'models:\n  - name: model1',
      yamlCatalogPath: undefined,
      includedModels: [],
      excludedModels: [],
    });
  });

  it('should transform YAML form data with existing source config including yamlCatalogPath', () => {
    const existingConfig = mockYamlCatalogSourceConfig({
      yamlCatalogPath: 'sample_source_1.yaml',
    });
    expect(transformFormDataToConfig(yamlFormData, existingConfig)).toEqual({
      id: 'sample_source_1',
      name: 'Source 1',
      enabled: true,
      isDefault: false,
      type: CatalogSourceType.YAML,
      yaml: 'models:\n  - name: model1',
      yamlCatalogPath: 'sample_source_1.yaml',
      includedModels: [],
      excludedModels: [],
    });
  });

  it('should transform HuggingFace form data to full config', () => {
    expect(transformFormDataToConfig(hfFormData)).toEqual({
      id: 'source_2',
      name: 'Huggingface source 2',
      enabled: true,
      isDefault: false,
      type: CatalogSourceType.HUGGING_FACE,
      apiKey: 'apikey',
      allowedOrganization: 'org1',
      includedModels: [],
      excludedModels: [],
    });
  });

  it('should transform default source form data to full config', () => {
    expect(transformFormDataToConfig(yamlDefaultFormData)).toEqual({
      id: 'sample_source_1',
      name: 'Source 1',
      enabled: true,
      isDefault: true,
      type: CatalogSourceType.YAML,
      yaml: '',
      yamlCatalogPath: undefined,
      includedModels: [],
      excludedModels: [],
    });
  });
});

describe('getPayloadForConfig', () => {
  it('should return full config for non-default source (create mode)', () => {
    const config = transformFormDataToConfig(yamlFormData);
    expect(getPayloadForConfig(config, false)).toEqual({
      id: 'sample_source_1',
      name: 'Source 1',
      enabled: true,
      isDefault: false,
      type: CatalogSourceType.YAML,
      yaml: 'models:\n  - name: model1',
      includedModels: [],
      excludedModels: [],
    });
  });

  it('should return config without id for non-default source (edit mode)', () => {
    const config = transformFormDataToConfig(yamlFormData);
    expect(getPayloadForConfig(config, true)).toEqual({
      name: 'Source 1',
      enabled: true,
      isDefault: false,
      type: CatalogSourceType.YAML,
      yaml: 'models:\n  - name: model1',
      includedModels: [],
      excludedModels: [],
    });
  });

  it('should return only allowed fields for default source', () => {
    const config = transformFormDataToConfig(yamlDefaultFormData);
    expect(getPayloadForConfig(config, false)).toEqual({
      enabled: true,
      includedModels: [],
      excludedModels: [],
    });
  });

  it('should return only allowed fields for default source (edit mode)', () => {
    const config = transformFormDataToConfig(yamlDefaultFormData);
    expect(getPayloadForConfig(config, true)).toEqual({
      enabled: true,
      includedModels: [],
      excludedModels: [],
    });
  });

  it('should return full config for HuggingFace source', () => {
    const config = transformFormDataToConfig(hfFormData);
    expect(getPayloadForConfig(config, false)).toEqual({
      id: 'source_2',
      name: 'Huggingface source 2',
      enabled: true,
      isDefault: false,
      type: CatalogSourceType.HUGGING_FACE,
      apiKey: 'apikey',
      allowedOrganization: 'org1',
      includedModels: [],
      excludedModels: [],
    });
  });
});
