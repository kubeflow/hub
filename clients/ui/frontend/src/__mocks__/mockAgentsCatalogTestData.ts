/* eslint-disable camelcase */
import type { Agent, AgentList } from '~/app/agentsCatalogTypes';
import type { AgentsCatalogFilterOptionsList } from '~/app/pages/agentsCatalog/types/agentsCatalogFilterOptions';

export const mockAgent = (partial?: Partial<Agent>): Agent => ({
  id: '1',
  name: 'research-assistant',
  displayName: 'Research Assistant',
  description: 'An agent that performs research tasks using web search and summarization.',
  framework: 'langgraph',
  sourceId: 'agent-templates-source',
  labels: ['Web search', 'General purpose'],
  logo: undefined,
  env: [{ name: 'OPENAI_API_KEY', required: true, description: 'API key for the LLM provider' }],
  repositoryUrl: 'https://github.com/example/research-assistant',
  readme: '# Research Assistant\n\n### Overview\n\nAn agent for automated research.',
  ...partial,
});

export const mockAgentList = (partial?: Partial<AgentList>): AgentList => ({
  items: [mockAgent()],
  pageSize: 10,
  size: 1,
  nextPageToken: '',
  ...partial,
});

export const mockAgentsCatalogFilterOptions = (
  partial?: Partial<AgentsCatalogFilterOptionsList>,
): AgentsCatalogFilterOptionsList => ({
  filters: {
    framework: {
      type: 'string',
      values: ['a2a', 'autogen', 'claude-code', 'crewai', 'google-adk', 'langgraph'],
    },
  },
  ...partial,
});
