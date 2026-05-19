import { renderHook } from '@testing-library/react';
import { useCatalogFilterConfigs } from '~/app/shared/components/catalog';

const filterOptions = {
  provider: { type: 'string', values: ['Red Hat', 'IBM'] },
  license: { type: 'string', values: ['MIT', 'Apache'] },
  empty: { type: 'string', values: [] },
};

const filterNames: Record<string, string> = {
  provider: 'Provider',
  license: 'License',
  empty: 'Empty',
};

describe('useCatalogFilterConfigs', () => {
  it('returns FilterPanelItem for each key with non-empty values', () => {
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider', 'license', 'empty'],
        filterNames,
        filterOptions,
        selectedFilters: {},
        onFilterChange: jest.fn(),
      }),
    );

    expect(result.current).toHaveLength(2);
    expect(result.current[0].key).toBe('provider');
    expect(result.current[1].key).toBe('license');
  });

  it('skips keys with no matching filter option', () => {
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider', 'nonexistent'],
        filterNames,
        filterOptions,
        selectedFilters: {},
        onFilterChange: jest.fn(),
      }),
    );

    expect(result.current).toHaveLength(1);
    expect(result.current[0].key).toBe('provider');
  });

  it('populates filterValues and title from options and names', () => {
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider'],
        filterNames,
        filterOptions,
        selectedFilters: {},
        onFilterChange: jest.fn(),
      }),
    );

    expect(result.current[0].title).toBe('Provider');
    expect(result.current[0].filterValues).toEqual(['Red Hat', 'IBM']);
  });

  it('uses key as fallback title when filterNames has no entry', () => {
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider'],
        filterNames: {},
        filterOptions,
        selectedFilters: {},
        onFilterChange: jest.fn(),
      }),
    );

    expect(result.current[0].title).toBe('provider');
  });

  it('populates selectedValues from selectedFilters', () => {
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider'],
        filterNames,
        filterOptions,
        selectedFilters: { provider: ['Red Hat'] },
        onFilterChange: jest.fn(),
      }),
    );

    expect(result.current[0].selectedValues).toEqual(['Red Hat']);
  });

  it('defaults selectedValues to empty array when filter not in selectedFilters', () => {
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider'],
        filterNames,
        filterOptions,
        selectedFilters: {},
        onFilterChange: jest.fn(),
      }),
    );

    expect(result.current[0].selectedValues).toEqual([]);
  });

  it('onToggle adds value when checked', () => {
    const onFilterChange = jest.fn();
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider'],
        filterNames,
        filterOptions,
        selectedFilters: { provider: ['Red Hat'] },
        onFilterChange,
      }),
    );

    result.current[0].onToggle('IBM', true);
    expect(onFilterChange).toHaveBeenCalledWith('provider', ['Red Hat', 'IBM']);
  });

  it('onToggle removes value when unchecked', () => {
    const onFilterChange = jest.fn();
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider'],
        filterNames,
        filterOptions,
        selectedFilters: { provider: ['Red Hat', 'IBM'] },
        onFilterChange,
      }),
    );

    result.current[0].onToggle('Red Hat', false);
    expect(onFilterChange).toHaveBeenCalledWith('provider', ['IBM']);
  });

  it('applies label mapping via getLabel', () => {
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider'],
        filterNames,
        filterOptions,
        selectedFilters: {},
        onFilterChange: jest.fn(),
        labelMappings: { provider: { 'Red Hat': 'Red Hat Inc.' } },
      }),
    );

    expect(result.current[0].getLabel?.('Red Hat')).toBe('Red Hat Inc.');
    expect(result.current[0].getLabel?.('IBM')).toBe('IBM');
  });

  it('getLabel is undefined when no mapping provided for key', () => {
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider'],
        filterNames,
        filterOptions,
        selectedFilters: {},
        onFilterChange: jest.fn(),
      }),
    );

    expect(result.current[0].getLabel).toBeUndefined();
  });

  it('handles undefined filterOptions gracefully', () => {
    const { result } = renderHook(() =>
      useCatalogFilterConfigs({
        filterKeys: ['provider'],
        filterNames,
        filterOptions: undefined,
        selectedFilters: {},
        onFilterChange: jest.fn(),
      }),
    );

    expect(result.current).toHaveLength(0);
  });
});
