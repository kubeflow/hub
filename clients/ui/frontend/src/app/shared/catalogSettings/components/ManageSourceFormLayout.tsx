import * as React from 'react';
import { Form, Sidebar, SidebarContent, SidebarPanel } from '@patternfly/react-core';

export type ManageSourceFormLayoutProps = {
  /** Form body sections rendered inside the left sidebar content. */
  children: React.ReactNode;
  /** Preview panel rendered in the right sidebar. */
  previewPanel: React.ReactNode;
  /** Sticky footer (submit / preview / cancel). */
  footer: React.ReactNode;
};

/**
 * Shared manage-source layout: form on the left, preview on the right, sticky footer.
 * Domain forms own submit logic and section composition; this only owns the chrome.
 */
const ManageSourceFormLayout: React.FC<ManageSourceFormLayoutProps> = ({
  children,
  previewPanel,
  footer,
}) => (
  <>
    <Sidebar hasBorder isPanelRight hasGutter>
      <SidebarContent>
        <Form isWidthLimited>{children}</Form>
      </SidebarContent>
      <SidebarPanel width={{ default: 'width_50' }}>{previewPanel}</SidebarPanel>
    </Sidebar>
    {footer}
  </>
);

export default ManageSourceFormLayout;
