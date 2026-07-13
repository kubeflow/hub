import {
  mcpCatalogSettings,
  mcpManageSourcePage,
} from '~/__tests__/cypress/cypress/pages/mcpCatalogSettings';
import {
  mockMcpCatalogSourceConfigList,
  mockMcpCatalogSourceConfig,
} from '~/__mocks__/mockMcpCatalogSourceConfigList';
import type { McpCatalogSourceConfig } from '~/app/mcpServerCatalogTypes';
import { McpCatalogSourceType } from '~/app/mcpServerCatalogTypes';

const NAMESPACE = 'kubeflow';
const userMock = {
  data: {
    userId: 'user@example.com',
    clusterAdmin: true,
  },
};

const setupMocks = () => {
  cy.intercept('GET', '/model-registry/api/v1/namespaces', {
    data: [{ metadata: { name: NAMESPACE } }],
  });
  cy.intercept('GET', '/model-registry/api/v1/user', userMock);
  cy.intercept('GET', '/model-registry/api/v1/settings/mcp_catalog/source_configs', {
    data: { catalogs: [] },
  });
};

const interceptSourceConfig = (id: string, overrides: Partial<McpCatalogSourceConfig> = {}) => {
  cy.intercept('GET', `/model-registry/api/v1/settings/mcp_catalog/source_configs/${id}`, {
    data: mockMcpCatalogSourceConfig({ id, isDefault: false, yaml: 'source: test', ...overrides }),
  });
};

describe('MCP Catalog Settings', () => {
  beforeEach(() => {
    setupMocks();
    cy.intercept('GET', '**/settings/mcp_catalog/source_configs*', {
      data: { catalogs: [] },
    });
  });

  it('should display the empty state with title and description', () => {
    mcpCatalogSettings.visit();
    mcpCatalogSettings.shouldBeEmpty();
    mcpCatalogSettings.findEmptyState().should('contain', 'No MCP sources');
    mcpCatalogSettings
      .findEmptyState()
      .should('contain', 'No MCP sources have been configured. Add a source to get started.');
  });

  it('should display the "Add a source" button in empty state', () => {
    mcpCatalogSettings.visit();
    mcpCatalogSettings.findAddSourceButton().should('be.visible');
    mcpCatalogSettings.findAddSourceButton().should('contain', 'Add a source');
  });

  it('should navigate to add-source page when button is clicked', () => {
    mcpCatalogSettings.visit();
    mcpCatalogSettings.findAddSourceButton().click();
    mcpManageSourcePage.findAddSourceTitle();
    mcpManageSourcePage.findAddSourceDescription();
    mcpManageSourcePage.findBreadcrumb().should('exist');
    mcpManageSourcePage.findBreadcrumbAction().should('contain', 'Add a source');
  });
});

