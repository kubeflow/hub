import * as React from 'react';
import {
  Flex,
  Stack,
  StackItem,
  Toolbar,
  ToolbarContent,
  ToolbarGroup,
  ToolbarItem,
  ToolbarToggleGroup,
} from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';
import { ThemeAwareSearchInput } from 'mod-arch-shared';
import { SkillCatalogContext } from '~/app/context/skillCatalog/SkillCatalogContext';
import { CatalogSourceLabelToggle, getLabelDisplayName } from '~/app/shared/components/catalog';
import { OTHER_SKILLS_DISPLAY_NAME, ALL_SKILLS_LABEL } from '~/app/pages/skillCatalog/const';

type SkillCatalogSourceLabelSelectorProps = {
  searchTerm: string;
  onSearch: (term: string) => void;
  onClearSearch: () => void;
};

const SkillCatalogSourceLabelSelector: React.FC<SkillCatalogSourceLabelSelectorProps> = ({
  searchTerm,
  onSearch,
  onClearSearch,
}) => {
  const [inputValue, setInputValue] = React.useState(searchTerm || '');
  const {
    catalogSources,
    catalogLabels,
    catalogSourcesLoaded,
    selectedSourceLabel,
    setSelectedSourceLabel,
    emptyCategoryLabels,
  } = React.useContext(SkillCatalogContext);

  React.useEffect(() => {
    setInputValue(searchTerm || '');
  }, [searchTerm]);

  const handleSearchInputChange = React.useCallback((value: string) => {
    setInputValue(value);
  }, []);

  const handleSearchInputSearch = React.useCallback(
    (_: React.SyntheticEvent<HTMLButtonElement>, value: string) => {
      onSearch(value.trim());
    },
    [onSearch],
  );

  const getLabelDisplayNameForSkill = React.useCallback(
    (label: string) =>
      getLabelDisplayName(label, catalogLabels, OTHER_SKILLS_DISPLAY_NAME, 'skills'),
    [catalogLabels],
  );

  return (
    <Stack hasGutter>
      <StackItem>
        <Toolbar className="pf-v6-u-pb-0">
          <ToolbarContent rowWrap={{ default: 'wrap' }}>
            <Flex style={{ flex: 1 }}>
              <ToolbarToggleGroup style={{ flex: 1 }} breakpoint="md" toggleIcon={<FilterIcon />}>
                <ToolbarGroup
                  style={{ flex: 1 }}
                  variant="filter-group"
                  gap={{ default: 'gapMd' }}
                  alignItems="center"
                >
                  <ToolbarItem style={{ flex: 1 }}>
                    <ThemeAwareSearchInput
                      data-testid="skill-catalog-search-input"
                      aria-label="Search skills"
                      className="toolbar-fieldset-wrapper"
                      placeholder="Search by name, keyword, or description"
                      value={inputValue}
                      onChange={handleSearchInputChange}
                      onSearch={handleSearchInputSearch}
                      onClear={onClearSearch}
                    />
                  </ToolbarItem>
                </ToolbarGroup>
              </ToolbarToggleGroup>
            </Flex>
          </ToolbarContent>
        </Toolbar>
      </StackItem>
      {catalogSourcesLoaded && (
        <StackItem>
          <Flex
            justifyContent={{ default: 'justifyContentSpaceBetween' }}
            alignItems={{ default: 'alignItemsCenter' }}
          >
            <CatalogSourceLabelToggle
              catalogSources={catalogSources}
              catalogLabels={catalogLabels}
              selectedSourceLabel={selectedSourceLabel}
              onSelectSourceLabel={setSelectedSourceLabel}
              allBlockDisplayName={ALL_SKILLS_LABEL}
              getLabelDisplayNameOverride={getLabelDisplayNameForSkill}
              emptyCategoryLabels={emptyCategoryLabels}
            />
          </Flex>
        </StackItem>
      )}
    </Stack>
  );
};

export default SkillCatalogSourceLabelSelector;
