import * as React from 'react';
import { CatalogActiveFilters, CatalogSourceLabelSelector } from '~/app/shared/components/catalog';
import { AgentsCatalogContext } from '~/app/context/agentsCatalog/AgentsCatalogContext';
import { hasAgentFiltersApplied } from '~/app/pages/agentsCatalog/utils/agentsCatalogUtils';
import {
  AGENT_FILTER_KEYS,
  AGENT_FILTER_CATEGORY_NAMES,
  AGENT_LABEL_MAPPINGS,
} from '~/app/pages/agentsCatalog/const';
import AgentsCatalogSourceLabelBlocks from './AgentsCatalogSourceLabelBlocks';

type AgentsCatalogSourceLabelSelectorProps = {
  searchTerm: string;
  onSearch: (term: string) => void;
  onClearSearch: () => void;
  onResetAllFilters: () => void;
};

const AgentsCatalogSourceLabelSelector: React.FC<AgentsCatalogSourceLabelSelectorProps> = ({
  searchTerm,
  onSearch,
  onClearSearch,
  onResetAllFilters,
}) => {
  const { filters, setFilters } = React.useContext(AgentsCatalogContext);
  const hasFiltersAppliedValue = hasAgentFiltersApplied(filters, searchTerm);

  return (
    <CatalogSourceLabelSelector
      searchTerm={searchTerm}
      onSearch={onSearch}
      onClearSearch={onClearSearch}
      onResetAllFilters={onResetAllFilters}
      hasFiltersApplied={hasFiltersAppliedValue}
      searchPlaceholder="Search by name, keyword, or description"
      searchInputTestId="agents-catalog-search-input"
      searchButtonTestId="agents-search-button"
      searchAriaLabel="Search agents"
      renderActiveFilters={() => (
        <CatalogActiveFilters
          filterKeys={AGENT_FILTER_KEYS}
          categoryNames={AGENT_FILTER_CATEGORY_NAMES}
          filters={filters}
          setFilters={setFilters}
          labelMappings={AGENT_LABEL_MAPPINGS}
          testIdPrefix="agent-filter"
        />
      )}
      renderSourceLabelBlocks={() => <AgentsCatalogSourceLabelBlocks />}
    />
  );
};

export default AgentsCatalogSourceLabelSelector;