describe('MCP Manage Source Page - Add Source Mode', () => {
  beforeEach(() => {
    setupMocks();
  });

  describe('Page layout and navigation', () => {
    it('should display add source page with correct breadcrumb', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findAddSourceTitle();
      mcpManageSourcePage.findBreadcrumb().should('exist');
      mcpManageSourcePage.findBreadcrumbAction().should('contain', 'Add a source');
    });

    it('should show correct page description', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findAddSourceDescription();
    });

    it('should navigate back to settings from breadcrumb', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findBreadcrumb().click({ force: true });
      mcpCatalogSettings.findHeading();
    });

    it('should navigate back to settings from cancel button', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findCancelButton().click();
      mcpCatalogSettings.findHeading();
    });

    it('should have breadcrumb link pointing to settings page', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findBreadcrumb().should('have.attr', 'href', '/mcp-catalog-settings');
    });
  });

  describe('Form fields display', () => {
    it('should display all required form fields', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findNameInput().should('exist');
      mcpManageSourcePage.findYamlSection().should('exist');
      mcpManageSourcePage.findYamlContentInput().should('exist');
      mcpManageSourcePage.findEnableSourceCheckbox().should('exist');
      mcpManageSourcePage.findSubmitButton().should('exist');
      mcpManageSourcePage.findPreviewButton().should('exist');
      mcpManageSourcePage.findCancelButton().should('exist');
    });

    it('should display server visibility section (collapsed by default)', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findServerVisibilitySection().should('exist');
      mcpManageSourcePage.findIncludedServersInput().should('not.exist');
      mcpManageSourcePage.findExcludedServersInput().should('not.exist');
    });

    it('should display the MCP catalog preview panel', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findPreviewPanel().should('exist');
      mcpManageSourcePage.findPreviewPanelTitle().should('be.visible');
      mcpManageSourcePage.findPreviewPanelEmptyMessage().should('be.visible');
    });

    it('should display "View expected file format" link', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findExpectedFormatLink().should('exist');
      mcpManageSourcePage.findExpectedFormatLink().should('contain', 'View expected file format');
    });
  });

  describe('Form validation', () => {
    it('should have Add button disabled by default', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findSubmitButton().should('be.disabled');
    });

    it('should have Preview button disabled by default', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findPreviewButton().should('be.disabled');
    });

    it('should show validation error when name field is empty and touched', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findNameInput().focus().blur();
      mcpManageSourcePage.findNameError().should('exist');
      mcpManageSourcePage.findNameError().should('contain', 'Name is required');
    });

    it('should show validation error for YAML content when touched', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findYamlContentInput().focus().blur();
      mcpManageSourcePage.findYamlContentError().should('exist');
      mcpManageSourcePage.findYamlContentError().should('contain', 'YAML content is required');
    });

    it('should clear name validation error when name is filled', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findNameInput().focus().blur();
      mcpManageSourcePage.findNameError().should('exist');
      mcpManageSourcePage.fillSourceName('Test Source');
      mcpManageSourcePage.findNameError().should('not.exist');
    });

    it('should clear YAML validation error when content is filled', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findYamlContentInput().focus().blur();
      mcpManageSourcePage.findYamlContentError().should('exist');
      mcpManageSourcePage.fillYamlContent('source: test\nmcp_servers:\n  - name: server1');
      mcpManageSourcePage.findYamlContentError().should('not.exist');
    });

    it('should enable Add button when all required fields are filled', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillSourceName('Test Source');
      mcpManageSourcePage.fillYamlContent('source: test\nmcp_servers:\n  - name: server1');
      mcpManageSourcePage.findSubmitButton().should('not.be.disabled');
    });

    it('should enable Preview button when YAML content is filled', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillYamlContent('source: test\nmcp_servers:\n  - name: server1');
      mcpManageSourcePage.findPreviewButton().should('not.be.disabled');
    });

    it('should keep Add button disabled when only name is filled (no YAML)', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillSourceName('Test Source');
      mcpManageSourcePage.findSubmitButton().should('be.disabled');
    });

    it('should keep Add button disabled when only YAML is filled (no name)', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillYamlContent('source: test');
      mcpManageSourcePage.findSubmitButton().should('be.disabled');
    });
  });

  describe('Server visibility section', () => {
    it('should expand and collapse server visibility section', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findIncludedServersInput().should('not.exist');
      mcpManageSourcePage.findExcludedServersInput().should('not.exist');

      mcpManageSourcePage.toggleServerVisibility();
      mcpManageSourcePage.findIncludedServersInput().should('exist');
      mcpManageSourcePage.findExcludedServersInput().should('exist');

      mcpManageSourcePage.toggleServerVisibility();
      mcpManageSourcePage.findIncludedServersInput().should('not.exist');
      mcpManageSourcePage.findExcludedServersInput().should('not.exist');
    });

    it('should allow entering server filter values', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.toggleServerVisibility();

      mcpManageSourcePage.fillIncludedServers('Kubernetes, GitHub');
      mcpManageSourcePage.fillExcludedServers('*preview*');

      mcpManageSourcePage.findIncludedServersInput().should('have.value', 'Kubernetes, GitHub');
      mcpManageSourcePage.findExcludedServersInput().should('have.value', '*preview*');
    });

    it('should show correct placeholder text for included servers', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.toggleServerVisibility();
      mcpManageSourcePage
        .findIncludedServersInput()
        .should('have.attr', 'placeholder', 'Example: Kubernetes, GitHub, PostgreSQL');
    });

    it('should show correct placeholder text for excluded servers', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.toggleServerVisibility();
      mcpManageSourcePage
        .findExcludedServersInput()
        .should('have.attr', 'placeholder', 'Example: *preview*');
    });
  });

  describe('Enable source checkbox', () => {
    it('should have enable source checkbox checked by default', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findEnableSourceCheckbox().should('be.checked');
    });

    it('should allow toggling enable source checkbox', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findEnableSourceCheckbox().should('be.checked');
      mcpManageSourcePage.toggleEnableSource();
      mcpManageSourcePage.findEnableSourceCheckbox().should('not.be.checked');
      mcpManageSourcePage.toggleEnableSource();
      mcpManageSourcePage.findEnableSourceCheckbox().should('be.checked');
    });
  });

  describe('Preview panel', () => {
    it('should have three preview buttons (footer, header, panel)', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findPreviewButton().should('exist');
      mcpManageSourcePage.findPreviewButtonHeader().should('exist');
      mcpManageSourcePage.findPreviewButtonPanel().should('exist');
    });

    it('should have all three preview buttons disabled by default', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findPreviewButton().should('be.disabled');
      mcpManageSourcePage.findPreviewButtonHeader().should('be.disabled');
      mcpManageSourcePage.findPreviewButtonPanel().should('be.disabled');
    });

    it('should enable all preview buttons when YAML content is filled', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillYamlContent('source: test\nmcp_servers:\n  - name: server1');
      mcpManageSourcePage.findPreviewButton().should('not.be.disabled');
      mcpManageSourcePage.findPreviewButtonHeader().should('not.be.disabled');
      mcpManageSourcePage.findPreviewButtonPanel().should('not.be.disabled');
    });
  });

  describe('Expected file format drawer', () => {
    it('should open drawer when link is clicked', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findExpectedFormatLink().click();
      mcpManageSourcePage.findExpectedFormatDrawerTitle().should('exist');
      mcpManageSourcePage.findExpectedFormatDrawerTitle().should('contain', 'Expected file format');
    });

    it('should close drawer when close button is clicked', () => {
      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.findExpectedFormatLink().click();
      mcpManageSourcePage.findExpectedFormatDrawerTitle().should('exist');
      mcpManageSourcePage.findExpectedFormatDrawerClose().click();
      mcpManageSourcePage.findExpectedFormatDrawerTitle().should('not.exist');
    });
  });

  describe('Form submission', () => {
    it('should submit add source form with YAML content', () => {
      cy.intercept('POST', '/model-registry/api/v1/settings/mcp_catalog/source_configs', {
        data: mockMcpCatalogSourceConfig({}),
      }).as('addMcpSource');

      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillSourceName('My MCP Source');
      mcpManageSourcePage.fillYamlContent('source: test\nmcp_servers:\n  - name: Kubernetes');

      mcpManageSourcePage.findSubmitButton().should('be.enabled');
      mcpManageSourcePage.findSubmitButton().should('contain', 'Add');
      mcpManageSourcePage.findSubmitButton().click();

      cy.wait('@addMcpSource').then((interception) => {
        expect(interception.request.body).to.eql({
          data: {
            id: 'my_mcp_source',
            name: 'My MCP Source',
            type: McpCatalogSourceType.YAML,
            enabled: true,
            isDefault: false,
            yaml: 'source: test\nmcp_servers:\n  - name: Kubernetes',
            includedServers: [],
            excludedServers: [],
          },
        });
      });
    });

    it('should submit add source form with server visibility filters', () => {
      cy.intercept('POST', '/model-registry/api/v1/settings/mcp_catalog/source_configs', {
        data: mockMcpCatalogSourceConfig({}),
      }).as('addMcpSourceWithFilters');

      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillSourceName('Filtered Source');
      mcpManageSourcePage.fillYamlContent('source: test\nmcp_servers:\n  - name: server1');

      mcpManageSourcePage.toggleServerVisibility();
      mcpManageSourcePage.fillIncludedServers('Kubernetes, GitHub*');
      mcpManageSourcePage.fillExcludedServers('*preview*, *experimental*');

      mcpManageSourcePage.findSubmitButton().click();

      cy.wait('@addMcpSourceWithFilters').then((interception) => {
        expect(interception.request.body).to.eql({
          data: {
            id: 'filtered_source',
            name: 'Filtered Source',
            type: McpCatalogSourceType.YAML,
            enabled: true,
            isDefault: false,
            yaml: 'source: test\nmcp_servers:\n  - name: server1',
            includedServers: ['Kubernetes', 'GitHub*'],
            excludedServers: ['*preview*', '*experimental*'],
          },
        });
      });
    });

    it('should submit add source form with source disabled', () => {
      cy.intercept('POST', '/model-registry/api/v1/settings/mcp_catalog/source_configs', {
        data: mockMcpCatalogSourceConfig({}),
      }).as('addMcpSourceDisabled');

      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillSourceName('Disabled Source');
      mcpManageSourcePage.fillYamlContent('source: disabled\nmcp_servers:\n  - name: srv');
      mcpManageSourcePage.toggleEnableSource();

      mcpManageSourcePage.findSubmitButton().click();

      cy.wait('@addMcpSourceDisabled').then((interception) => {
        expect(interception.request.body.data.enabled).to.eql(false);
      });
    });

    it('should navigate to settings page after successful submission', () => {
      cy.intercept('POST', '/model-registry/api/v1/settings/mcp_catalog/source_configs', {
        data: mockMcpCatalogSourceConfig({}),
      }).as('addMcpSourceSuccess');

      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillSourceName('New Source');
      mcpManageSourcePage.fillYamlContent('source: test');

      mcpManageSourcePage.findSubmitButton().click();
      cy.wait('@addMcpSourceSuccess');
      mcpCatalogSettings.findHeading();
    });

    it('should show error alert when submission fails', () => {
      cy.intercept('POST', '/model-registry/api/v1/settings/mcp_catalog/source_configs', {
        statusCode: 500,
        body: { message: 'Internal server error' },
      }).as('addMcpSourceFail');

      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillSourceName('Failing Source');
      mcpManageSourcePage.fillYamlContent('source: test');

      mcpManageSourcePage.findSubmitButton().click();
      cy.wait('@addMcpSourceFail');
      cy.contains('Failed to save source').should('exist');
    });

    it('should disable submit button while submitting', () => {
      cy.intercept('POST', '/model-registry/api/v1/settings/mcp_catalog/source_configs', (req) => {
        req.reply({
          statusCode: 200,
          delay: 1000,
          body: { data: mockMcpCatalogSourceConfig({}) },
        });
      }).as('addMcpSourceSlow');

      mcpManageSourcePage.visitAddSource();
      mcpManageSourcePage.fillSourceName('Slow Source');
      mcpManageSourcePage.fillYamlContent('source: test');

      mcpManageSourcePage.findSubmitButton().click();
      mcpManageSourcePage.findSubmitButton().should('be.disabled');
    });
  });
});

