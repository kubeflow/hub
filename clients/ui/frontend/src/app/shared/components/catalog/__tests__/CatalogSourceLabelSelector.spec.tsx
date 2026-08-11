import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import CatalogSourceLabelSelector from '~/app/shared/components/catalog/CatalogSourceLabelSelector';

jest.mock('mod-arch-kubeflow', () => ({
  useThemeContext: () => ({ isMUITheme: false }),
}));

jest.mock('mod-arch-shared', () => ({
  ThemeAwareSearchInput: ({
    value,
    onChange,
    onSearch,
    onClear,
    placeholder,
    'data-testid': testId,
  }: {
    value: string;
    onChange: (value: string) => void;
    onSearch: (event: React.SyntheticEvent<HTMLButtonElement>, value: string) => void;
    onClear: () => void;
    placeholder: string;
    'data-testid': string;
  }) => (
    <div>
      <input
        data-testid={testId}
        aria-label={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
      <button type="button" onClick={(e) => onSearch(e, value)}>
        Search
      </button>
      <button type="button" onClick={onClear}>
        Clear search
      </button>
    </div>
  ),
}));

const baseProps = {
  searchTerm: '',
  onSearch: jest.fn(),
  onClearSearch: jest.fn(),
  onResetAllFilters: jest.fn(),
  hasFiltersApplied: false,
  searchPlaceholder: 'Search catalogs',
  searchInputTestId: 'catalog-search-input',
  searchButtonTestId: 'catalog-search-button',
};

describe('CatalogSourceLabelSelector', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('syncs the search input from the searchTerm prop', () => {
    const { rerender } = render(<CatalogSourceLabelSelector {...baseProps} searchTerm="alpha" />);
    expect(screen.getByTestId('catalog-search-input')).toHaveValue('alpha');

    rerender(<CatalogSourceLabelSelector {...baseProps} searchTerm="beta" />);
    expect(screen.getByTestId('catalog-search-input')).toHaveValue('beta');
  });

  it('calls onSearch when the search action is used', () => {
    const onSearch = jest.fn();
    render(<CatalogSourceLabelSelector {...baseProps} onSearch={onSearch} />);

    fireEvent.change(screen.getByTestId('catalog-search-input'), {
      target: { value: 'llama' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Search' }));

    expect(onSearch).toHaveBeenCalledWith('llama');
  });

  it('renders active filters only when filters are applied by default', () => {
    const { rerender } = render(
      <CatalogSourceLabelSelector
        {...baseProps}
        hasFiltersApplied={false}
        renderActiveFilters={() => <div data-testid="active-filters">filters</div>}
      />,
    );
    expect(screen.queryByTestId('active-filters')).not.toBeInTheDocument();

    rerender(
      <CatalogSourceLabelSelector
        {...baseProps}
        hasFiltersApplied
        renderActiveFilters={() => <div data-testid="active-filters">filters</div>}
      />,
    );
    expect(screen.getByTestId('active-filters')).toBeInTheDocument();
  });

  it('always mounts active filters when alwaysMountActiveFilters is true', () => {
    render(
      <CatalogSourceLabelSelector
        {...baseProps}
        hasFiltersApplied={false}
        alwaysMountActiveFilters
        renderActiveFilters={() => <div data-testid="active-filters">filters</div>}
      />,
    );

    expect(screen.getByTestId('active-filters')).toBeInTheDocument();
  });

  it('renders source label blocks and extra row content', () => {
    render(
      <CatalogSourceLabelSelector
        {...baseProps}
        renderSourceLabelBlocks={() => <div data-testid="source-blocks">blocks</div>}
        sourceLabelRowExtra={<div data-testid="row-extra">extra</div>}
        additionalSections={<div data-testid="extra-section">section</div>}
      />,
    );

    expect(screen.getByTestId('source-blocks')).toBeInTheDocument();
    expect(screen.getByTestId('row-extra')).toBeInTheDocument();
    expect(screen.getByTestId('extra-section')).toBeInTheDocument();
  });
});
