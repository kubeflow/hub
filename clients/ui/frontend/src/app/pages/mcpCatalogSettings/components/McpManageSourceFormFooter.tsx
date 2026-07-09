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
import { MCP_ERROR_MESSAGES, MCP_BUTTON_LABELS } from '~/app/pages/mcpCatalogSettings/constants';

type McpManageSourceFormFooterProps = {
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

const McpManageSourceFormFooter: React.FC<McpManageSourceFormFooterProps> = ({
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
          <Alert variant="danger" isInline title={MCP_ERROR_MESSAGES.SAVE_FAILED}>
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
                id="mcp-submit-button"
                data-testid="mcp-submit-button"
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
                data-testid="mcp-preview-button"
              >
                {MCP_BUTTON_LABELS.PREVIEW}
              </Button>
            </ActionListItem>
            <ActionListItem>
              <Button
                isDisabled={isSubmitting || isPreviewLoading}
                variant="link"
                id="mcp-cancel-button"
                data-testid="mcp-cancel-button"
                onClick={onCancel}
              >
                {MCP_BUTTON_LABELS.CANCEL}
              </Button>
            </ActionListItem>
          </ActionListGroup>
        </ActionList>
      </StackItem>
    </Stack>
  </PageSection>
);

export default McpManageSourceFormFooter;
