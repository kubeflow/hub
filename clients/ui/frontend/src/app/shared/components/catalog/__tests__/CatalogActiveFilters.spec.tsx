import React from 'react';
import { Toolbar, ToolbarContent } from '@patternfly/react-core';
import { render, screen, fireEvent, within } from '@testing-library/react';
import '@testing-library/jest-dom';
import CatalogActiveFilters from '~/app/shared/components/catalog/CatalogActiveFilters';

const FILTER_KEYS = ['framework', 'license'] as const;
type FilterKey = (typeof FILTER_KEYS)[number];

const CATEGORY_NAMES: Record<FilterKey, string> = {
  framework: 'Framework',
  license: 'License',
};

const renderWithToolbar = (ui: React.ReactElement) =>
  render(
    <Toolbar clearAllFilters={jest.fn()}>
      <ToolbarContent>{ui}</ToolbarContent>
    </Toolbar>,
  );

describe('CatalogActiveFilters', () => {
  it('renders chips for selected filter values', () => {
    renderWithToolbar(
      <CatalogActiveFilters
        filterKeys={FILTER_KEYS}
        categoryNames={CATEGORY_NAMES}
        filters={{ framework: ['langgraph'], license: ['apache-2.0'] }}
        setFilters={jest.fn()}
        testIdPrefix="test-filter"
      />,
    );

    expect(screen.getByTestId('test-filter-chip-framework-langgraph')).toHaveTextContent(
      'langgraph',
    );
    expect(screen.getByTestId('test-filter-chip-license-apache-2.0')).toHaveTextContent(
      'apache-2.0',
    );
  });

  it('uses labelMappings for display text when provided', () => {
    renderWithToolbar(
      <CatalogActiveFilters
        filterKeys={FILTER_KEYS}
        categoryNames={CATEGORY_NAMES}
        filters={{ framework: ['langgraph'] }}
        setFilters={jest.fn()}
        labelMappings={{ framework: { langgraph: 'LangGraph' } }}
        testIdPrefix="test-filter"
      />,
    );

    expect(screen.getByTestId('test-filter-chip-framework-langgraph')).toHaveTextContent(
      'LangGraph',
    );
  });

  it('removes a single value when a chip is deleted', () => {
    const setFilters = jest.fn();
    renderWithToolbar(
      <CatalogActiveFilters
        filterKeys={FILTER_KEYS}
        categoryNames={CATEGORY_NAMES}
        filters={{ framework: ['langgraph', 'crewai'] }}
        setFilters={setFilters}
        testIdPrefix="test-filter"
      />,
    );

    const chip = screen
      .getByTestId('test-filter-chip-framework-langgraph')
      .closest('.pf-v6-c-label');
    expect(chip).not.toBeNull();
    fireEvent.click(within(chip as HTMLElement).getByRole('button'));

    expect(setFilters).toHaveBeenCalledTimes(1);
    const updater = setFilters.mock.calls[0][0] as (prev: { framework?: string[] }) => {
      framework?: string[];
    };
    expect(updater({ framework: ['langgraph', 'crewai'] })).toEqual({
      framework: ['crewai'],
    });
  });

  it('clears a category when the label group is deleted', () => {
    const setFilters = jest.fn();
    renderWithToolbar(
      <CatalogActiveFilters
        filterKeys={FILTER_KEYS}
        categoryNames={CATEGORY_NAMES}
        filters={{ framework: ['langgraph'] }}
        setFilters={setFilters}
        testIdPrefix="test-filter"
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /close label group/i }));

    expect(setFilters).toHaveBeenCalledTimes(1);
    const updater = setFilters.mock.calls[0][0] as (prev: { framework?: string[] }) => {
      framework?: string[];
    };
    expect(updater({ framework: ['langgraph'] })).toEqual({ framework: [] });
  });

  it('keeps ToolbarFilter mounted with empty labels when a category has no values', () => {
    renderWithToolbar(
      <CatalogActiveFilters
        filterKeys={FILTER_KEYS}
        categoryNames={CATEGORY_NAMES}
        filters={{ framework: [] }}
        setFilters={jest.fn()}
        testIdPrefix="test-filter"
      />,
    );

    expect(screen.getByTestId('test-filter-container-framework')).toBeInTheDocument();
    expect(screen.queryByTestId('test-filter-chip-framework-langgraph')).not.toBeInTheDocument();
  });
});
