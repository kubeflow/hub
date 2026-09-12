import React from 'react';
import {
  Brand,
  Button,
  Dropdown,
  DropdownItem,
  DropdownList,
  Masthead,
  MastheadBrand,
  MastheadContent,
  MastheadLogo,
  MastheadMain,
  MastheadToggle,
  MenuToggle,
  MenuToggleElement,
  PageToggleButton,
  Toolbar,
  ToolbarContent,
  ToolbarGroup,
  ToolbarItem,
} from '@patternfly/react-core';
import { BarsIcon, MoonIcon, SunIcon } from '@patternfly/react-icons';
import { useThemeContext } from 'mod-arch-kubeflow';
import { images as sharedImages } from 'mod-arch-shared';
import GlobalNamespaceSelector from './GlobalNamespaceSelector';
import { useCustomTheme } from '~/app/context/ThemeContext';

interface NavBarProps {
  username?: string;
  onLogout: () => void;
}

const NavBar: React.FC<NavBarProps> = ({ username, onLogout }) => {
  const { isMUITheme } = useThemeContext();
  const { effectiveTheme, toggleTheme } = useCustomTheme();

  const [userMenuOpen, setUserMenuOpen] = React.useState(false);

  const handleLogout = () => {
    setUserMenuOpen(false);
    onLogout();
  };

  const userMenuItems = [
    <DropdownItem key="logout" onClick={handleLogout}>
      Log out
    </DropdownItem>,
  ];

  return (
    <Masthead>
      <MastheadMain>
        <MastheadToggle>
          <PageToggleButton id="page-nav-toggle" variant="plain" aria-label="Dashboard navigation">
            <BarsIcon />
          </PageToggleButton>
        </MastheadToggle>
        {!isMUITheme ? (
          <MastheadBrand>
            <MastheadLogo component="a">
              <Brand
                src={sharedImages.logoLightThemePath}
                alt="Kubeflow"
                heights={{ default: '36px' }}
              />
            </MastheadLogo>
          </MastheadBrand>
        ) : null}
      </MastheadMain>
      <MastheadContent>
        <Toolbar>
          <ToolbarContent>
            <ToolbarGroup variant="action-group-plain" align={{ default: 'alignStart' }}>
              <ToolbarItem className="kubeflow-u-namespace-select">
                <GlobalNamespaceSelector />
              </ToolbarItem>
            </ToolbarGroup>
            <ToolbarGroup variant="action-group-plain" align={{ default: 'alignEnd' }}>
              <ToolbarItem>
                <Button
                  variant="plain"
                  aria-label="Toggle dark mode"
                  id="theme-toggle-button"
                  onClick={toggleTheme}
                  title={effectiveTheme === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
                  style={{ fontSize: '1.1rem', marginRight: '8px' }}
                >
                  {effectiveTheme === 'dark' ? (
                    <SunIcon style={{ color: '#f59e0b' }} />
                  ) : (
                    <MoonIcon style={{ color: '#64748b' }} />
                  )}
                </Button>
              </ToolbarItem>
              {username && (
                <ToolbarItem>
                  <Dropdown
                    popperProps={{ position: 'right' }}
                    onOpenChange={(isOpen) => setUserMenuOpen(isOpen)}
                    toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                      <MenuToggle
                        aria-label="User menu"
                        id="user-menu-toggle"
                        data-testid="user-menu-toggle-button"
                        ref={toggleRef}
                        onClick={() => setUserMenuOpen(!userMenuOpen)}
                        isExpanded={userMenuOpen}
                      >
                        {username}
                      </MenuToggle>
                    )}
                    isOpen={userMenuOpen}
                  >
                    <DropdownList>{userMenuItems}</DropdownList>
                  </Dropdown>
                </ToolbarItem>
              )}
            </ToolbarGroup>
          </ToolbarContent>
        </Toolbar>
      </MastheadContent>
    </Masthead>
  );
};

export default NavBar;
