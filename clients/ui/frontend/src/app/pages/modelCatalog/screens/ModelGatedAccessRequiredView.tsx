import React from 'react';
import {
  EmptyState,
  EmptyStateActions,
  EmptyStateBody,
  EmptyStateFooter,
  EmptyStateVariant,
  PageSection,
} from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';
import type { CatalogModel } from '~/app/modelCatalogTypes';
import { getHuggingFaceModelUrl } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import ExternalLink from '~/app/shared/components/ExternalLink';
import { MODEL_CATALOG_GATED_ACCESS_REQUIRED } from '~/concepts/modelCatalog/const';

type ModelGatedAccessRequiredViewProps = {
  model: CatalogModel;
};

const ModelGatedAccessRequiredView: React.FC<ModelGatedAccessRequiredViewProps> = ({ model }) => (
  <PageSection hasBodyWrapper={false} isFilled padding={{ default: 'noPadding' }}>
    <EmptyState
      headingLevel="h2"
      icon={ExclamationTriangleIcon}
      titleText={MODEL_CATALOG_GATED_ACCESS_REQUIRED.TITLE}
      variant={EmptyStateVariant.lg}
      data-testid="model-gated-access-required"
    >
      <EmptyStateBody>{MODEL_CATALOG_GATED_ACCESS_REQUIRED.DESCRIPTION}</EmptyStateBody>
      <EmptyStateFooter>
        <EmptyStateActions>
          <ExternalLink
            text={MODEL_CATALOG_GATED_ACCESS_REQUIRED.REQUEST_ACCESS_LINK_TEXT}
            to={getHuggingFaceModelUrl(model)}
            testId="model-gated-access-request-link"
          />
        </EmptyStateActions>
      </EmptyStateFooter>
    </EmptyState>
  </PageSection>
);

export default ModelGatedAccessRequiredView;
