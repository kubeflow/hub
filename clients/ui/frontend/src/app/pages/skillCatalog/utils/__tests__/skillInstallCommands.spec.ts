import {
  buildManualCommand,
  buildNpxCommand,
} from '~/app/pages/skillCatalog/utils/skillInstallCommands';
import type { Skill } from '~/app/skillCatalogTypes';

const baseSkill: Skill = {
  id: '1',
  name: 'diagnosing-bugs',
  repository: 'https://github.com/mattpocock/skills.git',
  path: 'skills/engineering/diagnosing-bugs',
};

describe('buildNpxCommand', () => {
  it('scopes the install to this skill via --skill, rather than the whole repository', () => {
    expect(buildNpxCommand(baseSkill)).toBe(
      'npx skills add https://github.com/mattpocock/skills.git --skill diagnosing-bugs',
    );
  });
});

describe('buildManualCommand', () => {
  it('copies from the local clone directory git clone actually creates, not the remote URL', () => {
    const command = buildManualCommand(baseSkill);
    const [cloneLine, cpLine] = command.split('\n');

    expect(cloneLine).toBe('git clone https://github.com/mattpocock/skills.git');
    // `git clone <url>` creates a local dir named after the repo ("skills"), not the URL.
    expect(cpLine).toBe(
      'cp -r skills/skills/engineering/diagnosing-bugs ~/.claude/skills/diagnosing-bugs',
    );
  });

  it('strips a trailing slash before deriving the clone directory name', () => {
    const command = buildManualCommand({
      ...baseSkill,
      repository: 'https://github.com/mattpocock/skills.git/',
    });
    expect(command.split('\n')[1]).toContain('cp -r skills/');
  });

  it('falls back to the skill name when path is absent', () => {
    const command = buildManualCommand({ ...baseSkill, path: undefined });
    expect(command.split('\n')[1]).toBe(
      'cp -r skills/diagnosing-bugs ~/.claude/skills/diagnosing-bugs',
    );
  });
});
