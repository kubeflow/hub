import * as React from 'react';
import { Link } from 'react-router-dom';
import {
  Breadcrumb,
  BreadcrumbItem,
  Drawer,
  DrawerContent,
  DrawerContentBody,
  DrawerPanelContent,
} from '@patternfly/react-core';
import { ApplicationsPage } from 'mod-arch-shared';

type ManageSourcePageShellProps = {
  listPageUrl: string;
  listPageLabel: string;
  breadcrumbLabel: string;
  breadcrumbTestId: string;
  title: string;
  description: string;
  errorMessage?: string;
  empty: boolean;
  loaded: boolean;
  isExpectedFormatDrawerOpen: boolean;
  drawerPanelContent: React.ReactNode;
  children: React.ReactNode;
};

const ManageSourcePageShell: React.FC<ManageSourcePageShellProps> = ({
  listPageUrl,
  listPageLabel,
  breadcrumbLabel,
  breadcrumbTestId,
  title,
  description,
  errorMessage,
  empty,
  loaded,
  isExpectedFormatDrawerOpen,
  drawerPanelContent,
  children,
}) => (
  <Drawer isExpanded={isExpectedFormatDrawerOpen}>
    <DrawerContent
      panelContent={
        <DrawerPanelContent isResizable defaultSize="50%">
          {drawerPanelContent}
        </DrawerPanelContent>
      }
    >
      <DrawerContentBody>
        <ApplicationsPage
          breadcrumb={
            <Breadcrumb>
              <BreadcrumbItem>
                <Link to={listPageUrl}>{listPageLabel}</Link>
              </BreadcrumbItem>
              <BreadcrumbItem data-testid={breadcrumbTestId} isActive>
                {breadcrumbLabel}
              </BreadcrumbItem>
            </Breadcrumb>
          }
          title={title}
          description={description}
          errorMessage={errorMessage}
          empty={empty}
          loaded={loaded}
          provideChildrenPadding
        >
          {children}
        </ApplicationsPage>
      </DrawerContentBody>
    </DrawerContent>
  </Drawer>
);

export default ManageSourcePageShell;
