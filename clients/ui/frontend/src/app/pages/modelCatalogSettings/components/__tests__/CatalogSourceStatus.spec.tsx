import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import {
  ModelCatalogSettingsContext,
  ModelCatalogSettingsContextType,
} from '~/app/context/modelCatalogSettings/ModelCatalogSettingsContext';
import { CatalogSourceConfig, CatalogSourceType } from '~/app/modelCatalogTypes';
import CatalogSourceStatus from '~/app/pages/modelCatalogSettings/components/CatalogSourceStatus';
import { CatalogSourceStatus as CatalogSourceStatusEnum } from '~/concepts/modelCatalogSettings/const';

const mockConfig: CatalogSourceConfig = {
  id: 'test-source',
  name: 'Test Source',
  type: CatalogSourceType.YAML,
  enabled: true,
};

const defaultPagination = { size: 0, pageSize: 10, nextPageToken: '' };

const renderWithContext = (
  config: CatalogSourceConfig,
  contextOverrides: Partial<ModelCatalogSettingsContextType>,
) => {
  const defaultContext: ModelCatalogSettingsContextType = {
    apiState: {
      apiAvailable: false,
      api: null as unknown as ModelCatalogSettingsContextType['apiState']['api'],
    },
    refreshAPIState: jest.fn(),
    catalogSourceConfigs: null,
    catalogSourceConfigsLoaded: false,
    catalogSourceConfigsLoadError: undefined,
    refreshCatalogSourceConfigs: jest.fn(),
    catalogSources: null,
    catalogSourcesLoaded: true,
    catalogSourcesLoadError: undefined,
    refreshCatalogSources: jest.fn(),
    ...contextOverrides,
  };

  return render(
    <ModelCatalogSettingsContext.Provider value={defaultContext}>
      <CatalogSourceStatus catalogSourceConfig={config} />
    </ModelCatalogSettingsContext.Provider>,
  );
};

describe('CatalogSourceStatus', () => {
  it('renders "Ready" label with outline variant', () => {
    renderWithContext(mockConfig, {
      catalogSources: {
        ...defaultPagination,
        items: [
          {
            id: 'test-source',
            name: 'Test',
            labels: [],
            status: CatalogSourceStatusEnum.AVAILABLE,
          },
        ],
      },
      catalogSourcesLoaded: true,
    });

    const label = screen.getByTestId('source-status-connected-test-source');
    expect(screen.getByText('Ready')).toBeVisible();
    expect(label.className).toMatch(/outline/);
    expect(label.className).not.toMatch(/filled/);
  });

  it('renders "Failed" label with outline variant', () => {
    renderWithContext(mockConfig, {
      catalogSources: {
        ...defaultPagination,
        items: [
          {
            id: 'test-source',
            name: 'Test',
            labels: [],
            status: CatalogSourceStatusEnum.ERROR,
            error: 'Connection refused',
          },
        ],
      },
      catalogSourcesLoaded: true,
    });

    const label = screen.getByTestId('source-status-failed-test-source');
    expect(screen.getByText('Failed')).toBeVisible();
    expect(label.className).toMatch(/outline/);
    expect(label.className).not.toMatch(/filled/);
  });

  it('renders "Starting" label with outline variant when source has no status', () => {
    renderWithContext(mockConfig, {
      catalogSources: {
        ...defaultPagination,
        items: [{ id: 'test-source', name: 'Test', labels: [] }],
      },
      catalogSourcesLoaded: true,
    });

    const label = screen.getByTestId('source-status-starting-test-source');
    expect(screen.getByText('Starting')).toBeVisible();
    expect(label.className).toMatch(/outline/);
  });

  it('renders "Unknown" label with outline variant when there is a load error', () => {
    renderWithContext(mockConfig, {
      catalogSources: null,
      catalogSourcesLoaded: true,
      catalogSourcesLoadError: new Error('API error'),
    });

    const label = screen.getByTestId('source-status-unknown-test-source');
    expect(screen.getByText('Unknown')).toBeVisible();
    expect(label.className).toMatch(/outline/);
  });

  it('renders "-" for default sources', () => {
    renderWithContext({ ...mockConfig, isDefault: true }, { catalogSourcesLoaded: true });
    expect(screen.getByText('-')).toBeVisible();
  });

  it('renders "-" for disabled sources', () => {
    renderWithContext({ ...mockConfig, enabled: false }, { catalogSourcesLoaded: true });
    expect(screen.getByText('-')).toBeVisible();
  });
});
