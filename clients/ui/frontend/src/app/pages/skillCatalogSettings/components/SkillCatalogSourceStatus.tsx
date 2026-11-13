import * as React from 'react';
import {
  Button,
  Label,
  Spinner,
  Stack,
  StackItem,
  Tooltip,
  Truncate,
} from '@patternfly/react-core';
import { InProgressIcon } from '@patternfly/react-icons';
import type { SkillCatalogSourceConfig } from '~/app/skillCatalogTypes';
import { SkillCatalogSettingsContext } from '~/app/context/skillCatalogSettings/SkillCatalogSettingsContext';
import { CatalogSourceStatus } from '~/app/shared/types/catalogTypes';
import CatalogSourceStatusErrorModal from '~/app/pages/modelCatalogSettings/components/CatalogSourceStatusErrorModal';

type SkillCatalogSourceStatusProps = {
  skillCatalogSourceConfig: SkillCatalogSourceConfig;
};

const SkillCatalogSourceStatus: React.FC<SkillCatalogSourceStatusProps> = ({
  skillCatalogSourceConfig,
}) => {
  const {
    skillCatalogSources,
    skillCatalogSourcesLoaded,
    skillCatalogSourcesLoadError,
    isSkillSourceSyncPending,
  } = React.useContext(SkillCatalogSettingsContext);
  const [isErrorModalOpen, setIsErrorModalOpen] = React.useState(false);

  if (!skillCatalogSourceConfig.enabled || skillCatalogSourceConfig.isDefault) {
    return <>-</>;
  }

  // Takes precedence over the reported status: until the catalog confirms the
  // reload, the status it returns describes the previous sync, so showing "Ready"
  // here would claim the edit had already been applied.
  if (isSkillSourceSyncPending(skillCatalogSourceConfig.id)) {
    return (
      <Tooltip content="Changes saved. The catalog is re-reading its configuration and re-indexing the repositories; skills can take a few minutes to update.">
        <Label
          color="blue"
          variant="outline"
          icon={<Spinner size="sm" />}
          data-testid={`skill-source-status-syncing-${skillCatalogSourceConfig.id}`}
        >
          Syncing
        </Label>
      </Tooltip>
    );
  }

  if (!skillCatalogSourcesLoaded) {
    return (
      <Spinner
        size="md"
        data-testid={`skill-source-status-loading-${skillCatalogSourceConfig.id}`}
      />
    );
  }

  const matchingSource = skillCatalogSources?.items?.find(
    (source) => source.id === skillCatalogSourceConfig.id,
  );

  const startingOrUnknownLabel = (
    <Label
      color="grey"
      variant="outline"
      icon={<InProgressIcon />}
      data-testid={`skill-source-status-${skillCatalogSourcesLoadError ? 'unknown' : 'starting'}-${skillCatalogSourceConfig.id}`}
    >
      {skillCatalogSourcesLoadError ? 'Unknown' : 'Starting'}
    </Label>
  );

  if (!matchingSource || !matchingSource.status) {
    return startingOrUnknownLabel;
  }

  switch (matchingSource.status) {
    case CatalogSourceStatus.AVAILABLE:
      return (
        <Label
          status="success"
          variant="outline"
          data-testid={`skill-source-status-connected-${skillCatalogSourceConfig.id}`}
        >
          Ready
        </Label>
      );

    case CatalogSourceStatus.ERROR: {
      const errorMessage = matchingSource.error || 'Unknown error occurred';

      return (
        <>
          <Stack hasGutter>
            <StackItem>
              <Label
                status="danger"
                variant="outline"
                data-testid={`skill-source-status-failed-${skillCatalogSourceConfig.id}`}
              >
                Failed
              </Label>
            </StackItem>
            <StackItem>
              <Button
                variant="link"
                isInline
                isDanger
                onClick={() => setIsErrorModalOpen(true)}
                data-testid={`skill-source-status-error-link-${skillCatalogSourceConfig.id}`}
              >
                <Truncate content={errorMessage} tooltipProps={{ hidden: true }} />
              </Button>
            </StackItem>
          </Stack>
          <CatalogSourceStatusErrorModal
            isOpen={isErrorModalOpen}
            onClose={() => setIsErrorModalOpen(false)}
            errorMessage={errorMessage}
          />
        </>
      );
    }

    case CatalogSourceStatus.DISABLED:
      return startingOrUnknownLabel;
    default:
      return startingOrUnknownLabel;
  }
};

export default SkillCatalogSourceStatus;
