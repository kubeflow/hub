import { parseAllowedTools } from '~/app/pages/skillCatalog/utils/skillCatalogUtils';

describe('parseAllowedTools', () => {
  it('returns an empty array when allowedTools is undefined or empty', () => {
    expect(parseAllowedTools(undefined)).toEqual([]);
    expect(parseAllowedTools([])).toEqual([]);
  });

  it('rejoins whitespace-split fragments and splits on comma into one entry per tool pattern', () => {
    // Real shape returned by the catalog API: a single "Bash(aws logs tail:*)"
    // pattern arrives split into three array entries by an upstream whitespace split.
    const allowedTools = [
      'Bash(aws',
      'logs',
      'describe-log-groups:*),Bash(aws',
      'logs',
      'tail:*),Bash(*/tools/*/install.sh:*)',
    ];

    expect(parseAllowedTools(allowedTools)).toEqual([
      'Bash(aws logs describe-log-groups:*)',
      'Bash(aws logs tail:*)',
      'Bash(*/tools/*/install.sh:*)',
    ]);
  });

  it('leaves a single, already-clean pattern unchanged', () => {
    expect(parseAllowedTools(['Bash(git diff:*)'])).toEqual(['Bash(git diff:*)']);
  });

  it('splits a single array entry containing multiple comma-separated patterns', () => {
    expect(
      parseAllowedTools([
        'Bash(*/gitlab-branch-manager/scripts/*.sh:*),Bash(*/tools/*/install.sh:*)',
      ]),
    ).toEqual(['Bash(*/gitlab-branch-manager/scripts/*.sh:*)', 'Bash(*/tools/*/install.sh:*)']);
  });
});
