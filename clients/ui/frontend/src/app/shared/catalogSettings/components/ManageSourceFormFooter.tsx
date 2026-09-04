import * as React from 'react';
import {
  PageSection,
  Stack,
  StackItem,
  Button,
  ActionList,
  ActionListItem,
  ActionListGroup,
  Alert,
} from '@patternfly/react-core';
import PreviewButton from './PreviewButton';

type ManageSourceFormFooterProps = {
  submitLabel: string;
  submitError?: Error;
  saveFailedTitle: string;
  isSubmitDisabled: boolean;
  isSubmitting: boolean;
  onSubmit: () => void;
  onCancel: () => void;
  cancelLabel?: string;
  isPreviewDisabled: boolean;
  isPreviewLoading: boolean;
  onPreview: () => void;
  previewLabel: string;
  testIdPrefix?: string;
};

const ManageSourceFormFooter: React.FC<ManageSourceFormFooterProps> = ({
  submitLabel,
  submitError,
  saveFailedTitle,
  isSubmitDisabled,
  isSubmitting,
  onSubmit,
  onCancel,
  cancelLabel = 'Cancel',
  isPreviewDisabled,
  isPreviewLoading,
  onPreview,
  previewLabel,
  testIdPrefix = '',
}) => (
  <PageSection hasBodyWrapper={false} stickyOnBreakpoint={{ default: 'bottom' }}>
    <Stack hasGutter>
      {submitError && (
        <StackItem>
          <Alert variant="danger" isInline title={saveFailedTitle}>
            {submitError.message}
          </Alert>
        </StackItem>
      )}
      <StackItem>
        <ActionList>
          <ActionListGroup>
            <ActionListItem>
              <Button
                isDisabled={isSubmitDisabled || isPreviewLoading}
                variant="primary"
                id={`${testIdPrefix}submit-button`}
                data-testid={`${testIdPrefix}submit-button`}
                isLoading={isSubmitting}
                onClick={onSubmit}
              >
                {submitLabel}
              </Button>
            </ActionListItem>
            <ActionListItem>
              <PreviewButton
                label={previewLabel}
                onClick={onPreview}
                isDisabled={isPreviewDisabled}
                isLoading={isPreviewLoading}
                variant="secondary"
                testId={`${testIdPrefix}preview-button`}
              />
            </ActionListItem>
            <ActionListItem>
              <Button
                isDisabled={isSubmitting || isPreviewLoading}
                variant="link"
                id={`${testIdPrefix}cancel-button`}
                data-testid={`${testIdPrefix}cancel-button`}
                onClick={onCancel}
              >
                {cancelLabel}
              </Button>
            </ActionListItem>
          </ActionListGroup>
        </ActionList>
      </StackItem>
    </Stack>
  </PageSection>
);

export default ManageSourceFormFooter;
