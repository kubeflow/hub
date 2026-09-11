import * as React from 'react';
import { MODEL_CATALOG_GATED_ACCESS_REQUIRED_COPY } from '~/concepts/modelCatalog/const';

export const getGatedAccessRequiredDescriptionText = (hfUsername?: string): string => {
  const { INTRO, GENERIC_ACTION, USERNAME_PROMPT, USERNAME_ACTION, OUTRO } =
    MODEL_CATALOG_GATED_ACCESS_REQUIRED_COPY;

  if (!hfUsername) {
    return `${INTRO} ${GENERIC_ACTION} ${OUTRO}`;
  }

  return `${INTRO} ${USERNAME_PROMPT} ${hfUsername} ${USERNAME_ACTION} ${OUTRO}`;
};

export const renderGatedAccessRequiredDescription = (hfUsername?: string): React.ReactNode => {
  const { INTRO, USERNAME_PROMPT, USERNAME_ACTION, OUTRO } =
    MODEL_CATALOG_GATED_ACCESS_REQUIRED_COPY;

  if (!hfUsername) {
    return getGatedAccessRequiredDescriptionText();
  }

  return (
    <>
      {INTRO} {USERNAME_PROMPT} <strong>{hfUsername}</strong> {USERNAME_ACTION} {OUTRO}
    </>
  );
};