describe('MCP Manage Source Page - Manage Source Mode', () => {
  beforeEach(() => {
    setupMocks();
  });

  describe('Page layout', () => {
    it('should display manage source page with correct breadcrumb', () => {
      interceptSourceConfig('mcp_source_2', { name: 'MCP Source 2' });

      mcpManageSourcePage.visitManageSource('mcp_source_2');
      mcpManageSourcePage.findManageSourceTitle();
      mcpManageSourcePage.findBreadcrumb().should('exist');
      mcpManageSourcePage.findBreadcrumbAction().should('contain', 'Manage source');
    });

    it('should show correct description for manage mode', () => {
      interceptSourceConfig('mcp_source_2', { name: 'MCP Source 2' });

      mcpManageSourcePage.visitManageSource('mcp_source_2');
      mcpManageSourcePage.findManageSourceDescription();
    });

    it('should show Save button instead of Add button', () => {
      interceptSourceConfig('mcp_source_2', { name: 'MCP Source 2' });

      mcpManageSourcePage.visitManageSource('mcp_source_2');
      mcpManageSourcePage.findSubmitButton().should('exist');
      mcpManageSourcePage.findSubmitButton().should('contain', 'Save');
    });
  });

  describe('Pre-populated form data', () => {
    it('should pre-populate form with existing source data', () => {
      interceptSourceConfig('mcp_source_2', {
        name: 'MCP Source 2',
        enabled: true,
        yaml: 'source: my-source\nmcp_servers:\n  - name: Kubernetes',
        includedServers: ['Kubernetes', 'GitHub'],
        excludedServers: ['*preview*'],
      });

      mcpManageSourcePage.visitManageSource('mcp_source_2');
      mcpManageSourcePage.findNameInput().should('have.value', 'MCP Source 2');
      mcpManageSourcePage.findEnableSourceCheckbox().should('be.checked');
    });

    it('should expand server visibility when existing filters present', () => {
      interceptSourceConfig('mcp_source_2', {
        name: 'MCP Source 2',
        includedServers: ['Kubernetes'],
        excludedServers: [],
      });

      mcpManageSourcePage.visitManageSource('mcp_source_2');
      mcpManageSourcePage.findIncludedServersInput().should('exist');
      mcpManageSourcePage.findIncludedServersInput().should('have.value', 'Kubernetes');
    });

    it('should have Save button enabled for valid existing data', () => {
      interceptSourceConfig('mcp_source_2', { name: 'MCP Source 2' });

      mcpManageSourcePage.visitManageSource('mcp_source_2');
      mcpManageSourcePage.findSubmitButton().should('be.enabled');
    });
  });

  describe('Default source management', () => {
    it('should allow managing default source (Save button enabled)', () => {
      interceptSourceConfig('default_source', {
        name: 'Red Hat validated MCP servers',
        isDefault: true,
        enabled: true,
        includedServers: [],
        excludedServers: [],
      });

      mcpManageSourcePage.visitManageSource('default_source');
      mcpManageSourcePage.findNameInput().should('have.value', 'Red Hat validated MCP servers');
      mcpManageSourcePage.findSubmitButton().should('be.enabled');
      mcpManageSourcePage.findSubmitButton().should('contain', 'Save');
    });

    it('should not show YAML section for default sources', () => {
      interceptSourceConfig('default_source', {
        name: 'Red Hat validated MCP servers',
        isDefault: true,
      });

      mcpManageSourcePage.visitManageSource('default_source');
      mcpManageSourcePage.findYamlSection().should('not.exist');
    });
  });

  describe('Form submission (edit mode)', () => {
    it('should successfully update a non-default source', () => {
      interceptSourceConfig('mcp_source_2', {
        name: 'MCP Source 2',
        includedServers: ['server1'],
        excludedServers: [],
        enabled: false,
        yaml: 'source: test\nmcp_servers:\n  - name: server1',
      });

      cy.intercept('PATCH', '/model-registry/api/v1/settings/mcp_catalog/source_configs/*', {
        statusCode: 200,
        body: {
          data: mockMcpCatalogSourceConfig({ id: 'mcp_source_2', isDefault: false }),
        },
      }).as('updateMcpSource');

      mcpManageSourcePage.visitManageSource('mcp_source_2');
      mcpManageSourcePage.findNameInput().should('have.value', 'MCP Source 2');

      mcpManageSourcePage.findIncludedServersInput().should('exist');
      mcpManageSourcePage.findIncludedServersInput().type(', Kubernetes*');
      mcpManageSourcePage.findExcludedServersInput().type('*preview*');
      mcpManageSourcePage.findEnableSourceCheckbox().should('not.be.checked');
      mcpManageSourcePage.findEnableSourceCheckbox().check();

      mcpManageSourcePage.findSubmitButton().should('be.enabled');
      mcpManageSourcePage.findSubmitButton().should('have.text', 'Save');
      mcpManageSourcePage.findSubmitButton().click();

      cy.wait('@updateMcpSource').then((interception) => {
        expect(interception.request.body).to.eql({
          data: {
            name: 'MCP Source 2',
            type: McpCatalogSourceType.YAML,
            includedServers: ['server1', 'Kubernetes*'],
            excludedServers: ['*preview*'],
            enabled: true,
            isDefault: false,
            yaml: 'source: test\nmcp_servers:\n  - name: server1',
          },
        });
      });
    });

    it('should submit only allowed fields for default source', () => {
      interceptSourceConfig('default_source', {
        name: 'Red Hat validated MCP servers',
        isDefault: true,
        enabled: true,
        includedServers: [],
        excludedServers: [],
      });

      cy.intercept('PATCH', '/model-registry/api/v1/settings/mcp_catalog/source_configs/*', {
        statusCode: 200,
        body: {
          data: mockMcpCatalogSourceConfig({ id: 'default_source', isDefault: true }),
        },
      }).as('updateDefaultMcpSource');

      mcpManageSourcePage.visitManageSource('default_source');

      mcpManageSourcePage.fillIncludedServers('Kubernetes, GitHub');
      mcpManageSourcePage.fillExcludedServers('*experimental*');
      mcpManageSourcePage.findEnableSourceCheckbox().uncheck();

      mcpManageSourcePage.findSubmitButton().click();

      cy.wait('@updateDefaultMcpSource').then((interception) => {
        expect(interception.request.body).to.eql({
          data: {
            enabled: false,
            includedServers: ['Kubernetes', 'GitHub'],
            excludedServers: ['*experimental*'],
          },
        });
      });
    });

    it('should display catalog YAML file and server count for default source', () => {
      interceptSourceConfig('default_source', {
        name: 'Red Hat validated MCP servers',
        isDefault: true,
        enabled: true,
        yamlCatalogPath: 'redhat-mcp-servers-catalog.yaml',
        includedServers: [],
        excludedServers: [],
      });

      cy.intercept('POST', '/model-registry/api/v1/settings/model_catalog/source_preview*', {
        data: {
          items: [
            { name: 'Kubernetes', included: true },
            { name: 'GitHub', included: true },
            { name: 'PostgreSQL', included: true },
          ],
          summary: { totalAssets: 3, includedAssets: 3, excludedAssets: 0 },
          nextPageToken: '',
          pageSize: 10,
          size: 3,
        },
      }).as('previewDefaultSource');

      cy.visit('/mcp-catalog-settings/manage-source/default_source');
      cy.wait('@previewDefaultSource');
      cy.findByTestId('app-page-title').should('exist');

      cy.findByTestId('mcp-catalog-yaml-file').should(
        'have.value',
        'redhat-mcp-servers-catalog.yaml',
      );
      cy.findByTestId('mcp-servers-count').should('contain.text', '3 servers');
    });

    it('should not display catalog YAML file for non-default source', () => {
      interceptSourceConfig('user_source', {
        name: 'My Custom Source',
        enabled: true,
        yaml: 'source: custom\nmcp_servers:\n  - name: Custom Server',
        yamlCatalogPath: undefined,
      });

      mcpManageSourcePage.visitManageSource('user_source');

      cy.findByTestId('mcp-catalog-yaml-file').should('not.exist');
    });

    it('should navigate to settings page after successful update', () => {
      interceptSourceConfig('mcp_source_2', { name: 'MCP Source 2' });

      cy.intercept('PATCH', '/model-registry/api/v1/settings/mcp_catalog/source_configs/*', {
        statusCode: 200,
        body: { data: mockMcpCatalogSourceConfig({}) },
      }).as('updateSuccess');

      mcpManageSourcePage.visitManageSource('mcp_source_2');
      mcpManageSourcePage.findSubmitButton().click();
      cy.wait('@updateSuccess');
      mcpCatalogSettings.findHeading();
    });

    it('should show error when update fails', () => {
      interceptSourceConfig('mcp_source_2', { name: 'MCP Source 2' });

      cy.intercept('PATCH', '/model-registry/api/v1/settings/mcp_catalog/source_configs/*', {
        statusCode: 500,
        body: { message: 'Server error' },
      }).as('updateFail');

      mcpManageSourcePage.visitManageSource('mcp_source_2');
      mcpManageSourcePage.findSubmitButton().click();
      cy.wait('@updateFail');
      cy.contains('Failed to save source').should('exist');
    });
  });
});

