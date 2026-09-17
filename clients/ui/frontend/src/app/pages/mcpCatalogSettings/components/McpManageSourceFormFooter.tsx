import * as React from 'react';
import { ManageSourceFormFooter } from '~/app/shared/catalogSettings';
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

const McpManageSourceFormFooter: React.FC<McpManageSourceFormFooterProps> = (props) => (
  <ManageSourceFormFooter
    {...props}
    saveFailedTitle={MCP_ERROR_MESSAGES.SAVE_FAILED}
    previewLabel={MCP_BUTTON_LABELS.PREVIEW}
    cancelLabel={MCP_BUTTON_LABELS.CANCEL}
    testIdPrefix="mcp-"
  />
);

export default McpManageSourceFormFooter;
