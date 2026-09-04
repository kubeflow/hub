/* eslint-disable camelcase */
import { mockModArchResponse } from 'mod-arch-core';
import {
  mockCatalogLabel,
  mockCatalogSource,
  mockCatalogSourceList,
  mockAgent,
  mockAgentList,
  mockAgentsCatalogFilterOptions,
} from '~/__mocks__';
import type { Agent } from '~/app/agentsCatalogTypes';
import type { CatalogSource } from '~/app/shared/types/catalogTypes';
import { MODEL_CATALOG_API_VERSION } from '~/__tests__/cypress/cypress/support/commands/api';

const defaultSources: CatalogSource[] = [
  mockCatalogSource({
    id: 'agent-templates-source',
    name: 'Agent Templates',
    labels: ['agent_templates'],
  }),
];

type InitInterceptsConfig = {
  sources?: CatalogSource[];
  agentsPerCategory?: number;
  includeFilterOptions?: boolean;
};

export const interceptAgentSources = (sources: CatalogSource[]): void => {
  cy.interceptApi(
    `GET /api/:apiVersion/model_catalog/sources`,
    { path: { apiVersion: MODEL_CATALOG_API_VERSION }, query: { assetType: 'agents' } },
    mockCatalogSourceList({ items: sources }),
  );
};

export const interceptAgentLabels = (): void => {
  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(`/api/${MODEL_CATALOG_API_VERSION}/model_catalog/labels`),
    },
    mockModArchResponse({
      items: [
        mockCatalogLabel({
          name: 'agent_templates',
          displayName: 'Agent templates',
          description: 'Pre-built agent templates from the agentic starter kits collection.',
        }),
      ],
      size: 1,
      pageSize: 10,
      nextPageToken: '',
    }),
  );
};

const buildAgentsForSources = (sources: CatalogSource[], agentsPerCategory: number): Agent[] =>
  sources.flatMap((source) =>
    source.labels.flatMap((label) =>
      Array.from({ length: agentsPerCategory }, (_, i) =>
        mockAgent({
          id: `${label}-agent-${i + 1}`,
          name: `${label}-agent-${i + 1}`,
          displayName: `${label} Agent ${i + 1}`,
          sourceId: source.id,
        }),
      ),
    ),
  );

export const interceptAgentsByLabel = (
  sources: CatalogSource[],
  agentsPerCategory: number,
): void => {
  sources.forEach((source) => {
    source.labels.forEach((label) => {
      const agents: Agent[] = Array.from({ length: agentsPerCategory }, (_, i) =>
        mockAgent({
          id: `${label}-agent-${i + 1}`,
          name: `${label}-agent-${i + 1}`,
          displayName: `${label} Agent ${i + 1}`,
          sourceId: source.id,
        }),
      );

      cy.interceptApi(
        `GET /api/:apiVersion/agent_catalog/agents`,
        {
          path: { apiVersion: MODEL_CATALOG_API_VERSION },
          query: { sourceLabel: label },
        },
        mockAgentList({ items: agents, size: agents.length }),
      );
    });
  });
};

export const interceptAgentFilterOptions = (): void => {
  cy.interceptApi(
    `GET /api/:apiVersion/agent_catalog/agents_filter_options`,
    { path: { apiVersion: MODEL_CATALOG_API_VERSION } },
    mockAgentsCatalogFilterOptions(),
  );
};

export const interceptSingleAgent = (agent: Agent): void => {
  cy.intercept(
    {
      method: 'GET',
      url: new RegExp(
        `/model-registry/api/${MODEL_CATALOG_API_VERSION}/agent_catalog/agents/[^/]+$`,
      ),
    },
    mockModArchResponse(agent),
  ).as('getSingleAgent');
};

export const initAgentsCatalogIntercepts = ({
  sources = defaultSources,
  agentsPerCategory = 0,
  includeFilterOptions = true,
}: InitInterceptsConfig = {}): void => {
  interceptAgentSources(sources);
  interceptAgentLabels();

  // Generic intercept (no sourceLabel query) registered FIRST so that per-label
  // intercepts registered after take priority (Cypress: last-registered wins).
  const allAgents = buildAgentsForSources(sources, agentsPerCategory);
  cy.interceptApi(
    `GET /api/:apiVersion/agent_catalog/agents`,
    {
      path: { apiVersion: MODEL_CATALOG_API_VERSION },
    },
    mockAgentList({ items: allAgents, size: allAgents.length }),
  );

  interceptAgentsByLabel(sources, agentsPerCategory);

  if (includeFilterOptions) {
    interceptAgentFilterOptions();
  }
};
