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

type SkillManageSourceFormFooterProps = {
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

const SkillManageSourceFormFooter: React.FC<SkillManageSourceFormFooterProps> = ({
  submitLabel,
  submitError,
  isSubmitDisabled,
  isSubmitting,
  onSubmit,
  onCancel,
  isPreviewDisabled,
  isPreviewLoading,
  onPreview,
}) => (
  <PageSection hasBodyWrapper={false} stickyOnBreakpoint={{ default: 'bottom' }}>
    <Stack hasGutter>
      {submitError && (
        <StackItem>
          <Alert variant="danger" isInline title="Failed to save source">
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
                id="skill-submit-button"
                data-testid="skill-submit-button"
                isLoading={isSubmitting}
                onClick={onSubmit}
              >
                {submitLabel}
              </Button>
            </ActionListItem>
            <ActionListItem>
              <Button
                variant="secondary"
                onClick={onPreview}
                isDisabled={isPreviewDisabled}
                isLoading={isPreviewLoading}
                data-testid="skill-preview-button"
              >
                Preview
              </Button>
            </ActionListItem>
            <ActionListItem>
              <Button
                isDisabled={isSubmitting || isPreviewLoading}
                variant="link"
                id="skill-cancel-button"
                data-testid="skill-cancel-button"
                onClick={onCancel}
              >
                Cancel
              </Button>
            </ActionListItem>
          </ActionListGroup>
        </ActionList>
      </StackItem>
    </Stack>
  </PageSection>
);

export default SkillManageSourceFormFooter;
