import React from 'react';
import { Label, Popover } from '@patternfly/react-core';
import type { HfAccessLabelVariant } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';
import { MODEL_CATALOG_POPOVER_MESSAGES } from '~/concepts/modelCatalog/const';

type ModelCatalogAccessLabelProps = {
  variant: HfAccessLabelVariant;
};

const ModelCatalogAccessLabel: React.FC<ModelCatalogAccessLabelProps> = ({ variant }) => {
  if (variant === 'private') {
    return (
      <Popover bodyContent={MODEL_CATALOG_POPOVER_MESSAGES.HF_PRIVATE}>
        <Label isClickable data-testid="model-catalog-access-label-private">
          Private
        </Label>
      </Popover>
    );
  }

  return (
    <Popover bodyContent={MODEL_CATALOG_POPOVER_MESSAGES.HF_GATED}>
      <Label isClickable data-testid="model-catalog-access-label-gated">
        Gated
      </Label>
    </Popover>
  );
};

export default ModelCatalogAccessLabel;
