import {
  agentsCatalogUrl,
  getAgentsCatalogDetailsRoute,
} from '~/app/routes/agentsCatalog/agentsCatalog';

describe('agentsCatalog routes', () => {
  it('should return the base catalog URL', () => {
    expect(agentsCatalogUrl()).toBe('/agents-catalog');
  });

  it('should build encoded agent details path', () => {
    expect(getAgentsCatalogDetailsRoute('my-agent')).toBe('/agents-catalog/my-agent/overview');
    expect(getAgentsCatalogDetailsRoute('agent/with/slashes')).toBe(
      '/agents-catalog/agent%2Fwith%2Fslashes/overview',
    );
  });
});