describe('MCP Catalog Source Configs Table', () => {
  beforeEach(() => {
    setupMocks();
  });

  describe('Table rendering', () => {
    it('should display the table with sources when data exists', () => {
      const mockData = mockMcpCatalogSourceConfigList();
      cy.intercept('GET', '**/settings/mcp_catalog/source_configs*', {
        data: mockData,
      });

      mcpCatalogSettings.visit();
      mcpCatalogSettings.shouldHaveRows();
      mcpCatalogSettings.findRows().should('have.length', 3);
    });

    it('should display source names and types', () => {
      const mockData = mockMcpCatalogSourceConfigList();
      cy.intercept('GET', '**/settings/mcp_catalog/source_configs*', {
        data: mockData,
      });

      mcpCatalogSettings.visit();
      mcpCatalogSettings.getRow('Sample MCP source 1').find().should('exist');
      mcpCatalogSettings.getRow('MCP Source 2').find().should('exist');
      mcpCatalogSettings.getRow('Sample MCP source 4').find().should('exist');
    });

    it('should show YAML file as the source type label', () => {
      const mockData = mockMcpCatalogSourceConfigList({
        catalogs: [
          mockMcpCatalogSourceConfig({
            id: 'yaml-source',
            name: 'YAML Source',
            type: McpCatalogSourceType.YAML,
            isDefault: false,
          }),
        ],
      });
      cy.intercept('GET', '**/settings/mcp_catalog/source_configs*', {
        data: mockData,
      });

      mcpCatalogSettings.visit();
      mcpCatalogSettings.getRow('YAML Source').find().should('contain', 'YAML file');
    });

    it('should NOT show kebab menu for default sources', () => {
      const mockData = mockMcpCatalogSourceConfigList({
        catalogs: [
          mockMcpCatalogSourceConfig({
            id: 'default-source',
            name: 'Default Source',
            isDefault: true,
          }),
        ],
      });
      cy.intercept('GET', '**/settings/mcp_catalog/source_configs*', {
        data: mockData,
      });

      mcpCatalogSettings.visit();
      const row = mcpCatalogSettings.getRow('Default Source');
      row.find().within(() => {
        cy.get('[data-testid*="source-actions"]').should('not.exist');
      });
    });

    it('should show kebab menu for non-default sources', () => {
      const mockData = mockMcpCatalogSourceConfigList({
        catalogs: [
          mockMcpCatalogSourceConfig({
            id: 'custom-source',
            name: 'Custom Source',
            isDefault: false,
          }),
        ],
      });
      cy.intercept('GET', '**/settings/mcp_catalog/source_configs*', {
        data: mockData,
      });

      mcpCatalogSettings.visit();
      const row = mcpCatalogSettings.getRow('Custom Source');
      row.findKebab().should('exist');
    });
  });

  describe('Sort functionality', () => {
    it('should have sortable source name column header', () => {
      const mockData = mockMcpCatalogSourceConfigList({
        catalogs: [
          mockMcpCatalogSourceConfig({ id: 'c', name: 'Charlie', isDefault: false }),
          mockMcpCatalogSourceConfig({ id: 'a', name: 'Alpha', isDefault: false }),
        ],
      });
      cy.intercept('GET', '**/settings/mcp_catalog/source_configs*', {
        data: mockData,
      });

      mcpCatalogSettings.visit();
      mcpCatalogSettings
        .findTable()
        .find('thead th')
        .contains('Source name')
        .parents('th')
        .should('have.attr', 'aria-sort');
    });
  });

  describe('Delete functionality', () => {
    it('should open delete modal when clicking Delete source from kebab menu', () => {
      const mockData = mockMcpCatalogSourceConfigList({
        catalogs: [
          mockMcpCatalogSourceConfig({
            id: 'deletable-source',
            name: 'Deletable Source',
            isDefault: false,
          }),
        ],
      });
      cy.intercept('GET', '**/settings/mcp_catalog/source_configs*', {
        data: mockData,
      });

      mcpCatalogSettings.visit();
      const row = mcpCatalogSettings.getRow('Deletable Source');
      row.findKebab().click();
      cy.findByRole('menuitem', { name: /delete source/i }).click();

      cy.findByTestId('mcp-delete-source-modal').should('exist');
    });

    it('should close delete modal when cancel is clicked', () => {
      const mockData = mockMcpCatalogSourceConfigList({
        catalogs: [
          mockMcpCatalogSourceConfig({
            id: 'deletable-source',
            name: 'Deletable Source',
            isDefault: false,
          }),
        ],
      });
      cy.intercept('GET', '**/settings/mcp_catalog/source_configs*', {
        data: mockData,
      });

      mcpCatalogSettings.visit();
      const row = mcpCatalogSettings.getRow('Deletable Source');
      row.findKebab().click();
      cy.findByRole('menuitem', { name: /delete source/i }).click();

      cy.findByTestId('mcp-delete-source-modal').should('exist');
      cy.findByRole('button', { name: /cancel/i }).click();
      cy.findByTestId('mcp-delete-source-modal').should('not.exist');
    });
  });

  describe('Error status', () => {
    it('should display error status for source with error', () => {
      const mockData = mockMcpCatalogSourceConfigList({
        catalogs: [
          mockMcpCatalogSourceConfig({
            id: 'error-source',
            name: 'Error Source',
            isDefault: false,
            enabled: true,
          }),
        ],
      });
      cy.intercept('GET', '**/settings/mcp_catalog/source_configs*', {
        data: mockData,
      });
      cy.intercept('GET', '**/model_catalog/sources*', {
        data: {
          items: [
            {
              id: 'error-source',
              status: 'error',
              error: 'Connection failed',
              name: 'Error Source',
              enabled: true,
            },
          ],
          size: 1,
          pageSize: 10,
          nextPageToken: '',
        },
      });

      mcpCatalogSettings.visit();
      const row = mcpCatalogSettings.getRow('Error Source');
      row.find().should('contain', 'Failed');
    });
  });
});
