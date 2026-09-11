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
  hfUsername?: string;
};

const ModelGatedAccessRequiredView: React.FC<ModelGatedAccessRequiredViewProps> = ({
  model,
  hfUsername,
}) => (
  <PageSection hasBodyWrapper={false} isFilled padding={{ default: 'noPadding' }}>
    <EmptyState
      headingLevel="h2"
      icon={ExclamationTriangleIcon}
      titleText={MODEL_CATALOG_GATED_ACCESS_REQUIRED.TITLE}
      variant={EmptyStateVariant.lg}
      data-testid="model-gated-access-required"
    >
      <EmptyStateBody>
        {hfUsername ? (
          <>
            This model is gated on Hugging Face. Log in to the Hugging Face account{' '}
            <strong>{hfUsername}</strong> and request access. After access is granted on Hugging
            Face, it might take a few hours for this model to show as available in the catalog.
          </>
        ) : (
          MODEL_CATALOG_GATED_ACCESS_REQUIRED.DESCRIPTION_WITHOUT_USERNAME
        )}
      </EmptyStateBody>
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
