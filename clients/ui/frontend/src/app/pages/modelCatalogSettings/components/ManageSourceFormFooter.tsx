import * as React from 'react';
import { ManageSourceFormFooter as SharedManageSourceFormFooter } from '~/app/shared/catalogSettings';
import { ERROR_MESSAGES, BUTTON_LABELS } from '~/app/pages/modelCatalogSettings/constants';

type ManageSourceFormFooterProps = {
  submitLabel: string;
  submitError?: Error;
  isSubmitDisabled: boolean;
  isSubmitting: boolean;
  onSubmit: () => void;
  onCancel: () => void;
  isPreviewDisabled: boolean;
  isPreviewLoading: boolean;
  onPreview: () => void;
};

const ManageSourceFormFooter: React.FC<ManageSourceFormFooterProps> = (props) => (
  <SharedManageSourceFormFooter
    {...props}
    saveFailedTitle={ERROR_MESSAGES.SAVE_FAILED}
    previewLabel={BUTTON_LABELS.PREVIEW}
    cancelLabel={BUTTON_LABELS.CANCEL}
  />
);

export default ManageSourceFormFooter;
